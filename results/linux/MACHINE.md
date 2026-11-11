# Machine `vm-x86v2`: virtualised x86 server (tested environment)

The paper describes this machine only as a tested environment: a virtualised x86 server. It is never described as QChain's deployment or production environment.

| | |
|---|---|
| Guest CPU | Proxmox CPU type `x86-64-v2-AES` (Proxmox VE 8's default), reported as "QEMU Virtual CPU version 2.5+". Flags include AES and SSE4.2, but there is **no AVX, AVX2, BMI2, PCLMULQDQ or AVX-512**, so liboqs runs its portable C code. |
| vCPUs | 2 (1 socket × 2 cores), **unpinned, time-shared with other VMs** on the host |
| Memory | 30 GiB |
| Host | Intel Xeon Silver 4510 (Emerald Rapids), KVM/QEMU under Proxmox VE 8 |
| Timers | clocksource `kvm-clock`; no constant TSC, no PMU, no cpufreq, no RAPL |
| Container runtime | Docker Engine (Linux; the container shares the VM's kernel) |
| Container | Pinned to vCPU 1 (`--cpuset-cpus=1`, `GOMAXPROCS=1`), no CPU-time quota, 4 GB memory without swap, no network. The host monitor runs on vCPU 0. |
| Image | Built natively for amd64 |

- **Steal time** on the pinned vCPU is measured for every block. CPU sharing is treated as a studied factor.
- **Exclusive use.** QChain's services are stopped on this machine for the whole study.

The Linux launcher (`scripts/run-linux.sh`) and host monitor (`scripts/monitor-linux.sh`) ran every VM run from 25 Oct to 31 Oct 2026; the results are analysed in `docs/results-vm.md`.
