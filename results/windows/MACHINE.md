# Machine `win-ryzen`: HP laptop, AMD Ryzen 5 7520U (x86-64 with AVX2)

| | |
|---|---|
| Model | HP Laptop 15-fc0xxx. Whether it has a fan is unconfirmed; thermal state is logged. Runs on mains power, lid open, on a hard surface. |
| CPU | AMD Ryzen 5 7520U (Zen 2, "Mendocino", 15 W): 4 cores / 8 threads, 2.8 GHz base. L2 2 MB, L3 4 MB. All cores are the same type. |
| Instruction sets | AVX and AVX2, no AVX-512 |
| Memory | 8 GB LPDDR5-5500 (7.2 GB usable) |
| OS | Windows 11 Home 25H2 (build 26200). Virtualisation-based security and memory integrity are on (recorded, not changed). |
| Container runtime | Docker Desktop on WSL2 (WSL 2.7.12, kernel 6.18). Docker Engine 29.7.2 sees 8 CPUs and 3.45 GB. |
| Container | Pinned to CPU 1 of the WSL2 VM (`--cpuset-cpus=1`, `GOMAXPROCS=1`), no CPU-time quota, 3 GB memory without swap, no network |
| Image | Built natively for amd64 |

**Limitations:** `--cpuset-cpus` pins to a WSL2 virtual CPU, not a physical core.

The launcher (`run.ps1`, Windows PowerShell 5.1) and the Windows host monitor arrive in phase 1b.
