# Machine `win-ryzen`: HP laptop, AMD Ryzen 5 7520U (x86-64 with AVX2)

| | |
|---|---|
| Model | HP Laptop 15-fc0xxx, with a fan; thermal state is logged. Runs on mains power, lid open, on a hard surface. |
| CPU | AMD Ryzen 5 7520U (Zen 2, "Mendocino", 15 W): 4 cores / 8 threads, 2.8 GHz base. L2 2 MB, L3 4 MB. All cores are the same type. |
| Instruction sets | AVX and AVX2, no AVX-512 |
| Memory | 8 GB LPDDR5-5500 (7.2 GB usable) |
| OS | Windows 11 Home 25H2 (build 26200). Virtualisation-based security and memory integrity are on (recorded, not changed). |
| Container runtime | Docker Desktop on WSL2 (WSL 2.7.12, kernel 6.18). Docker Engine 29.7.2 sees 8 CPUs and 3.45 GB. |
| Container | Pinned to CPU 1 of the WSL2 VM (`--cpuset-cpus=1`, `GOMAXPROCS=1`), no CPU-time quota, 3 GB memory without swap, no network |
| Image | Built natively for amd64 |

**Limitations:** `--cpuset-cpus` pins to a WSL2 virtual CPU, not a physical core. Under all-core load the CPU reaches its thermal limit after about a minute (measured 6 Nov 2026 with a plain busy loop: 99 C, then a short frequency drop), so E12 with 8 workers gives short-burst figures, not sustained ones. One busy thread settled at about 71 C with no limit active.

The launchers are `scripts/run-windows.ps1` (one plan) and `scripts/run-windows-series.ps1` (several in a row), for Windows PowerShell 5.1, with the host monitor `scripts/monitor-windows.ps1`.
