# Machine `mac-m1`: Apple M1 MacBook Air (ARM)

| | |
|---|---|
| Model | MacBook Air (M1, 2020), MacBookAir10,1; **fanless**. Runs on a laptop cooling pad with a USB fan, on mains power, lid open, on a hard surface. |
| CPU | Apple M1: 4 performance + 4 efficiency cores. The P-cluster shares 12 MB of L2, the E-cluster 4 MB. 128-byte cache lines, 16 KB pages. |
| Instruction sets | NEON (fp16), AES, PMULL, SHA-256, SHA-512, SHA-3, DotProd |
| Memory | 8 GB |
| OS | macOS 27.0.1 (26A434). Not updated during the study. |
| Container runtime | Colima 0.10.3 with Apple's Virtualization.framework (`--vm-type vz`): a Linux VM with 4 CPUs and 4 GiB, no host mounts. Docker Engine 29.5.2 inside the VM, Docker CLI 29.9.0. |
| Container | Pinned to CPU 1 of the VM (`--cpuset-cpus=1`, `GOMAXPROCS=1`), no CPU-time quota, 3 GB memory without swap, no network |
| Image | Built natively for arm64 |
| Timer (E0) | clocksource `arch_sys_counter` (24 MHz): effective resolution 41 ns, `time.Now` overhead about 42 ns |

**Facts measured inside the container:**
- The Virtualization.framework hides the CPU part number (`CPU part 0x000`), so the core type cannot be read from inside.
- Every canary is therefore classed against native references, measured at the start of each session: P-core about 2.5 M SHA3-256 hashes/s, E-core about 0.58 M.

**Limitations:**
- macOS cannot pin a thread to a core type, so under Docker the core type is checked, not controlled.
- The Mac is fanless; thermal pressure is logged every 5 s.
