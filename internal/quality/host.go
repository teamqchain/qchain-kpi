package quality

// Host-monitor samples. The host launcher's monitor writes one CSV row every
// ~5 s on the HOST and delivers them into the run volume every ~60 s with
// `docker cp`, as chunk files monitor/host-<seq>.csv followed by a marker
// monitor/host-<seq>.ok (a chunk is read only once its marker exists, so a
// half-copied file is never read). No process runs inside the measurement
// container for this.
//
// Row format (header line "utc_ns,other_pct,throttle,on_ac,alert,detail"):
//   utc_ns     host wall clock, Unix nanoseconds (UTC)
//   other_pct  host CPU used by anything except the benchmark VM process and
//              the monitor itself, % of host capacity (-1 if not measurable)
//   throttle   1 if a thermal/power limit was active (Mac thermal pressure >= 1
//              or a pmset warning; Windows "% Performance Limit" < 100), 0 if
//              not, -1 if not measurable
//   on_ac      1 on mains power, 0 on battery, -1 not applicable
//   alert      1 if the monitor saw a forbidden condition (e.g. a non-kpi
//              container or a QChain service running on the VM), else 0
//   detail     free text for audit (raw readings), never parsed for flags

import (
	"encoding/csv"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// HostSample is one parsed row.
type HostSample struct {
	UnixNS   int64
	OtherPct float64
	Throttle int
	OnAC     int
	Alert    int
	Detail   string
}

// ReadHostChunks returns every sample from completed chunks in dir, sorted by
// time. Rows that do not parse are skipped (and counted).
func ReadHostChunks(dir string) ([]HostSample, int, error) {
	oks, err := filepath.Glob(filepath.Join(dir, "host-*.ok"))
	if err != nil {
		return nil, 0, err
	}
	var out []HostSample
	bad := 0
	for _, ok := range oks {
		chunk := strings.TrimSuffix(ok, ".ok") + ".csv"
		f, err := os.Open(chunk)
		if err != nil {
			continue
		}
		r := csv.NewReader(f)
		r.FieldsPerRecord = -1
		rows, err := r.ReadAll()
		f.Close()
		if err != nil {
			bad++
			continue
		}
		for _, row := range rows {
			if len(row) < 5 || row[0] == "utc_ns" {
				continue
			}
			t, err1 := strconv.ParseInt(row[0], 10, 64)
			o, err2 := strconv.ParseFloat(row[1], 64)
			th, err3 := strconv.Atoi(row[2])
			ac, err4 := strconv.Atoi(row[3])
			al, err5 := strconv.Atoi(row[4])
			if err1 != nil || err2 != nil || err3 != nil || err4 != nil || err5 != nil {
				bad++
				continue
			}
			s := HostSample{UnixNS: t, OtherPct: o, Throttle: th, OnAC: ac, Alert: al}
			if len(row) > 5 {
				s.Detail = row[5]
			}
			out = append(out, s)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].UnixNS < out[j].UnixNS })
	return out, bad, nil
}

// HostVerdict summarises the host samples that overlap one block.
type HostVerdict struct {
	Samples     int     `json:"samples"`
	MaxOtherPct float64 `json:"max_other_pct"` // -1 if not measurable
	Throttled   bool    `json:"throttled"`
	OnBattery   bool    `json:"on_battery"`
	Alert       bool    `json:"alert"`
	Covered     bool    `json:"covered"` // a sample at or after the block end exists
}

// HostOver evaluates the host samples overlapping [startNS, endNS] (container
// clock). offsetNS = host clock - container clock. A sample covers the
// interval before it (one sampling period, periodNS).
func HostOver(samples []HostSample, startNS, endNS, offsetNS, periodNS int64) HostVerdict {
	v := HostVerdict{MaxOtherPct: -1}
	for _, s := range samples {
		t := s.UnixNS - offsetNS // to container clock
		if t >= endNS {
			v.Covered = true
		}
		if t < startNS || t-periodNS > endNS {
			continue
		}
		v.Samples++
		if s.OtherPct > v.MaxOtherPct {
			v.MaxOtherPct = s.OtherPct
		}
		if s.Throttle == 1 {
			v.Throttled = true
		}
		if s.OnAC == 0 {
			v.OnBattery = true
		}
		if s.Alert == 1 {
			v.Alert = true
		}
	}
	return v
}
