// Command kpi-env is experiment E0: it captures the environment a batch runs
// in, from inside the measurement container, and prints it as JSON.
//
// Only an allow-list of facts is recorded (coding plan §10): nothing that
// identifies a person or a machine on a network (no hostname, user, path
// outside the container, IP or MAC address, serial number).
package main

import (
	"bufio"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"syscall"
	"time"

	"qchain-kpi/internal/algs"
	"qchain-kpi/internal/buildinfo"
	"qchain-kpi/internal/classical"
	"qchain-kpi/internal/pqc"
	"qchain-kpi/internal/procsetup"
	"qchain-kpi/internal/quality"
	"qchain-kpi/internal/scheme"
)

// Env is the E0 record.
type Env struct {
	Schema       int               `json:"schema"`
	CapturedUTC  string            `json:"captured_utc"`
	Harness      Harness           `json:"harness"`
	Liboqs       Liboqs            `json:"liboqs"`
	OpenSSL      string            `json:"openssl"`
	Glibc        string            `json:"glibc"`
	Kernel       Kernel            `json:"kernel"`
	CPU          CPU               `json:"cpu"`
	Vulns        map[string]string `json:"cpu_vulnerabilities"`
	Clock        Clock             `json:"clock"`
	Cgroup       map[string]string `json:"cgroup"`
	MemTotalKB   uint64            `json:"mem_total_kb"`
	OutFreeBytes uint64            `json:"out_free_bytes"`
	TZ           TZ                `json:"timezone"`
	CryptoRand   Bench             `json:"crypto_rand_read_32_bytes"`
	Algorithms   []AlgInfo         `json:"algorithms"`
	BuildInfo    []FileInfo        `json:"build_info_files"`
}

type Harness struct {
	Commit     string `json:"commit"`
	Go         string `json:"go"`
	GOOS       string `json:"goos"`
	GOARCH     string `json:"goarch"`
	GOMAXPROCS int    `json:"gomaxprocs"`
	NumCPU     int    `json:"num_cpu_affinity"`
	CPUSet     []int  `json:"cpuset_effective"`
}

type Liboqs struct {
	Version       string            `json:"version"`
	EnabledSigs   []string          `json:"enabled_sigs"`
	EnabledKEMs   []string          `json:"enabled_kems"`
	CPUExtensions map[string]bool   `json:"cpu_extensions_runtime"`
	Build         map[string]string `json:"build"` // oqs-build.txt
}

type Kernel struct {
	Release string `json:"release"`
	Version string `json:"version"`
	Machine string `json:"machine"`
}

type CPU struct {
	Logical    int               `json:"logical_cpus"`
	Fields     map[string]string `json:"fields"`            // processor 0 (model, implementer/part, ...)
	Flags      []string          `json:"flags"`             // x86 "flags" or arm64 "Features" of processor 0
	Distinct   map[string]string `json:"distinct_per_cpu"`  // fields whose value differs between processors
	Caches     []string          `json:"caches_cpu0"`       // "L1 Data 64K", ...
}

type Clock struct {
	Source             string  `json:"clocksource"`
	Available          string  `json:"clocksource_available"`
	GetresNS           int64   `json:"clock_getres_monotonic_ns"`
	EffectiveResNS     int64   `json:"effective_resolution_ns"` // smallest non-zero step between consecutive time.Now() calls
	OverheadMedianNS   float64 `json:"time_now_overhead_median_ns"`
	OverheadP99NS      float64 `json:"time_now_overhead_p99_ns"`
	ZeroStepFraction   float64 `json:"zero_step_fraction"`
	Pairs              int     `json:"pairs"`
}

type TZ struct {
	Env        string `json:"tz_env"`
	Local      string `json:"local"`
	Zone       string `json:"zone_now"`
	OffsetSecs int    `json:"offset_s"`
}

type Bench struct {
	MedianNS float64 `json:"median_ns"`
	P99NS    float64 `json:"p99_ns"`
	Samples  int     `json:"samples"`
}

type AlgInfo struct {
	ID       string       `json:"id"`
	Track    string       `json:"track"`
	Family   string       `json:"family"`
	Tier     string       `json:"tier"`
	Category int          `json:"nist_category"`
	OQSName  string       `json:"oqs_name,omitempty"`
	Spec     string       `json:"spec"`
	Sizes    scheme.Sizes `json:"sizes"`
}

type FileInfo struct {
	Name   string `json:"name"`
	Bytes  int64  `json:"bytes"`
	SHA256 string `json:"sha256"`
}

func main() {
	buildInfoDir := flag.String("build-info", "/opt/kpi/build-info", "directory of build records")
	outDir := flag.String("out", "/out", "results volume (for free space)")
	flag.Parse()
	if err := procsetup.GoGate(); err != nil {
		fatal(err)
	}
	if err := pqc.CheckLibrary(); err != nil {
		fatal(err)
	}
	cpus, _ := quality.CPUSet()
	e := Env{
		Schema:      1,
		CapturedUTC: time.Now().UTC().Format(time.RFC3339Nano),
		Harness: Harness{Commit: buildinfo.HarnessCommit, Go: runtime.Version(), GOOS: runtime.GOOS, GOARCH: runtime.GOARCH,
			GOMAXPROCS: runtime.GOMAXPROCS(0), NumCPU: runtime.NumCPU(), CPUSet: cpus},
		OpenSSL: classical.OpenSSLVersion(),
		Glibc:   glibcVersion(),
		Kernel:  kernel(),
		CPU:     cpuInfo(),
		Vulns:   dirFiles("/sys/devices/system/cpu/vulnerabilities"),
		Cgroup:  cgroup(),
		TZ:      tz(),
	}
	sigs, kems := pqc.EnabledAlgorithms()
	e.Liboqs = Liboqs{Version: pqc.Version(), EnabledSigs: sigs, EnabledKEMs: kems,
		CPUExtensions: pqc.CPUExtensions(), Build: keyValues(filepath.Join(*buildInfoDir, "oqs-build.txt"))}
	e.Clock = measureTimer()
	e.CryptoRand = benchCryptoRand()
	e.MemTotalKB = memTotalKB()
	var st syscall.Statfs_t
	if syscall.Statfs(*outDir, &st) == nil {
		e.OutFreeBytes = st.Bavail * uint64(st.Bsize)
	}
	for _, a := range algs.All() {
		info := AlgInfo{ID: a.ID, Track: string(a.Track), Family: a.Family, Tier: a.Tier, Category: a.Category, OQSName: a.OQSName, Spec: a.Spec}
		if s, err := sizesOf(a); err == nil {
			info.Sizes = s
		} else {
			fatal(err)
		}
		e.Algorithms = append(e.Algorithms, info)
	}
	e.BuildInfo = fileList(*buildInfoDir)
	b, err := json.MarshalIndent(e, "", "  ")
	if err != nil {
		fatal(err)
	}
	fmt.Println(string(b))
}

func fatal(err error) {
	fmt.Fprintf(os.Stderr, "kpi-env: %v\n", err)
	os.Exit(1)
}

func sizesOf(a algs.Alg) (scheme.Sizes, error) {
	switch {
	case a.Track == algs.Sig && a.Classical():
		s, err := classical.NewSignature(a)
		if err != nil {
			return scheme.Sizes{}, err
		}
		return s.Sizes(), nil
	case a.Track == algs.Sig:
		s, err := pqc.NewSignature(a)
		if err != nil {
			return scheme.Sizes{}, err
		}
		return s.Sizes(), nil
	case a.Classical():
		k, err := classical.NewKEM(a)
		if err != nil {
			return scheme.Sizes{}, err
		}
		return k.Sizes(), nil
	default:
		k, err := pqc.NewKEM(a)
		if err != nil {
			return scheme.Sizes{}, err
		}
		return k.Sizes(), nil
	}
}

func readTrim(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

func kernel() Kernel {
	k := Kernel{Release: readTrim("/proc/sys/kernel/osrelease"), Version: readTrim("/proc/sys/kernel/version")}
	var u syscall.Utsname
	if syscall.Uname(&u) == nil {
		var sb strings.Builder
		for _, c := range u.Machine {
			if c == 0 {
				break
			}
			sb.WriteByte(byte(c))
		}
		k.Machine = sb.String() // only the machine field; never the node name
	}
	return k
}

// cpuInfo parses /proc/cpuinfo.
func cpuInfo() CPU {
	f, err := os.Open("/proc/cpuinfo")
	if err != nil {
		return CPU{}
	}
	defer f.Close()
	var procs []map[string]string
	cur := map[string]string{}
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	for sc.Scan() {
		line := sc.Text()
		if strings.TrimSpace(line) == "" {
			if len(cur) > 0 {
				procs = append(procs, cur)
				cur = map[string]string{}
			}
			continue
		}
		k, v, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		cur[strings.TrimSpace(k)] = strings.TrimSpace(v)
	}
	if len(cur) > 0 {
		procs = append(procs, cur)
	}
	c := CPU{Fields: map[string]string{}, Distinct: map[string]string{}}
	for _, p := range procs {
		if _, ok := p["processor"]; ok {
			c.Logical++
		}
	}
	if len(procs) == 0 {
		return c
	}
	// Fields that vary per reading or per core and carry no configuration
	// information are skipped.
	skip := map[string]bool{"processor": true, "cpu MHz": true, "core id": true, "apicid": true, "initial apicid": true}
	p0 := procs[0]
	for k, v := range p0 {
		if skip[k] {
			continue
		}
		switch k {
		case "flags", "Features":
			c.Flags = strings.Fields(v)
		default:
			c.Fields[k] = v
		}
	}
	// Machine-wide lines (arm64 kernels print "Hardware" etc. after the last processor).
	for _, p := range procs[1:] {
		if _, ok := p["processor"]; !ok {
			for k, v := range p {
				c.Fields[k] = v
			}
		}
	}
	for _, p := range procs {
		if _, ok := p["processor"]; !ok {
			continue
		}
		for k, v := range p {
			if skip[k] || k == "flags" || k == "Features" || k == "bogomips" || k == "BogoMIPS" {
				continue
			}
			if v != p0[k] {
				c.Distinct[k] = "differs between processors"
			}
		}
	}
	for i := 0; i < 8; i++ {
		d := fmt.Sprintf("/sys/devices/system/cpu/cpu0/cache/index%d", i)
		lv := readTrim(d + "/level")
		if lv == "" {
			continue
		}
		c.Caches = append(c.Caches, fmt.Sprintf("L%s %s %s", lv, readTrim(d+"/type"), readTrim(d+"/size")))
	}
	return c
}

func dirFiles(dir string) map[string]string {
	out := map[string]string{}
	ents, err := os.ReadDir(dir)
	if err != nil {
		return out
	}
	for _, e := range ents {
		if !e.IsDir() {
			out[e.Name()] = readTrim(filepath.Join(dir, e.Name()))
		}
	}
	return out
}

func cgroup() map[string]string {
	out := map[string]string{}
	if _, err := os.Stat("/sys/fs/cgroup/cgroup.controllers"); err == nil {
		out["version"] = "2"
	} else {
		out["version"] = "1 or none"
	}
	for _, f := range []string{"cpu.max", "cpu.weight", "cpuset.cpus.effective", "cpuset.mems.effective",
		"memory.max", "memory.high", "memory.swap.max", "pids.max"} {
		if v := readTrim("/sys/fs/cgroup/" + f); v != "" {
			out[f] = v
		}
	}
	if _, err := os.Stat("/sys/fs/cgroup/memory.peak"); err == nil {
		out["memory.peak"] = "available"
	} else {
		out["memory.peak"] = "not available"
	}
	out["clocksource"] = readTrim("/sys/devices/system/clocksource/clocksource0/current_clocksource")
	return out
}

func tz() TZ {
	name, off := time.Now().Zone()
	return TZ{Env: os.Getenv("TZ"), Local: time.Local.String(), Zone: name, OffsetSecs: off}
}

func memTotalKB() uint64 {
	for _, l := range strings.Split(readTrim("/proc/meminfo"), "\n") {
		if strings.HasPrefix(l, "MemTotal:") {
			var kb uint64
			fmt.Sscanf(strings.TrimSpace(strings.TrimPrefix(l, "MemTotal:")), "%d", &kb)
			return kb
		}
	}
	return 0
}

// measureTimer times back-to-back time.Now() calls, the clock reads that
// bracket every timed sample.
func measureTimer() Clock {
	const n = 2_000_000
	d := make([]int64, n)
	prev := time.Now()
	for i := range d {
		t := time.Now()
		d[i] = int64(t.Sub(prev))
		prev = t
	}
	zero := 0
	minNZ := int64(math.MaxInt64)
	for _, x := range d {
		if x == 0 {
			zero++
		} else if x > 0 && x < minNZ {
			minNZ = x
		}
	}
	sort.Slice(d, func(i, j int) bool { return d[i] < d[j] })
	if minNZ == math.MaxInt64 {
		minNZ = -1
	}
	return Clock{
		Source:           readTrim("/sys/devices/system/clocksource/clocksource0/current_clocksource"),
		Available:        readTrim("/sys/devices/system/clocksource/clocksource0/available_clocksource"),
		GetresNS:         clockResNS(),
		EffectiveResNS:   minNZ,
		OverheadMedianNS: float64(d[n/2]),
		OverheadP99NS:    float64(d[n*99/100]),
		ZeroStepFraction: float64(zero) / n,
		Pairs:            n,
	}
}

// benchCryptoRand times crypto/rand.Read of 32 bytes (Go 1.24 uses the
// getrandom vDSO only on Linux 6.11 and later; E0 records the kernel).
func benchCryptoRand() Bench {
	buf := make([]byte, 32)
	for i := 0; i < 1000; i++ {
		rand.Read(buf)
	}
	const n = 10000
	d := make([]float64, n)
	for i := range d {
		t := time.Now()
		rand.Read(buf)
		d[i] = float64(time.Since(t))
	}
	sort.Float64s(d)
	return Bench{MedianNS: d[n/2], P99NS: d[n*99/100], Samples: n}
}

func keyValues(path string) map[string]string {
	out := map[string]string{}
	for _, l := range strings.Split(readTrim(path), "\n") {
		if k, v, ok := strings.Cut(l, "="); ok {
			out[strings.TrimSpace(k)] = strings.TrimSpace(v)
		}
	}
	return out
}

func fileList(dir string) []FileInfo {
	var out []FileInfo
	ents, err := os.ReadDir(dir)
	if err != nil {
		return out
	}
	for _, e := range ents {
		if e.IsDir() {
			continue
		}
		b, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		s := sha256.Sum256(b)
		out = append(out, FileInfo{Name: e.Name(), Bytes: int64(len(b)), SHA256: hex.EncodeToString(s[:])})
	}
	return out
}
