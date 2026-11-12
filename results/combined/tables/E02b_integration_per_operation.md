# E02b  Reused key against the production pattern, every operation (µs)

Medians over five repetitions. '–': not run on that machine (SLH-DSA on the laptop and the VM).

| Family | Algorithm | Operation | Reuse, Mac M1 | Production, Mac M1 | Ratio, Mac M1 | Reuse, Laptop | Production, Laptop | Ratio, Laptop | Reuse, VM | Production, VM | Ratio, VM |
|---|---|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| ML-DSA | ml-dsa-44 | key generation | 30.7 | 35.5 | 1.16× | 32.2 | 39.0 | 1.21× | 80.2 | 87.6 | 1.09× |
| ML-DSA | ml-dsa-44 | signing | 60.4 | 67.3 | 1.12× | 80.0 | 89.2 | 1.12× | 207 | 217 | 1.05× |
| ML-DSA | ml-dsa-44 | verification | 28.7 | 32.8 | 1.14× | 32.1 | 37.5 | 1.17× | 78.5 | 84.1 | 1.07× |
| ML-DSA | ml-dsa-65 | key generation | 53.3 | 60.9 | 1.14× | 54.9 | 65.2 | 1.19× | 133 | 145 | 1.09× |
| ML-DSA | ml-dsa-65 | signing | 97.6 | 108 | 1.12× | 126 | 139 | 1.10× | 323 | 336 | 1.04× |
| ML-DSA | ml-dsa-65 | verification | 48.9 | 54.6 | 1.12× | 53.2 | 60.6 | 1.14× | 115 | 121 | 1.06× |
| ML-DSA | ml-dsa-87 | key generation | 93.1 | 102 | 1.10× | 92.0 | 104 | 1.13× | 188 | 200 | 1.07× |
| ML-DSA | ml-dsa-87 | signing | 140 | 161 | 1.15× | 154 | 170 | 1.11× | 416 | 427 | 1.03× |
| ML-DSA | ml-dsa-87 | verification | 86.7 | 95.1 | 1.09× | 87.6 | 97.3 | 1.10× | 183 | 191 | 1.05× |
| Falcon | falcon-1024 | key generation | 14,783 | 14,822 | 1.00× | 15,788 | 15,857 | 1.00× | 33,372 | 33,274 | 1.00× |
| Falcon | falcon-1024 | signing | 292 | 298 | 1.02× | 420 | 428 | 1.02× | 7,986 | 7,989 | 1.00× |
| Falcon | falcon-1024 | verification | 40.7 | 44.4 | 1.09× | 81.6 | 87.0 | 1.07× | 80.5 | 86.4 | 1.07× |
| Falcon | falcon-512 | key generation | 4,801 | 4,814 | 1.00× | 5,612 | 5,625 | 1.00× | 11,631 | 11,612 | 1.00× |
| Falcon | falcon-512 | signing | 147 | 150 | 1.02× | 209 | 214 | 1.02× | 3,654 | 3,660 | 1.00× |
| Falcon | falcon-512 | verification | 20.8 | 22.9 | 1.10× | 38.9 | 43.3 | 1.11× | 34.5 | 37.5 | 1.08× |
| Falcon | falcon-padded-1024 | key generation | 14,778 | 14,790 | 1.00× | 15,796 | 15,803 | 1.00× | 33,664 | 33,694 | 1.01× |
| Falcon | falcon-padded-1024 | signing | 289 | 295 | 1.02× | 421 | 430 | 1.02× | 7,955 | 7,980 | 1.00× |
| Falcon | falcon-padded-1024 | verification | 40.8 | 44.6 | 1.09× | 81.8 | 86.9 | 1.06× | 77.5 | 82.9 | 1.07× |
| Falcon | falcon-padded-512 | key generation | 4,795 | 4,815 | 1.00× | 5,595 | 5,630 | 1.00× | 11,449 | 11,599 | 1.01× |
| Falcon | falcon-padded-512 | signing | 146 | 149 | 1.02× | 210 | 215 | 1.02× | 3,657 | 3,662 | 1.00× |
| Falcon | falcon-padded-512 | verification | 20.5 | 22.5 | 1.10× | 38.9 | 43.3 | 1.11× | 34.5 | 37.2 | 1.08× |
| SLH-DSA | slh-dsa-sha2-128f | key generation | 888 | 892 | 1.00× | – | – | – | – | – | – |
| SLH-DSA | slh-dsa-sha2-128f | signing | 20,741 | 20,783 | 1.00× | – | – | – | – | – | – |
| SLH-DSA | slh-dsa-sha2-128f | verification | 1,257 | 1,282 | 1.02× | – | – | – | – | – | – |
| SLH-DSA | slh-dsa-sha2-128s | key generation | 56,811 | 56,819 | 1.00× | – | – | – | – | – | – |
| SLH-DSA | slh-dsa-sha2-128s | signing | 431,471 | 431,444 | 1.00× | – | – | – | – | – | – |
| SLH-DSA | slh-dsa-sha2-128s | verification | 430 | 441 | 1.02× | – | – | – | – | – | – |
| SLH-DSA | slh-dsa-sha2-192f | key generation | 1,298 | 1,300 | 1.00× | – | – | – | – | – | – |
| SLH-DSA | slh-dsa-sha2-192f | signing | 33,798 | 33,860 | 1.00× | – | – | – | – | – | – |
| SLH-DSA | slh-dsa-sha2-192f | verification | 1,832 | 1,874 | 1.02× | – | – | – | – | – | – |
| SLH-DSA | slh-dsa-sha2-192s | key generation | 82,763 | 82,774 | 1.00× | – | – | – | – | – | – |
| SLH-DSA | slh-dsa-sha2-192s | signing | 757,933 | 757,978 | 1.00× | – | – | – | – | – | – |
| SLH-DSA | slh-dsa-sha2-192s | verification | 634 | 654 | 1.03× | – | – | – | – | – | – |
| SLH-DSA | slh-dsa-sha2-256f | key generation | 3,423 | 3,427 | 1.00× | – | – | – | – | – | – |
| SLH-DSA | slh-dsa-sha2-256f | signing | 69,555 | 69,653 | 1.00× | – | – | – | – | – | – |
| SLH-DSA | slh-dsa-sha2-256f | verification | 1,873 | 1,924 | 1.03× | – | – | – | – | – | – |
| SLH-DSA | slh-dsa-sha2-256s | key generation | 54,658 | 54,667 | 1.00× | – | – | – | – | – | – |
| SLH-DSA | slh-dsa-sha2-256s | signing | 668,661 | 668,715 | 1.00× | – | – | – | – | – | – |
| SLH-DSA | slh-dsa-sha2-256s | verification | 930 | 976 | 1.05× | – | – | – | – | – | – |
| SLH-DSA | slh-dsa-shake-128f | key generation | 1,120 | 1,124 | 1.00× | – | – | – | – | – | – |
| SLH-DSA | slh-dsa-shake-128f | signing | 26,488 | 26,534 | 1.00× | – | – | – | – | – | – |
| SLH-DSA | slh-dsa-shake-128f | verification | 1,571 | 1,595 | 1.02× | – | – | – | – | – | – |
| SLH-DSA | slh-dsa-shake-128s | key generation | 71,862 | 71,867 | 1.00× | – | – | – | – | – | – |
| SLH-DSA | slh-dsa-shake-128s | signing | 549,142 | 549,846 | 1.00× | – | – | – | – | – | – |
| SLH-DSA | slh-dsa-shake-128s | verification | 544 | 556 | 1.02× | – | – | – | – | – | – |
| SLH-DSA | slh-dsa-shake-192f | key generation | 1,652 | 1,655 | 1.00× | – | – | – | – | – | – |
| SLH-DSA | slh-dsa-shake-192f | signing | 43,548 | 43,618 | 1.00× | – | – | – | – | – | – |
| SLH-DSA | slh-dsa-shake-192f | verification | 2,301 | 2,342 | 1.02× | – | – | – | – | – | – |
| SLH-DSA | slh-dsa-shake-192s | key generation | 104,653 | 104,723 | 1.00× | – | – | – | – | – | – |
| SLH-DSA | slh-dsa-shake-192s | signing | 986,579 | 986,092 | 1.00× | – | – | – | – | – | – |
| SLH-DSA | slh-dsa-shake-192s | verification | 794 | 814 | 1.03× | – | – | – | – | – | – |
| SLH-DSA | slh-dsa-shake-256f | key generation | 4,347 | 4,353 | 1.00× | – | – | – | – | – | – |
| SLH-DSA | slh-dsa-shake-256f | signing | 89,736 | 89,706 | 1.00× | – | – | – | – | – | – |
| SLH-DSA | slh-dsa-shake-256f | verification | 2,385 | 2,438 | 1.02× | – | – | – | – | – | – |
| SLH-DSA | slh-dsa-shake-256s | key generation | 69,266 | 69,255 | 1.00× | – | – | – | – | – | – |
| SLH-DSA | slh-dsa-shake-256s | signing | 884,324 | 884,707 | 1.00× | – | – | – | – | – | – |
| SLH-DSA | slh-dsa-shake-256s | verification | 1,167 | 1,215 | 1.04× | – | – | – | – | – | – |
| ML-KEM | ml-kem-1024 | decapsulation | 24.0 | 29.4 | 1.22× | 34.2 | 41.1 | 1.20× | 58.7 | 65.6 | 1.11× |
| ML-KEM | ml-kem-1024 | encapsulation | 21.4 | 25.2 | 1.18× | 29.3 | 35.2 | 1.20× | 54.4 | 60.4 | 1.11× |
| ML-KEM | ml-kem-1024 | key generation | 20.2 | 26.2 | 1.29× | 28.7 | 37.1 | 1.29× | 54.9 | 63.8 | 1.16× |
| ML-KEM | ml-kem-512 | decapsulation | 10.2 | 13.2 | 1.29× | 14.7 | 18.9 | 1.28× | 26.3 | 30.2 | 1.14× |
| ML-KEM | ml-kem-512 | encapsulation | 9.2 | 11.3 | 1.24× | 12.9 | 16.5 | 1.28× | 25.8 | 29.2 | 1.14× |
| ML-KEM | ml-kem-512 | key generation | 8.5 | 11.6 | 1.36× | 12.2 | 17.0 | 1.40× | 23.2 | 28.4 | 1.22× |
| ML-KEM | ml-kem-768 | decapsulation | 16.0 | 20.1 | 1.26× | 23.8 | 29.5 | 1.24× | 41.9 | 47.0 | 1.12× |
| ML-KEM | ml-kem-768 | encapsulation | 14.2 | 17.1 | 1.20× | 20.6 | 25.1 | 1.22× | 38.6 | 43.1 | 1.12× |
| ML-KEM | ml-kem-768 | key generation | 13.6 | 18.0 | 1.33× | 20.4 | 27.1 | 1.32× | 36.8 | 43.8 | 1.19× |
| HQC | hqc-1 | decapsulation | 2,730 | 2,753 | 1.01× | 3,304 | 3,316 | 1.00× | 3,716 | 3,729 | 1.00× |
| HQC | hqc-1 | encapsulation | 1,791 | 1,817 | 1.01× | 2,164 | 2,182 | 1.01× | 2,444 | 2,450 | 1.00× |
| HQC | hqc-1 | key generation | 896 | 920 | 1.03× | 1,080 | 1,092 | 1.01× | 1,229 | 1,239 | 1.01× |
| HQC | hqc-3 | decapsulation | 7,047 | 7,072 | 1.00× | 9,810 | 9,844 | 1.00× | 11,105 | 11,128 | 1.00× |
| HQC | hqc-3 | encapsulation | 4,671 | 4,702 | 1.01× | 6,512 | 6,548 | 1.01× | 7,265 | 7,291 | 1.00× |
| HQC | hqc-3 | key generation | 2,332 | 2,362 | 1.01× | 3,272 | 3,292 | 1.01× | 3,653 | 3,668 | 1.00× |
| HQC | hqc-5 | decapsulation | 17,797 | 17,837 | 1.00× | 23,943 | 24,004 | 1.00× | 27,062 | 27,115 | 1.00× |
| HQC | hqc-5 | encapsulation | 11,795 | 11,835 | 1.00× | 15,901 | 15,954 | 1.00× | 17,739 | 17,776 | 1.00× |
| HQC | hqc-5 | key generation | 5,907 | 5,946 | 1.01× | 7,982 | 8,020 | 1.00× | 8,849 | 8,877 | 1.00× |
| classical | dhkem-p256 | decapsulation | 100 | 115 | 1.15× | 117 | 143 | 1.22× | 127 | 149 | 1.18× |
| classical | dhkem-p256 | encapsulation | 113 | 128 | 1.14× | 132 | 158 | 1.20× | 144 | 166 | 1.15× |
| classical | dhkem-p256 | key generation | 25.6 | 25.7 | 1.00× | 37.5 | 37.8 | 1.01× | 39.8 | 40.1 | 1.01× |
| classical | dhkem-x25519 | decapsulation | 37.7 | 68.4 | 1.81× | 39.1 | 77.9 | 1.99× | 44.7 | 86.0 | 1.90× |
| classical | dhkem-x25519 | encapsulation | 67.4 | 70.0 | 1.04× | 78.7 | 83.9 | 1.07× | 90.3 | 94.8 | 1.05× |
| classical | dhkem-x25519 | key generation | 31.4 | 31.5 | 1.00× | 40.1 | 40.4 | 1.01× | 44.6 | 44.9 | 1.01× |
| classical | ecdsa-p256 | key generation | 26.4 | 26.5 | 1.01× | 38.1 | 38.5 | 1.01× | 40.9 | 41.3 | 1.01× |
| classical | ecdsa-p256 | signing | 20.5 | 36.7 | 1.79× | 24.6 | 54.8 | 2.22× | 31.8 | 61.7 | 1.94× |
| classical | ecdsa-p256 | verification | 55.7 | 71.0 | 1.27× | 59.2 | 89.7 | 1.51× | 77.9 | 102 | 1.31× |
| classical | ed25519 | key generation | 32.5 | 32.6 | 1.00× | 41.7 | 42.0 | 1.01× | 44.6 | 44.9 | 1.01× |
| classical | ed25519 | signing | 30.4 | 61.4 | 2.01× | 38.1 | 77.9 | 2.05× | 40.6 | 82.5 | 2.03× |
| classical | ed25519 | verification | 81.5 | 83.7 | 1.03× | 101 | 105 | 1.04× | 101 | 104 | 1.03× |
| classical | rsa3072-pss | key generation | 437,695 | 451,661 | 1.00× | 499,458 | 507,452 | 0.99× | 533,062 | 510,892 | 0.94× |
| classical | rsa3072-pss | signing | 1,613 | 2,466 | 1.53× | 1,527 | 2,717 | 1.78× | 2,170 | 3,181 | 1.47× |
| classical | rsa3072-pss | verification | 33.7 | 128 | 3.79× | 37.5 | 182 | 4.87× | 53.7 | 171 | 3.20× |

Generated by `analysis/paper_tables.py` from `results/combined/analysis_tables/T3_E2_integration_overhead.csv`.
