# E01c  99th-percentile latency per call (µs), and p99 against the median

p99 of the individual calls of the headline attempts (median over the repetitions).

| Family | Algorithm | Operation | p99, Mac M1 | p99, Laptop | p99, VM | p99 ÷ median, Mac M1 | p99 ÷ median, Laptop | p99 ÷ median, VM |
|---|---|---|---:|---:|---:|---:|---:|---:|
| ML-DSA | ml-dsa-44 | key generation | 36.4 | 62.0 | 101 | 1.2× | 1.9× | 1.3× |
| ML-DSA | ml-dsa-44 | signing | 218 | 313 | 827 | 3.6× | 4.0× | 4.0× |
| ML-DSA | ml-dsa-44 | verification | 32.6 | 59.4 | 102 | 1.1× | 1.9× | 1.3× |
| ML-DSA | ml-dsa-65 | key generation | 67.3 | 95.3 | 170 | 1.3× | 1.7× | 1.3× |
| ML-DSA | ml-dsa-65 | signing | 328 | 494 | 1,325 | 3.4× | 3.9× | 4.1× |
| ML-DSA | ml-dsa-65 | verification | 53.8 | 91.7 | 145 | 1.1× | 1.7× | 1.3× |
| ML-DSA | ml-dsa-87 | key generation | 108 | 141 | 234 | 1.2× | 1.7× | 1.3× |
| ML-DSA | ml-dsa-87 | signing | 388 | 497 | 1,455 | 2.8× | 3.2× | 3.5× |
| ML-DSA | ml-dsa-87 | verification | 93.3 | 138 | 227 | 1.1× | 1.7× | 1.2× |
| Falcon | falcon-1024 | key generation | 32,292 | 35,909 | 83,941 | 2.2× | 2.3× | 2.5× |
| Falcon | falcon-1024 | signing | 317 | 528 | 8,456 | 1.1× | 1.3× | 1.1× |
| Falcon | falcon-1024 | verification | 48.6 | 131 | 104 | 1.1× | 1.6× | 1.3× |
| Falcon | falcon-512 | key generation | 9,701 | 13,277 | 27,830 | 2.0× | 2.4× | 2.4× |
| Falcon | falcon-512 | signing | 167 | 302 | 3,902 | 1.1× | 1.4× | 1.1× |
| Falcon | falcon-512 | verification | 24.2 | 71.5 | 54.0 | 1.1× | 1.7× | 1.5× |
| Falcon | falcon-padded-1024 | key generation | 32,282 | 35,756 | 89,987 | 2.2× | 2.3× | 2.7× |
| Falcon | falcon-padded-1024 | signing | 316 | 551 | 8,489 | 1.1× | 1.3× | 1.1× |
| Falcon | falcon-padded-1024 | verification | 47.9 | 128 | 105 | 1.1× | 1.6× | 1.3× |
| Falcon | falcon-padded-512 | key generation | 9,582 | 13,312 | 27,953 | 2.0× | 2.4× | 2.4× |
| Falcon | falcon-padded-512 | signing | 165 | 298 | 3,898 | 1.1× | 1.4× | 1.1× |
| Falcon | falcon-padded-512 | verification | 24.2 | 71.9 | 55.0 | 1.1× | 1.8× | 1.4× |
| SLH-DSA | slh-dsa-sha2-128f | key generation | 940 | 1,111 | 1,056 | 1.1× | 1.2× | 1.1× |
| SLH-DSA | slh-dsa-sha2-128f | signing | 21,516 | 22,173 | 23,991 | 1.0× | 1.1× | 1.1× |
| SLH-DSA | slh-dsa-sha2-128f | verification | 1,345 | 1,463 | 1,529 | 1.1× | 1.2× | 1.1× |
| SLH-DSA | slh-dsa-sha2-128s | key generation | 58,816 | 60,101 | 64,342 | 1.0× | 1.1× | 1.1× |
| SLH-DSA | slh-dsa-sha2-128s | signing | 436,269 | 451,001 | 489,656 | 1.0× | 1.0× | 1.0× |
| SLH-DSA | slh-dsa-sha2-128s | verification | 474 | 560 | 550 | 1.1× | 1.3× | 1.2× |
| SLH-DSA | slh-dsa-sha2-192f | key generation | 1,372 | 1,495 | 1,521 | 1.1× | 1.2× | 1.1× |
| SLH-DSA | slh-dsa-sha2-192f | signing | 34,964 | 37,440 | 39,117 | 1.0× | 1.1× | 1.1× |
| SLH-DSA | slh-dsa-sha2-192f | verification | 1,933 | 2,110 | 2,187 | 1.1× | 1.1× | 1.1× |
| SLH-DSA | slh-dsa-sha2-192s | key generation | 85,449 | 85,590 | 93,259 | 1.0× | 1.0× | 1.0× |
| SLH-DSA | slh-dsa-sha2-192s | signing | 762,614 | 805,977 | 867,939 | 1.0× | 1.0× | 1.0× |
| SLH-DSA | slh-dsa-sha2-192s | verification | 689 | 762 | 798 | 1.1× | 1.2× | 1.1× |
| SLH-DSA | slh-dsa-sha2-256f | key generation | 3,620 | 3,991 | 3,948 | 1.1× | 1.2× | 1.1× |
| SLH-DSA | slh-dsa-sha2-256f | signing | 71,641 | 77,064 | 80,160 | 1.0× | 1.1× | 1.0× |
| SLH-DSA | slh-dsa-sha2-256f | verification | 1,984 | 2,240 | 2,256 | 1.1× | 1.2× | 1.1× |
| SLH-DSA | slh-dsa-sha2-256s | key generation | 56,344 | 58,858 | 61,908 | 1.0× | 1.1× | 1.1× |
| SLH-DSA | slh-dsa-sha2-256s | signing | 673,986 | 733,124 | 779,046 | 1.0× | 1.0× | 1.0× |
| SLH-DSA | slh-dsa-sha2-256s | verification | 1,003 | 1,187 | 1,154 | 1.1× | 1.2× | 1.1× |
| SLH-DSA | slh-dsa-shake-128f | key generation | 1,185 | 1,950 | 1,743 | 1.1× | 1.1× | 1.1× |
| SLH-DSA | slh-dsa-shake-128f | signing | 27,018 | 44,402 | 39,745 | 1.0× | 1.1× | 1.1× |
| SLH-DSA | slh-dsa-shake-128f | verification | 1,677 | 2,819 | 2,481 | 1.1× | 1.2× | 1.1× |
| SLH-DSA | slh-dsa-shake-128s | key generation | 73,295 | 117,210 | 107,409 | 1.0× | 1.0× | 1.0× |
| SLH-DSA | slh-dsa-shake-128s | signing | 556,830 | 886,878 | 817,609 | 1.0× | 1.0× | 1.0× |
| SLH-DSA | slh-dsa-shake-128s | verification | 597 | 957 | 869 | 1.1× | 1.2× | 1.1× |
| SLH-DSA | slh-dsa-shake-192f | key generation | 1,731 | 2,847 | 2,536 | 1.1× | 1.1× | 1.1× |
| SLH-DSA | slh-dsa-shake-192f | signing | 44,805 | 70,854 | 65,152 | 1.0× | 1.1× | 1.1× |
| SLH-DSA | slh-dsa-shake-192f | verification | 2,439 | 4,010 | 3,564 | 1.1× | 1.1× | 1.1× |
| SLH-DSA | slh-dsa-shake-192s | key generation | 107,382 | 172,262 | 157,474 | 1.0× | 1.0× | 1.0× |
| SLH-DSA | slh-dsa-shake-192s | signing | 996,186 | 1,526,879 | 1,442,131 | 1.0× | 1.0× | 1.0× |
| SLH-DSA | slh-dsa-shake-192s | verification | 854 | 1,386 | 1,263 | 1.1× | 1.2× | 1.1× |
| SLH-DSA | slh-dsa-shake-256f | key generation | 4,562 | 6,665 | 6,633 | 1.1× | 1.1× | 1.1× |
| SLH-DSA | slh-dsa-shake-256f | signing | 92,260 | 134,125 | 133,914 | 1.0× | 1.1× | 1.1× |
| SLH-DSA | slh-dsa-shake-256f | verification | 2,489 | 3,733 | 3,662 | 1.1× | 1.1× | 1.1× |
| SLH-DSA | slh-dsa-shake-256s | key generation | 70,968 | 100,832 | 104,144 | 1.0× | 1.0× | 1.0× |
| SLH-DSA | slh-dsa-shake-256s | signing | 896,238 | 1,247,008 | 1,293,190 | 1.0× | 1.0× | 1.0× |
| SLH-DSA | slh-dsa-shake-256s | verification | 1,260 | 1,917 | 1,845 | 1.1× | 1.2× | 1.1× |
| ML-KEM | ml-kem-1024 | key generation | 23.7 | 54.2 | 76.5 | 1.2× | 1.9× | 1.4× |
| ML-KEM | ml-kem-1024 | encapsulation | 25.7 | 56.3 | 76.2 | 1.2× | 1.9× | 1.4× |
| ML-KEM | ml-kem-1024 | decapsulation | 26.4 | 60.9 | 81.3 | 1.1× | 1.8× | 1.4× |
| ML-KEM | ml-kem-512 | key generation | 10.3 | 25.9 | 36.1 | 1.2× | 2.2× | 1.5× |
| ML-KEM | ml-kem-512 | encapsulation | 11.5 | 27.6 | 43.2 | 1.3× | 2.2× | 1.7× |
| ML-KEM | ml-kem-512 | decapsulation | 11.7 | 29.4 | 43.0 | 1.2× | 2.0× | 1.6× |
| ML-KEM | ml-kem-768 | key generation | 16.2 | 38.0 | 54.5 | 1.2× | 1.9× | 1.5× |
| ML-KEM | ml-kem-768 | encapsulation | 18.3 | 42.0 | 58.4 | 1.3× | 2.0× | 1.5× |
| ML-KEM | ml-kem-768 | decapsulation | 18.4 | 42.1 | 59.7 | 1.1× | 1.8× | 1.5× |
| HQC | hqc-1 | key generation | 952 | 1,368 | 1,449 | 1.1× | 1.3× | 1.2× |
| HQC | hqc-1 | encapsulation | 1,870 | 2,573 | 2,761 | 1.0× | 1.2× | 1.1× |
| HQC | hqc-1 | decapsulation | 2,850 | 3,776 | 4,125 | 1.0× | 1.1× | 1.1× |
| HQC | hqc-3 | key generation | 2,452 | 3,875 | 3,898 | 1.0× | 1.2× | 1.1× |
| HQC | hqc-3 | encapsulation | 4,882 | 7,451 | 7,750 | 1.0× | 1.1× | 1.1× |
| HQC | hqc-3 | decapsulation | 7,364 | 10,921 | 11,657 | 1.0× | 1.1× | 1.1× |
| HQC | hqc-5 | key generation | 6,195 | 8,870 | 9,436 | 1.0× | 1.1× | 1.1× |
| HQC | hqc-5 | encapsulation | 12,211 | 17,022 | 18,818 | 1.0× | 1.1× | 1.1× |
| HQC | hqc-5 | decapsulation | 18,274 | 25,874 | 28,328 | 1.0× | 1.1× | 1.1× |
| classical | dhkem-p256 | key generation | 31.7 | 77.0 | 59.8 | 1.2× | 2.1× | 1.6× |
| classical | dhkem-p256 | encapsulation | 124 | 222 | 191 | 1.1× | 1.7× | 1.4× |
| classical | dhkem-p256 | decapsulation | 117 | 206 | 170 | 1.2× | 1.8× | 1.4× |
| classical | dhkem-x25519 | key generation | 37.0 | 77.4 | 62.8 | 1.2× | 1.9× | 1.5× |
| classical | dhkem-x25519 | encapsulation | 77.0 | 147 | 124 | 1.1× | 1.9× | 1.4× |
| classical | dhkem-x25519 | decapsulation | 43.5 | 74.7 | 68.5 | 1.2× | 1.9× | 1.5× |
| classical | ecdsa-p256 | key generation | 31.6 | 82.1 | 61.8 | 1.2× | 2.1× | 1.5× |
| classical | ecdsa-p256 | signing | 25.4 | 56.3 | 50.5 | 1.3× | 2.3× | 1.6× |
| classical | ecdsa-p256 | verification | 60.8 | 103 | 103 | 1.1× | 1.8× | 1.3× |
| classical | ed25519 | key generation | 37.7 | 78.6 | 67.2 | 1.2× | 1.9× | 1.4× |
| classical | ed25519 | signing | 38.9 | 80.1 | 60.0 | 1.3× | 2.1× | 1.5× |
| classical | ed25519 | verification | 91.2 | 161 | 136 | 1.1× | 1.6× | 1.3× |
| classical | rsa3072-pss | key generation | 1,231,200 | 1,486,649 | 1,543,241 | 3.1× | 2.8× | 3.0× |
| classical | rsa3072-pss | signing | 2,102 | 2,156 | 2,870 | 1.3× | 1.4× | 1.3× |
| classical | rsa3072-pss | verification | 38.3 | 68.8 | 75.5 | 1.2× | 1.9× | 1.4× |

Generated by `analysis/paper_tables.py` from `results/combined/analysis_tables/T2_E1_primitive_latency.csv`.
