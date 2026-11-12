# E01a  Signature schemes on the VM (virtualised x86 server without AVX2): latency per call (µs)

Median over five repetitions (headline attempts) with its 95 % bootstrap interval over the repetitions; p99, mean and coefficient of variation of the individual calls; the first call of a fresh process.

| Family | Algorithm | Operation | Median | 95 % CI | p99 | Mean | CoV | First call | Repetitions | Attempts |
|---|---|---|---:|---|---:|---:|---:|---:|---:|---:|
| ML-DSA | ml-dsa-44 | key generation | 77.3 | 76.6–80.4 | 101 | 76.2 | 0.12 | 1,823 | 5 | 5 |
| ML-DSA | ml-dsa-44 | signing | 206 | 200–217 | 827 | 255 | 0.62 | 2,225 | 5 | 5 |
| ML-DSA | ml-dsa-44 | verification | 78.7 | 72.8–79.5 | 102 | 77.1 | 0.12 | 212 | 5 | 5 |
| ML-DSA | ml-dsa-65 | key generation | 132 | 128–133 | 170 | 135 | 0.10 | 1,855 | 5 | 5 |
| ML-DSA | ml-dsa-65 | signing | 321 | 319–329 | 1,325 | 402 | 0.64 | 2,236 | 5 | 5 |
| ML-DSA | ml-dsa-65 | verification | 108 | 108–114 | 145 | 115 | 0.11 | 250 | 5 | 5 |
| ML-DSA | ml-dsa-87 | key generation | 183 | 181–192 | 234 | 188 | 0.09 | 1,925 | 5 | 5 |
| ML-DSA | ml-dsa-87 | signing | 415 | 406–419 | 1,455 | 493 | 0.53 | 2,291 | 5 | 5 |
| ML-DSA | ml-dsa-87 | verification | 184 | 178–189 | 227 | 185 | 0.09 | 333 | 5 | 5 |
| Falcon | falcon-1024 | key generation | 32,969 | 32,704–33,166 | 83,941 | 37,138 | 0.37 | 28,839 | 5 | 5 |
| Falcon | falcon-1024 | signing | 7,911 | 7,908–7,930 | 8,456 | 7,893 | 0.04 | 10,208 | 5 | 5 |
| Falcon | falcon-1024 | verification | 78.3 | 76.1–81.9 | 104 | 80.5 | 0.16 | 168 | 5 | 5 |
| Falcon | falcon-512 | key generation | 11,581 | 11,437–11,603 | 27,830 | 12,881 | 0.32 | 12,284 | 5 | 5 |
| Falcon | falcon-512 | signing | 3,667 | 3,639–3,673 | 3,902 | 3,703 | 0.05 | 5,595 | 5 | 5 |
| Falcon | falcon-512 | verification | 36.2 | 33.4–38.2 | 54.0 | 38.4 | 0.15 | 97.5 | 5 | 5 |
| Falcon | falcon-padded-1024 | key generation | 33,077 | 32,046–33,400 | 89,987 | 37,457 | 0.38 | 60,493 | 5 | 5 |
| Falcon | falcon-padded-1024 | signing | 7,931 | 7,690–8,159 | 8,489 | 7,993 | 0.05 | 10,316 | 5 | 5 |
| Falcon | falcon-padded-1024 | verification | 79.4 | 76.7–81.4 | 105 | 80.3 | 0.11 | 170 | 5 | 5 |
| Falcon | falcon-padded-512 | key generation | 11,508 | 11,358–11,767 | 27,953 | 12,789 | 0.32 | 15,947 | 5 | 5 |
| Falcon | falcon-padded-512 | signing | 3,655 | 3,639–3,738 | 3,898 | 3,670 | 0.05 | 5,766 | 5 | 5 |
| Falcon | falcon-padded-512 | verification | 38.0 | 37.5–38.3 | 55.0 | 39.8 | 0.14 | 83.5 | 5 | 5 |
| SLH-DSA | slh-dsa-sha2-128f | key generation | 981 | 963–987 | 1,056 | 977 | 0.05 | 2,868 | 5 | 5 |
| SLH-DSA | slh-dsa-sha2-128f | signing | 22,651 | 21,979–22,865 | 23,991 | 22,620 | 0.04 | 24,558 | 5 | 5 |
| SLH-DSA | slh-dsa-sha2-128f | verification | 1,394 | 1,347–1,408 | 1,529 | 1,392 | 0.06 | 1,460 | 5 | 5 |
| SLH-DSA | slh-dsa-sha2-128s | key generation | 61,058 | 60,233–61,394 | 64,342 | 60,986 | 0.03 | 61,776 | 5 | 5 |
| SLH-DSA | slh-dsa-sha2-128s | signing | 470,114 | 464,353–471,559 | 489,656 | 470,582 | 0.02 | 464,950 | 5 | 5 |
| SLH-DSA | slh-dsa-sha2-128s | verification | 471 | 460–485 | 550 | 470 | 0.08 | 552 | 5 | 5 |
| SLH-DSA | slh-dsa-sha2-192f | key generation | 1,404 | 1,399–1,425 | 1,521 | 1,396 | 0.06 | 3,027 | 5 | 5 |
| SLH-DSA | slh-dsa-sha2-192f | signing | 36,745 | 36,444–37,398 | 39,117 | 36,730 | 0.03 | 38,271 | 5 | 5 |
| SLH-DSA | slh-dsa-sha2-192f | verification | 1,996 | 1,983–2,018 | 2,187 | 1,980 | 0.06 | 2,051 | 5 | 5 |
| SLH-DSA | slh-dsa-sha2-192s | key generation | 90,012 | 88,110–91,178 | 93,259 | 89,738 | 0.02 | 90,279 | 5 | 5 |
| SLH-DSA | slh-dsa-sha2-192s | signing | 838,269 | 835,280–848,120 | 867,939 | 838,405 | 0.02 | 848,441 | 5 | 5 |
| SLH-DSA | slh-dsa-sha2-192s | verification | 724 | 704–726 | 798 | 722 | 0.06 | 780 | 5 | 5 |
| SLH-DSA | slh-dsa-sha2-256f | key generation | 3,674 | 3,655–3,705 | 3,948 | 3,643 | 0.05 | 5,044 | 5 | 5 |
| SLH-DSA | slh-dsa-sha2-256f | signing | 76,655 | 75,594–77,867 | 80,160 | 76,628 | 0.02 | 80,654 | 5 | 5 |
| SLH-DSA | slh-dsa-sha2-256f | verification | 2,067 | 2,050–2,081 | 2,256 | 2,073 | 0.06 | 2,151 | 5 | 5 |
| SLH-DSA | slh-dsa-sha2-256s | key generation | 58,677 | 57,555–59,284 | 61,908 | 58,341 | 0.03 | 60,911 | 5 | 5 |
| SLH-DSA | slh-dsa-sha2-256s | signing | 748,938 | 744,026–756,672 | 779,046 | 747,840 | 0.02 | 755,337 | 5 | 5 |
| SLH-DSA | slh-dsa-sha2-256s | verification | 1,042 | 1,039–1,052 | 1,154 | 1,041 | 0.06 | 1,125 | 5 | 5 |
| SLH-DSA | slh-dsa-shake-128f | key generation | 1,621 | 1,607–1,632 | 1,743 | 1,623 | 0.04 | 3,327 | 5 | 5 |
| SLH-DSA | slh-dsa-shake-128f | signing | 37,564 | 37,386–38,327 | 39,745 | 37,684 | 0.03 | 39,703 | 5 | 5 |
| SLH-DSA | slh-dsa-shake-128f | verification | 2,248 | 2,215–2,288 | 2,481 | 2,244 | 0.06 | 2,368 | 5 | 5 |
| SLH-DSA | slh-dsa-shake-128s | key generation | 102,508 | 101,223–103,025 | 107,409 | 102,275 | 0.03 | 102,131 | 5 | 5 |
| SLH-DSA | slh-dsa-shake-128s | signing | 789,143 | 775,907–795,635 | 817,609 | 787,626 | 0.02 | 788,370 | 5 | 5 |
| SLH-DSA | slh-dsa-shake-128s | verification | 776 | 736–793 | 869 | 776 | 0.06 | 832 | 5 | 5 |
| SLH-DSA | slh-dsa-shake-192f | key generation | 2,361 | 2,335–2,371 | 2,536 | 2,357 | 0.05 | 4,178 | 5 | 5 |
| SLH-DSA | slh-dsa-shake-192f | signing | 61,681 | 61,486–62,332 | 65,152 | 61,780 | 0.03 | 63,455 | 5 | 5 |
| SLH-DSA | slh-dsa-shake-192f | verification | 3,269 | 3,252–3,321 | 3,564 | 3,269 | 0.05 | 3,274 | 5 | 5 |
| SLH-DSA | slh-dsa-shake-192s | key generation | 150,323 | 149,349–153,941 | 157,474 | 149,838 | 0.03 | 150,866 | 5 | 5 |
| SLH-DSA | slh-dsa-shake-192s | signing | 1,404,275 | 1,395,579–1,419,865 | 1,442,131 | 1,399,188 | 0.02 | 1,416,279 | 5 | 5 |
| SLH-DSA | slh-dsa-shake-192s | verification | 1,134 | 1,120–1,155 | 1,263 | 1,132 | 0.07 | 1,215 | 5 | 5 |
| SLH-DSA | slh-dsa-shake-256f | key generation | 6,201 | 6,175–6,255 | 6,633 | 6,224 | 0.04 | 7,854 | 5 | 5 |
| SLH-DSA | slh-dsa-shake-256f | signing | 126,908 | 125,311–129,047 | 133,914 | 126,819 | 0.02 | 128,414 | 5 | 5 |
| SLH-DSA | slh-dsa-shake-256f | verification | 3,364 | 3,285–3,383 | 3,662 | 3,370 | 0.05 | 3,351 | 5 | 5 |
| SLH-DSA | slh-dsa-shake-256s | key generation | 100,021 | 97,107–100,655 | 104,144 | 99,988 | 0.03 | 100,224 | 5 | 5 |
| SLH-DSA | slh-dsa-shake-256s | signing | 1,247,915 | 1,180,219–1,255,170 | 1,293,190 | 1,248,374 | 0.02 | 1,266,923 | 5 | 5 |
| SLH-DSA | slh-dsa-shake-256s | verification | 1,667 | 1,634–1,670 | 1,845 | 1,663 | 0.06 | 1,780 | 5 | 5 |
| classical | ecdsa-p256 | key generation | 40.8 | 38.8–41.3 | 61.8 | 43.2 | 0.15 | 1,037 | 5 | 5 |
| classical | ecdsa-p256 | signing | 30.9 | 29.3–31.5 | 50.5 | 33.0 | 0.21 | 722 | 5 | 5 |
| classical | ecdsa-p256 | verification | 78.4 | 72.4–78.5 | 103 | 75.7 | 0.17 | 251 | 5 | 5 |
| classical | ed25519 | key generation | 46.7 | 46.2–47.1 | 67.2 | 49.2 | 0.14 | 842 | 5 | 5 |
| classical | ed25519 | signing | 40.3 | 38.1–40.6 | 60.0 | 42.9 | 0.16 | 93.0 | 5 | 5 |
| classical | ed25519 | verification | 102 | 97.3–103 | 136 | 105 | 0.11 | 166 | 5 | 5 |
| classical | rsa3072-pss | key generation | 510,364 | 476,637–591,476 | 1,543,241 | 578,258 | 0.53 | 1,021,128 | 5 | 5 |
| classical | rsa3072-pss | signing | 2,176 | 2,155–2,183 | 2,870 | 2,211 | 0.06 | 3,731 | 5 | 5 |
| classical | rsa3072-pss | verification | 53.3 | 50.2–53.4 | 75.5 | 56.6 | 0.21 | 211 | 5 | 5 |

Generated by `analysis/paper_tables.py` from `results/combined/analysis_tables/T2_E1_primitive_latency.csv`.
