# E01a  Signature schemes on the Laptop (AMD Ryzen 5 7520U, x86 with AVX2): latency per call (µs)

Median over five repetitions (headline attempts) with its 95 % bootstrap interval over the repetitions; p99, mean and coefficient of variation of the individual calls; the first call of a fresh process.

| Family | Algorithm | Operation | Median | 95 % CI | p99 | Mean | CoV | First call | Repetitions | Attempts |
|---|---|---|---:|---|---:|---:|---:|---:|---:|---:|
| ML-DSA | ml-dsa-44 | key generation | 32.0 | 31.8–32.1 | 62.0 | 34.1 | 0.22 | 1,768 | 5 | 5 |
| ML-DSA | ml-dsa-44 | signing | 79.2 | 78.6–81.0 | 313 | 98.1 | 0.61 | 1,944 | 5 | 5 |
| ML-DSA | ml-dsa-44 | verification | 32.0 | 31.9–32.3 | 59.4 | 34.2 | 0.26 | 118 | 5 | 5 |
| ML-DSA | ml-dsa-65 | key generation | 55.0 | 54.9–55.1 | 95.3 | 59.0 | 0.16 | 1,972 | 5 | 5 |
| ML-DSA | ml-dsa-65 | signing | 125 | 124–126 | 494 | 154 | 0.61 | 1,948 | 5 | 5 |
| ML-DSA | ml-dsa-65 | verification | 53.1 | 52.9–53.3 | 91.7 | 57.4 | 0.16 | 185 | 5 | 5 |
| ML-DSA | ml-dsa-87 | key generation | 85.2 | 85.0–88.1 | 141 | 91.3 | 0.16 | 1,910 | 5 | 5 |
| ML-DSA | ml-dsa-87 | signing | 153 | 152–154 | 497 | 181 | 0.49 | 2,082 | 5 | 5 |
| ML-DSA | ml-dsa-87 | verification | 81.3 | 80.6–81.9 | 138 | 86.9 | 0.15 | 234 | 5 | 5 |
| Falcon | falcon-1024 | key generation | 15,838 | 15,775–15,895 | 35,909 | 17,464 | 0.27 | 17,941 | 5 | 5 |
| Falcon | falcon-1024 | signing | 417 | 415–422 | 528 | 427 | 0.07 | 2,267 | 5 | 5 |
| Falcon | falcon-1024 | verification | 81.1 | 80.3–124 | 131 | 86.2 | 0.13 | 173 | 5 | 5 |
| Falcon | falcon-512 | key generation | 5,620 | 5,609–5,625 | 13,277 | 6,201 | 0.32 | 6,798 | 5 | 5 |
| Falcon | falcon-512 | signing | 210 | 207–211 | 302 | 218 | 0.10 | 1,924 | 5 | 5 |
| Falcon | falcon-512 | verification | 41.0 | 40.6–41.1 | 71.5 | 44.0 | 0.19 | 88.5 | 5 | 5 |
| Falcon | falcon-padded-1024 | key generation | 15,798 | 15,728–15,931 | 35,756 | 17,475 | 0.28 | 16,269 | 5 | 5 |
| Falcon | falcon-padded-1024 | signing | 419 | 415–421 | 551 | 432 | 0.07 | 2,289 | 5 | 5 |
| Falcon | falcon-padded-1024 | verification | 80.7 | 80.4–81.0 | 128 | 85.5 | 0.14 | 147 | 5 | 5 |
| Falcon | falcon-padded-512 | key generation | 5,614 | 5,579–5,625 | 13,312 | 6,195 | 0.32 | 7,635 | 5 | 5 |
| Falcon | falcon-padded-512 | signing | 208 | 207–211 | 298 | 215 | 0.09 | 1,961 | 5 | 5 |
| Falcon | falcon-padded-512 | verification | 41.0 | 40.9–41.3 | 71.9 | 44.0 | 0.21 | 87.5 | 5 | 5 |
| SLH-DSA | slh-dsa-sha2-128f | key generation | 890 | 880–898 | 1,111 | 912 | 0.07 | 2,678 | 5 | 5 |
| SLH-DSA | slh-dsa-sha2-128f | signing | 20,817 | 20,759–20,849 | 22,173 | 20,913 | 0.02 | 22,710 | 5 | 5 |
| SLH-DSA | slh-dsa-sha2-128f | verification | 1,264 | 1,254–1,268 | 1,463 | 1,273 | 0.05 | 1,323 | 5 | 5 |
| SLH-DSA | slh-dsa-sha2-128s | key generation | 55,977 | 55,863–56,258 | 60,101 | 56,168 | 0.02 | 57,999 | 5 | 5 |
| SLH-DSA | slh-dsa-sha2-128s | signing | 429,945 | 428,588–433,062 | 451,001 | 431,066 | 0.01 | 433,792 | 5 | 5 |
| SLH-DSA | slh-dsa-sha2-128s | verification | 438 | 434–438 | 560 | 445 | 0.08 | 508 | 5 | 5 |
| SLH-DSA | slh-dsa-sha2-192f | key generation | 1,286 | 1,284–1,289 | 1,495 | 1,301 | 0.05 | 3,234 | 5 | 5 |
| SLH-DSA | slh-dsa-sha2-192f | signing | 34,347 | 34,192–34,381 | 37,440 | 34,507 | 0.02 | 36,439 | 5 | 5 |
| SLH-DSA | slh-dsa-sha2-192f | verification | 1,840 | 1,838–1,847 | 2,110 | 1,852 | 0.04 | 1,871 | 5 | 5 |
| SLH-DSA | slh-dsa-sha2-192s | key generation | 81,565 | 81,452–81,683 | 85,590 | 81,733 | 0.01 | 83,984 | 5 | 5 |
| SLH-DSA | slh-dsa-sha2-192s | signing | 774,376 | 773,111–776,415 | 805,977 | 776,009 | 0.01 | 775,933 | 5 | 5 |
| SLH-DSA | slh-dsa-sha2-192s | verification | 641 | 638–642 | 762 | 648 | 0.06 | 728 | 5 | 5 |
| SLH-DSA | slh-dsa-sha2-256f | key generation | 3,462 | 3,450–3,474 | 3,991 | 3,495 | 0.04 | 5,267 | 5 | 5 |
| SLH-DSA | slh-dsa-sha2-256f | signing | 71,459 | 71,326–71,627 | 77,064 | 71,741 | 0.02 | 73,037 | 5 | 5 |
| SLH-DSA | slh-dsa-sha2-256f | verification | 1,910 | 1,907–1,916 | 2,240 | 1,924 | 0.05 | 1,970 | 5 | 5 |
| SLH-DSA | slh-dsa-sha2-256s | key generation | 54,867 | 54,716–54,893 | 58,858 | 55,011 | 0.01 | 57,871 | 5 | 5 |
| SLH-DSA | slh-dsa-sha2-256s | signing | 702,270 | 701,822–705,464 | 733,124 | 704,422 | 0.01 | 707,663 | 5 | 5 |
| SLH-DSA | slh-dsa-sha2-256s | verification | 953 | 949–960 | 1,187 | 963 | 0.07 | 1,035 | 5 | 5 |
| SLH-DSA | slh-dsa-shake-128f | key generation | 1,696 | 1,692–1,708 | 1,950 | 1,697 | 0.07 | 3,488 | 5 | 5 |
| SLH-DSA | slh-dsa-shake-128f | signing | 41,314 | 41,110–41,347 | 44,402 | 41,387 | 0.02 | 42,996 | 5 | 5 |
| SLH-DSA | slh-dsa-shake-128f | verification | 2,373 | 2,363–2,414 | 2,819 | 2,385 | 0.06 | 2,609 | 5 | 5 |
| SLH-DSA | slh-dsa-shake-128s | key generation | 113,318 | 113,107–113,406 | 117,210 | 113,440 | 0.01 | 114,661 | 5 | 5 |
| SLH-DSA | slh-dsa-shake-128s | signing | 861,502 | 860,665–863,277 | 886,878 | 863,646 | 0.01 | 866,438 | 5 | 5 |
| SLH-DSA | slh-dsa-shake-128s | verification | 780 | 772–782 | 957 | 786 | 0.10 | 854 | 5 | 5 |
| SLH-DSA | slh-dsa-shake-192f | key generation | 2,483 | 2,477–2,485 | 2,847 | 2,490 | 0.05 | 4,311 | 5 | 5 |
| SLH-DSA | slh-dsa-shake-192f | signing | 66,970 | 66,840–67,031 | 70,854 | 67,117 | 0.02 | 68,767 | 5 | 5 |
| SLH-DSA | slh-dsa-shake-192f | verification | 3,535 | 3,527–3,548 | 4,010 | 3,545 | 0.05 | 3,445 | 5 | 5 |
| SLH-DSA | slh-dsa-shake-192s | key generation | 166,066 | 166,016–166,145 | 172,262 | 166,222 | 0.01 | 168,127 | 5 | 5 |
| SLH-DSA | slh-dsa-shake-192s | signing | 1,502,757 | 1,500,819–1,504,985 | 1,526,879 | 1,502,800 | 0.00 | 1,509,705 | 5 | 5 |
| SLH-DSA | slh-dsa-shake-192s | verification | 1,162 | 1,158–1,162 | 1,386 | 1,167 | 0.08 | 1,230 | 5 | 5 |
| SLH-DSA | slh-dsa-shake-256f | key generation | 6,054 | 6,041–6,064 | 6,665 | 6,095 | 0.03 | 7,926 | 5 | 5 |
| SLH-DSA | slh-dsa-shake-256f | signing | 124,677 | 124,401–124,876 | 134,125 | 125,068 | 0.02 | 126,235 | 5 | 5 |
| SLH-DSA | slh-dsa-shake-256f | verification | 3,266 | 3,257–3,276 | 3,733 | 3,286 | 0.04 | 3,320 | 5 | 5 |
| SLH-DSA | slh-dsa-shake-256s | key generation | 96,074 | 95,901–96,244 | 100,832 | 96,331 | 0.01 | 98,234 | 5 | 5 |
| SLH-DSA | slh-dsa-shake-256s | signing | 1,223,962 | 1,221,178–1,225,435 | 1,247,008 | 1,225,966 | 0.01 | 1,229,477 | 5 | 5 |
| SLH-DSA | slh-dsa-shake-256s | verification | 1,617 | 1,615–1,627 | 1,917 | 1,631 | 0.06 | 1,623 | 5 | 5 |
| classical | ecdsa-p256 | key generation | 38.4 | 38.3–38.8 | 82.1 | 42.0 | 0.22 | 857 | 5 | 5 |
| classical | ecdsa-p256 | signing | 24.4 | 24.1–24.5 | 56.3 | 27.6 | 0.27 | 794 | 5 | 5 |
| classical | ecdsa-p256 | verification | 58.9 | 58.6–59.6 | 103 | 64.4 | 0.17 | 142 | 5 | 5 |
| classical | ed25519 | key generation | 41.7 | 41.4–41.8 | 78.6 | 44.8 | 0.19 | 733 | 5 | 5 |
| classical | ed25519 | signing | 38.1 | 38.0–38.2 | 80.1 | 41.5 | 0.21 | 82.5 | 5 | 5 |
| classical | ed25519 | verification | 101 | 101–102 | 161 | 108 | 0.14 | 159 | 5 | 5 |
| classical | rsa3072-pss | key generation | 523,698 | 465,024–598,382 | 1,486,649 | 578,459 | 0.57 | 585,699 | 5 | 5 |
| classical | rsa3072-pss | signing | 1,527 | 1,522–1,530 | 2,156 | 1,556 | 0.08 | 2,857 | 5 | 5 |
| classical | rsa3072-pss | verification | 36.9 | 36.8–37.0 | 68.8 | 39.8 | 0.18 | 138 | 5 | 5 |

Generated by `analysis/paper_tables.py` from `results/combined/analysis_tables/T2_E1_primitive_latency.csv`.
