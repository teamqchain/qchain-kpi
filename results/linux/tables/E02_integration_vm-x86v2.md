# E02  QChain's deployed pattern against a reused key on the VM (virtualised x86 server without AVX2) (µs)

Reuse: one key, one library object. Re-init: a new library object per call. Production: QChain's code (hex keys decoded per call). Components: library-object initialisation and hex conversion, timed alone.

| Family | Algorithm | Operation | Reuse | Re-init | Production | Production ÷ reuse | 95 % CI | Re-init ÷ reuse | Initialisation | Hex conversion |
|---|---|---|---:|---:|---:|---:|---|---:|---:|---:|
| ML-DSA | ml-dsa-44 | key generation | 80.2 | 82.2 | 87.6 | 1.093× | 1.092–1.108 | 1.026× | 0.3 | 4.4 |
| ML-DSA | ml-dsa-44 | signing | 207 | 211 | 217 | 1.050× | 1.033–1.057 | 1.019× | 0.3 | 5.1 |
| ML-DSA | ml-dsa-44 | verification | 78.5 | 79.6 | 84.1 | 1.071× | 1.071–1.073 | 1.016× | 0.3 | 3.8 |
| ML-DSA | ml-dsa-65 | key generation | 133 | 136 | 145 | 1.087× | 1.085–1.087 | 1.018× | 0.3 | 6.9 |
| ML-DSA | ml-dsa-65 | signing | 323 | 324 | 336 | 1.044× | 1.034–1.057 | 1.014× | 0.3 | 7.4 |
| ML-DSA | ml-dsa-65 | verification | 115 | 115 | 121 | 1.059× | 1.058–1.083 | 1.006× | 0.3 | 5.4 |
| ML-DSA | ml-dsa-87 | key generation | 188 | 191 | 200 | 1.066× | 1.065–1.073 | 1.014× | 0.3 | 8.2 |
| ML-DSA | ml-dsa-87 | signing | 416 | 417 | 427 | 1.027× | 1.023–1.034 | 1.000× | 0.3 | 9.5 |
| ML-DSA | ml-dsa-87 | verification | 183 | 183 | 191 | 1.049× | 1.047–1.050 | 1.004× | 0.3 | 7.4 |
| Falcon | falcon-1024 | key generation | 33,372 | 33,421 | 33,274 | 1.004× | 0.973–1.026 | 0.996× | 0.3 | 4.8 |
| Falcon | falcon-1024 | signing | 7,986 | 7,977 | 7,989 | 1.001× | 1.000–1.001 | 1.000× | 0.3 | 3.8 |
| Falcon | falcon-1024 | verification | 80.5 | 82.1 | 86.4 | 1.071× | 1.067–1.073 | 1.020× | 0.3 | 3.2 |
| Falcon | falcon-512 | key generation | 11,631 | 11,648 | 11,612 | 1.002× | 0.993–1.006 | 1.002× | 0.3 | 2.9 |
| Falcon | falcon-512 | signing | 3,654 | 3,656 | 3,660 | 1.001× | 1.000–1.002 | 1.000× | 0.3 | 2.1 |
| Falcon | falcon-512 | verification | 34.5 | 35.3 | 37.5 | 1.081× | 1.077–1.089 | 1.023× | 0.3 | 1.6 |
| Falcon | falcon-padded-1024 | key generation | 33,664 | 33,499 | 33,694 | 1.006× | 0.980–1.026 | 0.997× | 0.3 | 4.7 |
| Falcon | falcon-padded-1024 | signing | 7,955 | 7,967 | 7,980 | 1.002× | 1.001–1.003 | 1.001× | 0.3 | 3.6 |
| Falcon | falcon-padded-1024 | verification | 77.5 | 79.1 | 82.9 | 1.069× | 1.067–1.071 | 1.019× | 0.4 | 3.3 |
| Falcon | falcon-padded-512 | key generation | 11,449 | 11,596 | 11,599 | 1.011× | 1.003–1.018 | 1.006× | 0.3 | 2.8 |
| Falcon | falcon-padded-512 | signing | 3,657 | 3,658 | 3,662 | 1.001× | 1.001–1.002 | 1.001× | 0.3 | 2.0 |
| Falcon | falcon-padded-512 | verification | 34.5 | 35.2 | 37.2 | 1.079× | 1.075–1.086 | 1.020× | 0.3 | 1.6 |
| ML-KEM | ml-kem-1024 | decapsulation | 58.7 | 59.7 | 65.6 | 1.114× | 1.112–1.117 | 1.017× | 0.4 | 4.8 |
| ML-KEM | ml-kem-1024 | encapsulation | 54.4 | 55.7 | 60.4 | 1.111× | 1.107–1.123 | 1.024× | 0.5 | 3.4 |
| ML-KEM | ml-kem-1024 | key generation | 54.9 | 57.1 | 63.8 | 1.161× | 1.159–1.168 | 1.040× | 0.5 | 5.2 |
| ML-KEM | ml-kem-512 | decapsulation | 26.3 | 27.2 | 30.2 | 1.137× | 1.127–1.153 | 1.033× | 0.4 | 2.4 |
| ML-KEM | ml-kem-512 | encapsulation | 25.8 | 26.9 | 29.2 | 1.142× | 1.129–1.144 | 1.041× | 0.4 | 1.9 |
| ML-KEM | ml-kem-512 | key generation | 23.2 | 24.9 | 28.4 | 1.220× | 1.203–1.223 | 1.074× | 0.4 | 2.9 |
| ML-KEM | ml-kem-768 | decapsulation | 41.9 | 42.8 | 47.0 | 1.124× | 1.121–1.138 | 1.023× | 0.5 | 3.7 |
| ML-KEM | ml-kem-768 | encapsulation | 38.6 | 39.7 | 43.1 | 1.118× | 1.113–1.127 | 1.031× | 0.4 | 2.5 |
| ML-KEM | ml-kem-768 | key generation | 36.8 | 38.9 | 43.8 | 1.188× | 1.180–1.190 | 1.054× | 0.4 | 4.2 |
| HQC | hqc-1 | decapsulation | 3,716 | 3,720 | 3,729 | 1.002× | 1.002–1.004 | 1.000× | 0.4 | 6.9 |
| HQC | hqc-1 | encapsulation | 2,444 | 2,442 | 2,450 | 1.005× | 1.003–1.006 | 1.000× | 0.4 | 7.4 |
| HQC | hqc-1 | key generation | 1,229 | 1,233 | 1,239 | 1.008× | 1.007–1.009 | 1.002× | 0.4 | 4.9 |
| HQC | hqc-3 | decapsulation | 11,105 | 11,129 | 11,128 | 1.002× | 1.002–1.003 | 1.001× | 0.4 | 13.6 |
| HQC | hqc-3 | encapsulation | 7,265 | 7,267 | 7,291 | 1.004× | 1.003–1.004 | 1.000× | 0.4 | 17.0 |
| HQC | hqc-3 | key generation | 3,653 | 3,656 | 3,668 | 1.004× | 1.004–1.005 | 1.001× | 0.4 | 10.0 |
| HQC | hqc-5 | decapsulation | 27,062 | 27,044 | 27,115 | 1.002× | 0.999–1.003 | 1.001× | 0.4 | 22.9 |
| HQC | hqc-5 | encapsulation | 17,739 | 17,733 | 17,776 | 1.002× | 1.002–1.003 | 1.000× | 0.4 | 31.1 |
| HQC | hqc-5 | key generation | 8,849 | 8,854 | 8,877 | 1.003× | 1.002–1.004 | 1.001× | 0.4 | 17.9 |
| classical | dhkem-p256 | decapsulation | 127 | 148 | 149 | 1.176× | 1.175–1.177 | 1.172× | 21.0 | 0.3 |
| classical | dhkem-p256 | encapsulation | 144 | 165 | 166 | 1.154× | 1.151–1.157 | 1.149× | 19.8 | 0.3 |
| classical | dhkem-p256 | key generation | 39.8 | – | 40.1 | 1.009× | 1.008–1.011 | – | 0.0 | 0.2 |
| classical | dhkem-x25519 | decapsulation | 44.7 | 85.7 | 86.0 | 1.904× | 1.891–1.931 | 1.894× | 39.6 | 0.2 |
| classical | dhkem-x25519 | encapsulation | 90.3 | 94.2 | 94.8 | 1.048× | 1.046–1.052 | 1.044× | 2.9 | 0.2 |
| classical | dhkem-x25519 | key generation | 44.6 | – | 44.9 | 1.005× | 1.005–1.006 | – | 0.0 | 0.1 |
| classical | ecdsa-p256 | key generation | 40.9 | – | 41.3 | 1.010× | 1.008–1.011 | – | 0.0 | 0.2 |
| classical | ecdsa-p256 | signing | 31.8 | 60.7 | 61.7 | 1.941× | 1.851–2.031 | 1.912× | 20.2 | 0.3 |
| classical | ecdsa-p256 | verification | 77.9 | 102 | 102 | 1.313× | 1.302–1.326 | 1.304× | 20.6 | 0.2 |
| classical | ed25519 | key generation | 44.6 | – | 44.9 | 1.006× | 1.006–1.007 | – | 0.0 | 0.1 |
| classical | ed25519 | signing | 40.6 | 81.7 | 82.5 | 2.031× | 2.006–2.048 | 2.012× | 40.0 | 0.2 |
| classical | ed25519 | verification | 101 | 104 | 104 | 1.035× | 1.033–1.039 | 1.032× | 2.6 | 0.1 |
| classical | rsa3072-pss | key generation | 533,062 | – | 510,892 | 0.944× | 0.842–1.105 | – | 0.0 | 2.5 |
| classical | rsa3072-pss | signing | 2,170 | 3,178 | 3,181 | 1.468× | 1.461–1.489 | 1.466× | 375 | 2.7 |
| classical | rsa3072-pss | verification | 53.7 | 169 | 171 | 3.199× | 3.183–3.319 | 3.170× | 102 | 0.9 |

Generated by `analysis/paper_tables.py` from `results/combined/analysis_tables/T3_E2_integration_overhead.csv`.
