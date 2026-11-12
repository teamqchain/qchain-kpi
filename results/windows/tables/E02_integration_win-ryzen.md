# E02  QChain's deployed pattern against a reused key on the Laptop (AMD Ryzen 5 7520U, x86 with AVX2) (µs)

Reuse: one key, one library object. Re-init: a new library object per call. Production: QChain's code (hex keys decoded per call). Components: library-object initialisation and hex conversion, timed alone.

| Family | Algorithm | Operation | Reuse | Re-init | Production | Production ÷ reuse | 95 % CI | Re-init ÷ reuse | Initialisation | Hex conversion |
|---|---|---|---:|---:|---:|---:|---|---:|---:|---:|
| ML-DSA | ml-dsa-44 | key generation | 32.2 | 33.7 | 39.0 | 1.211× | 1.210–1.213 | 1.047× | 0.4 | 5.6 |
| ML-DSA | ml-dsa-44 | signing | 80.0 | 83.0 | 89.2 | 1.119× | 1.078–1.145 | 1.038× | 0.4 | 6.2 |
| ML-DSA | ml-dsa-44 | verification | 32.1 | 32.6 | 37.5 | 1.172× | 1.169–1.175 | 1.018× | 0.4 | 4.8 |
| ML-DSA | ml-dsa-65 | key generation | 54.9 | 56.8 | 65.2 | 1.186× | 1.184–1.188 | 1.033× | 0.4 | 8.0 |
| ML-DSA | ml-dsa-65 | signing | 126 | 127 | 139 | 1.102× | 1.097–1.106 | 1.011× | 0.4 | 8.7 |
| ML-DSA | ml-dsa-65 | verification | 53.2 | 53.8 | 60.6 | 1.139× | 1.138–1.143 | 1.011× | 0.4 | 6.2 |
| ML-DSA | ml-dsa-87 | key generation | 92.0 | 94.3 | 104 | 1.129× | 1.128–1.131 | 1.025× | 0.4 | 10.2 |
| ML-DSA | ml-dsa-87 | signing | 154 | 156 | 170 | 1.108× | 1.096–1.110 | 1.015× | 0.4 | 11.3 |
| ML-DSA | ml-dsa-87 | verification | 87.6 | 88.4 | 97.3 | 1.104× | 1.103–1.110 | 1.009× | 0.4 | 8.5 |
| Falcon | falcon-1024 | key generation | 15,788 | 15,800 | 15,857 | 1.004× | 0.999–1.013 | 1.000× | 0.4 | 5.9 |
| Falcon | falcon-1024 | signing | 420 | 421 | 428 | 1.020× | 1.020–1.021 | 1.006× | 0.4 | 4.5 |
| Falcon | falcon-1024 | verification | 81.6 | 82.6 | 87.0 | 1.066× | 1.062–1.067 | 1.011× | 0.4 | 3.6 |
| Falcon | falcon-512 | key generation | 5,612 | 5,606 | 5,625 | 1.002× | 0.998–1.004 | 1.000× | 0.4 | 3.3 |
| Falcon | falcon-512 | signing | 209 | 210 | 214 | 1.023× | 1.022–1.025 | 1.007× | 0.4 | 2.5 |
| Falcon | falcon-512 | verification | 38.9 | 40.5 | 43.3 | 1.114× | 1.112–1.119 | 1.043× | 0.4 | 1.8 |
| Falcon | falcon-padded-1024 | key generation | 15,796 | 15,846 | 15,803 | 1.000× | 0.989–1.010 | 1.000× | 0.4 | 5.9 |
| Falcon | falcon-padded-1024 | signing | 421 | 423 | 430 | 1.020× | 1.020–1.021 | 1.006× | 0.4 | 4.5 |
| Falcon | falcon-padded-1024 | verification | 81.8 | 82.7 | 86.9 | 1.064× | 1.061–1.068 | 1.011× | 0.4 | 3.7 |
| Falcon | falcon-padded-512 | key generation | 5,595 | 5,613 | 5,630 | 1.005× | 1.000–1.009 | 1.004× | 0.4 | 3.3 |
| Falcon | falcon-padded-512 | signing | 210 | 212 | 215 | 1.022× | 1.021–1.023 | 1.007× | 0.4 | 2.5 |
| Falcon | falcon-padded-512 | verification | 38.9 | 40.5 | 43.3 | 1.112× | 1.109–1.114 | 1.035× | 0.4 | 1.8 |
| ML-KEM | ml-kem-1024 | decapsulation | 34.2 | 35.2 | 41.1 | 1.203× | 1.202–1.206 | 1.029× | 0.6 | 5.5 |
| ML-KEM | ml-kem-1024 | encapsulation | 29.3 | 30.2 | 35.2 | 1.198× | 1.197–1.200 | 1.029× | 0.6 | 4.0 |
| ML-KEM | ml-kem-1024 | key generation | 28.7 | 30.5 | 37.1 | 1.292× | 1.290–1.294 | 1.061× | 0.6 | 6.4 |
| ML-KEM | ml-kem-512 | decapsulation | 14.7 | 15.6 | 18.9 | 1.280× | 1.276–1.285 | 1.061× | 0.5 | 2.8 |
| ML-KEM | ml-kem-512 | encapsulation | 12.9 | 13.7 | 16.5 | 1.279× | 1.272–1.285 | 1.062× | 0.5 | 2.2 |
| ML-KEM | ml-kem-512 | key generation | 12.2 | 13.6 | 17.0 | 1.404× | 1.398–1.408 | 1.115× | 0.5 | 3.6 |
| ML-KEM | ml-kem-768 | decapsulation | 23.8 | 24.8 | 29.5 | 1.236× | 1.231–1.238 | 1.040× | 0.5 | 4.1 |
| ML-KEM | ml-kem-768 | encapsulation | 20.6 | 21.4 | 25.1 | 1.218× | 1.216–1.219 | 1.041× | 0.5 | 3.0 |
| ML-KEM | ml-kem-768 | key generation | 20.4 | 22.1 | 27.1 | 1.324× | 1.319–1.327 | 1.082× | 0.5 | 5.3 |
| HQC | hqc-1 | decapsulation | 3,304 | 3,306 | 3,316 | 1.005× | 1.004–1.005 | 1.001× | 0.5 | 8.0 |
| HQC | hqc-1 | encapsulation | 2,164 | 2,169 | 2,182 | 1.008× | 1.008–1.008 | 1.002× | 0.5 | 8.3 |
| HQC | hqc-1 | key generation | 1,080 | 1,085 | 1,092 | 1.012× | 1.011–1.012 | 1.004× | 0.5 | 6.5 |
| HQC | hqc-3 | decapsulation | 9,810 | 9,819 | 9,844 | 1.003× | 1.003–1.004 | 1.001× | 0.5 | 14.9 |
| HQC | hqc-3 | encapsulation | 6,512 | 6,520 | 6,548 | 1.006× | 1.005–1.006 | 1.001× | 0.5 | 16.4 |
| HQC | hqc-3 | key generation | 3,272 | 3,278 | 3,292 | 1.006× | 1.006–1.007 | 1.002× | 0.5 | 11.9 |
| HQC | hqc-5 | decapsulation | 23,943 | 23,957 | 24,004 | 1.002× | 1.002–1.003 | 1.001× | 0.5 | 24.3 |
| HQC | hqc-5 | encapsulation | 15,901 | 15,905 | 15,954 | 1.004× | 1.003–1.004 | 1.001× | 0.5 | 27.6 |
| HQC | hqc-5 | key generation | 7,982 | 7,994 | 8,020 | 1.004× | 1.004–1.005 | 1.002× | 0.5 | 20.1 |
| classical | dhkem-p256 | decapsulation | 117 | 142 | 143 | 1.222× | 1.222–1.225 | 1.215× | 23.9 | 0.4 |
| classical | dhkem-p256 | encapsulation | 132 | 157 | 158 | 1.200× | 1.199–1.202 | 1.192× | 23.0 | 0.3 |
| classical | dhkem-p256 | key generation | 37.5 | – | 37.8 | 1.010× | 1.009–1.011 | – | 0.0 | 0.2 |
| classical | dhkem-x25519 | decapsulation | 39.1 | 77.3 | 77.9 | 1.986× | 1.984–1.998 | 1.971× | 37.6 | 0.3 |
| classical | dhkem-x25519 | encapsulation | 78.7 | 83.3 | 83.9 | 1.067× | 1.063–1.071 | 1.055× | 3.5 | 0.2 |
| classical | dhkem-x25519 | key generation | 40.1 | – | 40.4 | 1.007× | 1.006–1.008 | – | 0.0 | 0.1 |
| classical | ecdsa-p256 | key generation | 38.1 | – | 38.5 | 1.010× | 1.009–1.012 | – | 0.0 | 0.2 |
| classical | ecdsa-p256 | signing | 24.6 | 53.8 | 54.8 | 2.222× | 2.203–2.226 | 2.183× | 23.9 | 0.3 |
| classical | ecdsa-p256 | verification | 59.2 | 88.2 | 89.7 | 1.511× | 1.497–1.517 | 1.492× | 23.2 | 0.2 |
| classical | ed25519 | key generation | 41.7 | – | 42.0 | 1.006× | 1.006–1.007 | – | 0.0 | 0.1 |
| classical | ed25519 | signing | 38.1 | 77.0 | 77.9 | 2.049× | 2.039–2.054 | 2.027× | 38.1 | 0.2 |
| classical | ed25519 | verification | 101 | 105 | 105 | 1.043× | 1.042–1.043 | 1.037× | 3.3 | 0.2 |
| classical | rsa3072-pss | key generation | 499,458 | – | 507,452 | 0.991× | 0.844–1.152 | – | 0.0 | 3.1 |
| classical | rsa3072-pss | signing | 1,527 | 2,710 | 2,717 | 1.779× | 1.774–1.782 | 1.776× | 485 | 2.9 |
| classical | rsa3072-pss | verification | 37.5 | 180 | 182 | 4.867× | 4.808–4.876 | 4.806× | 124 | 1.3 |

Generated by `analysis/paper_tables.py` from `results/combined/analysis_tables/T3_E2_integration_overhead.csv`.
