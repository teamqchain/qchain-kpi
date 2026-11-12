# E01a  Signature schemes on the Mac M1 (Apple M1, ARM with NEON): latency per call (µs)

Median over five repetitions (headline attempts) with its 95 % bootstrap interval over the repetitions; p99, mean and coefficient of variation of the individual calls; the first call of a fresh process.

| Family | Algorithm | Operation | Median | 95 % CI | p99 | Mean | CoV | First call | Repetitions | Attempts |
|---|---|---|---:|---|---:|---:|---:|---:|---:|---:|
| ML-DSA | ml-dsa-44 | key generation | 30.2 | 30.2–41.4 | 36.4 | 30.7 | 0.07 | 926 | 5 | 5 |
| ML-DSA | ml-dsa-44 | signing | 60.2 | 59.2–60.7 | 218 | 74.0 | 0.55 | 967 | 5 | 5 |
| ML-DSA | ml-dsa-44 | verification | 28.4 | 27.9–42.2 | 32.6 | 28.7 | 0.05 | 49.8 | 5 | 5 |
| ML-DSA | ml-dsa-65 | key generation | 53.0 | 52.2–53.3 | 67.3 | 54.2 | 0.06 | 938 | 5 | 5 |
| ML-DSA | ml-dsa-65 | signing | 95.5 | 95.3–98.1 | 328 | 113 | 0.52 | 997 | 5 | 5 |
| ML-DSA | ml-dsa-65 | verification | 47.6 | 47.6–48.7 | 53.8 | 48.6 | 0.05 | 75.3 | 5 | 5 |
| ML-DSA | ml-dsa-87 | key generation | 90.9 | 89.1–91.2 | 108 | 92.3 | 0.04 | 1,033 | 5 | 5 |
| ML-DSA | ml-dsa-87 | signing | 140 | 138–141 | 388 | 158 | 0.41 | 1,037 | 5 | 5 |
| ML-DSA | ml-dsa-87 | verification | 83.5 | 83.2–84.7 | 93.3 | 84.7 | 0.04 | 110 | 5 | 5 |
| Falcon | falcon-1024 | key generation | 14,858 | 14,772–14,867 | 32,292 | 16,278 | 0.25 | 15,942 | 5 | 5 |
| Falcon | falcon-1024 | signing | 290 | 287–291 | 317 | 291 | 0.02 | 1,184 | 5 | 5 |
| Falcon | falcon-1024 | verification | 42.6 | 41.9–48.8 | 48.6 | 43.3 | 0.05 | 66.6 | 5 | 5 |
| Falcon | falcon-512 | key generation | 4,802 | 4,801–4,809 | 9,701 | 5,165 | 0.24 | 5,335 | 5 | 5 |
| Falcon | falcon-512 | signing | 145 | 145–146 | 167 | 146 | 0.04 | 1,058 | 5 | 5 |
| Falcon | falcon-512 | verification | 21.7 | 21.6–21.9 | 24.2 | 21.9 | 0.04 | 35.6 | 5 | 5 |
| Falcon | falcon-padded-1024 | key generation | 14,814 | 14,803–14,885 | 32,282 | 16,227 | 0.25 | 16,504 | 5 | 5 |
| Falcon | falcon-padded-1024 | signing | 287 | 287–292 | 316 | 289 | 0.03 | 1,170 | 5 | 5 |
| Falcon | falcon-padded-1024 | verification | 41.9 | 41.8–42.6 | 47.9 | 42.7 | 0.05 | 57.0 | 5 | 5 |
| Falcon | falcon-padded-512 | key generation | 4,803 | 4,799–4,806 | 9,582 | 5,153 | 0.24 | 5,494 | 5 | 5 |
| Falcon | falcon-padded-512 | signing | 145 | 144–145 | 165 | 146 | 0.04 | 1,052 | 5 | 5 |
| Falcon | falcon-padded-512 | verification | 21.8 | 21.6–22.1 | 24.2 | 22.0 | 0.05 | 34.6 | 5 | 5 |
| SLH-DSA | slh-dsa-sha2-128f | key generation | 888 | 888–888 | 940 | 893 | 0.01 | 1,804 | 5 | 5 |
| SLH-DSA | slh-dsa-sha2-128f | signing | 20,748 | 20,745–20,756 | 21,516 | 20,785 | 0.01 | 21,741 | 5 | 5 |
| SLH-DSA | slh-dsa-sha2-128f | verification | 1,252 | 1,251–1,256 | 1,345 | 1,255 | 0.03 | 1,263 | 5 | 5 |
| SLH-DSA | slh-dsa-sha2-128s | key generation | 56,810 | 56,804–56,846 | 58,816 | 56,888 | 0.01 | 57,753 | 5 | 5 |
| SLH-DSA | slh-dsa-sha2-128s | signing | 431,348 | 431,297–431,728 | 436,269 | 431,744 | 0.00 | 432,702 | 5 | 5 |
| SLH-DSA | slh-dsa-sha2-128s | verification | 428 | 426–434 | 474 | 428 | 0.04 | 444 | 5 | 5 |
| SLH-DSA | slh-dsa-sha2-192f | key generation | 1,298 | 1,298–1,298 | 1,372 | 1,300 | 0.01 | 2,216 | 5 | 5 |
| SLH-DSA | slh-dsa-sha2-192f | signing | 33,802 | 33,792–33,839 | 34,964 | 33,850 | 0.01 | 34,793 | 5 | 5 |
| SLH-DSA | slh-dsa-sha2-192f | verification | 1,823 | 1,823–1,830 | 1,933 | 1,825 | 0.02 | 1,903 | 5 | 5 |
| SLH-DSA | slh-dsa-sha2-192s | key generation | 82,750 | 82,707–82,755 | 85,449 | 82,848 | 0.01 | 83,642 | 5 | 5 |
| SLH-DSA | slh-dsa-sha2-192s | signing | 757,581 | 757,227–758,273 | 762,614 | 758,088 | 0.00 | 759,151 | 5 | 5 |
| SLH-DSA | slh-dsa-sha2-192s | verification | 633 | 628–635 | 689 | 635 | 0.03 | 636 | 5 | 5 |
| SLH-DSA | slh-dsa-sha2-256f | key generation | 3,423 | 3,423–3,426 | 3,620 | 3,432 | 0.01 | 4,433 | 5 | 5 |
| SLH-DSA | slh-dsa-sha2-256f | signing | 69,469 | 69,448–69,513 | 71,641 | 69,564 | 0.01 | 70,385 | 5 | 5 |
| SLH-DSA | slh-dsa-sha2-256f | verification | 1,870 | 1,865–1,873 | 1,984 | 1,873 | 0.02 | 1,924 | 5 | 5 |
| SLH-DSA | slh-dsa-sha2-256s | key generation | 54,656 | 54,647–54,665 | 56,344 | 54,723 | 0.01 | 55,618 | 5 | 5 |
| SLH-DSA | slh-dsa-sha2-256s | signing | 667,755 | 667,576–670,595 | 673,986 | 668,341 | 0.00 | 668,942 | 5 | 5 |
| SLH-DSA | slh-dsa-sha2-256s | verification | 928 | 922–930 | 1,003 | 931 | 0.03 | 941 | 5 | 5 |
| SLH-DSA | slh-dsa-shake-128f | key generation | 1,120 | 1,112–1,122 | 1,185 | 1,121 | 0.02 | 2,053 | 5 | 5 |
| SLH-DSA | slh-dsa-shake-128f | signing | 26,152 | 26,078–26,529 | 27,018 | 26,274 | 0.01 | 27,076 | 5 | 5 |
| SLH-DSA | slh-dsa-shake-128f | verification | 1,558 | 1,557–2,160 | 1,677 | 1,561 | 0.03 | 1,572 | 5 | 5 |
| SLH-DSA | slh-dsa-shake-128s | key generation | 70,822 | 70,763–71,638 | 73,295 | 71,213 | 0.01 | 71,921 | 5 | 5 |
| SLH-DSA | slh-dsa-shake-128s | signing | 552,782 | 549,987–591,777 | 556,830 | 550,067 | 0.01 | 554,052 | 5 | 5 |
| SLH-DSA | slh-dsa-shake-128s | verification | 539 | 536–541 | 597 | 541 | 0.04 | 560 | 5 | 5 |
| SLH-DSA | slh-dsa-shake-192f | key generation | 1,641 | 1,639–1,645 | 1,731 | 1,643 | 0.02 | 2,610 | 5 | 5 |
| SLH-DSA | slh-dsa-shake-192f | signing | 43,220 | 43,121–43,458 | 44,805 | 43,387 | 0.01 | 44,146 | 5 | 5 |
| SLH-DSA | slh-dsa-shake-192f | verification | 2,285 | 2,279–2,413 | 2,439 | 2,288 | 0.02 | 2,340 | 5 | 5 |
| SLH-DSA | slh-dsa-shake-192s | key generation | 104,123 | 103,945–109,687 | 107,382 | 104,694 | 0.01 | 104,835 | 5 | 5 |
| SLH-DSA | slh-dsa-shake-192s | signing | 982,146 | 972,081–997,933 | 996,186 | 978,136 | 0.01 | 971,472 | 5 | 5 |
| SLH-DSA | slh-dsa-shake-192s | verification | 786 | 784–795 | 854 | 787 | 0.03 | 785 | 5 | 5 |
| SLH-DSA | slh-dsa-shake-256f | key generation | 4,345 | 4,321–4,667 | 4,562 | 4,346 | 0.02 | 5,259 | 5 | 5 |
| SLH-DSA | slh-dsa-shake-256f | signing | 89,331 | 89,138–90,444 | 92,260 | 89,703 | 0.01 | 90,101 | 5 | 5 |
| SLH-DSA | slh-dsa-shake-256f | verification | 2,335 | 2,331–2,338 | 2,489 | 2,338 | 0.03 | 2,370 | 5 | 5 |
| SLH-DSA | slh-dsa-shake-256s | key generation | 68,979 | 68,689–69,578 | 70,968 | 69,295 | 0.01 | 69,900 | 5 | 5 |
| SLH-DSA | slh-dsa-shake-256s | signing | 886,612 | 874,533–1,364,766 | 896,238 | 884,186 | 0.01 | 875,281 | 5 | 5 |
| SLH-DSA | slh-dsa-shake-256s | verification | 1,163 | 1,158–1,198 | 1,260 | 1,163 | 0.03 | 1,147 | 5 | 5 |
| classical | ecdsa-p256 | key generation | 25.9 | 25.9–26.4 | 31.6 | 26.3 | 0.06 | 400 | 5 | 5 |
| classical | ecdsa-p256 | signing | 19.9 | 19.9–20.0 | 25.4 | 20.5 | 0.07 | 378 | 5 | 5 |
| classical | ecdsa-p256 | verification | 54.5 | 54.5–54.7 | 60.8 | 56.0 | 0.04 | 92.2 | 5 | 5 |
| classical | ed25519 | key generation | 32.1 | 32.1–32.7 | 37.7 | 32.7 | 0.05 | 393 | 5 | 5 |
| classical | ed25519 | signing | 30.3 | 30.3–30.4 | 38.9 | 31.1 | 0.08 | 52.7 | 5 | 5 |
| classical | ed25519 | verification | 81.6 | 81.5–81.8 | 91.2 | 82.6 | 0.04 | 109 | 5 | 5 |
| classical | rsa3072-pss | key generation | 401,964 | 381,262–431,375 | 1,231,200 | 455,949 | 0.54 | 544,980 | 5 | 5 |
| classical | rsa3072-pss | signing | 1,605 | 1,601–1,629 | 2,102 | 1,626 | 0.05 | 2,503 | 5 | 5 |
| classical | rsa3072-pss | verification | 33.0 | 33.0–33.6 | 38.3 | 33.7 | 0.05 | 84.3 | 5 | 5 |

Generated by `analysis/paper_tables.py` from `results/combined/analysis_tables/T2_E1_primitive_latency.csv`.
