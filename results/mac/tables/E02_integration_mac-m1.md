# E02  QChain's deployed pattern against a reused key on the Mac M1 (Apple M1, ARM with NEON) (µs)

Reuse: one key, one library object. Re-init: a new library object per call. Production: QChain's code (hex keys decoded per call). Components: library-object initialisation and hex conversion, timed alone.

| Family | Algorithm | Operation | Reuse | Re-init | Production | Production ÷ reuse | 95 % CI | Re-init ÷ reuse | Initialisation | Hex conversion |
|---|---|---|---:|---:|---:|---:|---|---:|---:|---:|
| ML-DSA | ml-dsa-44 | key generation | 30.7 | 31.5 | 35.5 | 1.159× | 1.156–1.160 | 1.026× | 0.2 | 4.1 |
| ML-DSA | ml-dsa-44 | signing | 60.4 | 61.4 | 67.3 | 1.116× | 1.101–1.127 | 1.017× | 0.2 | 5.0 |
| ML-DSA | ml-dsa-44 | verification | 28.7 | 29.0 | 32.8 | 1.145× | 1.143–1.146 | 1.013× | 0.2 | 3.8 |
| ML-DSA | ml-dsa-65 | key generation | 53.3 | 54.5 | 60.9 | 1.144× | 1.142–1.145 | 1.024× | 0.2 | 6.2 |
| ML-DSA | ml-dsa-65 | signing | 97.6 | 98.1 | 108 | 1.115× | 1.095–1.122 | 1.010× | 0.2 | 7.3 |
| ML-DSA | ml-dsa-65 | verification | 48.9 | 49.3 | 54.6 | 1.116× | 1.082–1.125 | 1.008× | 0.2 | 5.2 |
| ML-DSA | ml-dsa-87 | key generation | 93.1 | 94.7 | 102 | 1.102× | 1.100–1.103 | 1.017× | 0.3 | 7.7 |
| ML-DSA | ml-dsa-87 | signing | 140 | 142 | 161 | 1.152× | 1.126–1.155 | 1.012× | 0.3 | 9.4 |
| ML-DSA | ml-dsa-87 | verification | 86.7 | 87.2 | 95.1 | 1.091× | 1.089–1.097 | 1.006× | 0.3 | 7.2 |
| Falcon | falcon-1024 | key generation | 14,783 | 14,842 | 14,822 | 1.001× | 0.998–1.007 | 1.001× | 0.3 | 4.3 |
| Falcon | falcon-1024 | signing | 292 | 293 | 298 | 1.020× | 1.019–1.020 | 1.003× | 0.3 | 3.6 |
| Falcon | falcon-1024 | verification | 40.7 | 41.2 | 44.4 | 1.089× | 1.088–1.091 | 1.011× | 0.3 | 3.1 |
| Falcon | falcon-512 | key generation | 4,801 | 4,813 | 4,814 | 1.003× | 1.001–1.012 | 1.003× | 0.3 | 2.3 |
| Falcon | falcon-512 | signing | 147 | 148 | 150 | 1.020× | 1.020–1.021 | 1.005× | 0.3 | 2.0 |
| Falcon | falcon-512 | verification | 20.8 | 21.2 | 22.9 | 1.100× | 1.100–1.102 | 1.020× | 0.3 | 1.6 |
| Falcon | falcon-padded-1024 | key generation | 14,778 | 14,785 | 14,790 | 1.001× | 0.998–1.006 | 1.000× | 0.3 | 4.3 |
| Falcon | falcon-padded-1024 | signing | 289 | 290 | 295 | 1.019× | 1.019–1.020 | 1.003× | 0.3 | 3.6 |
| Falcon | falcon-padded-1024 | verification | 40.8 | 41.4 | 44.6 | 1.090× | 1.089–1.093 | 1.012× | 0.3 | 3.1 |
| Falcon | falcon-padded-512 | key generation | 4,795 | 4,810 | 4,815 | 1.004× | 1.001–1.005 | 1.003× | 0.3 | 2.3 |
| Falcon | falcon-padded-512 | signing | 146 | 147 | 149 | 1.020× | 1.019–1.021 | 1.005× | 0.3 | 2.0 |
| Falcon | falcon-padded-512 | verification | 20.5 | 20.9 | 22.5 | 1.100× | 1.098–1.102 | 1.020× | 0.3 | 1.6 |
| SLH-DSA | slh-dsa-sha2-128f | key generation | 888 | 889 | 892 | 1.005× | 1.004–1.006 | 1.001× | 0.5 | 0.1 |
| SLH-DSA | slh-dsa-sha2-128f | signing | 20,741 | 20,743 | 20,783 | 1.002× | 1.002–1.002 | 1.000× | 0.5 | 17.6 |
| SLH-DSA | slh-dsa-sha2-128f | verification | 1,257 | 1,258 | 1,282 | 1.020× | 1.018–1.020 | 1.001× | 0.5 | 16.7 |
| SLH-DSA | slh-dsa-sha2-128s | key generation | 56,811 | 56,817 | 56,819 | 1.000× | 1.000–1.000 | 1.000× | 0.5 | 0.1 |
| SLH-DSA | slh-dsa-sha2-128s | signing | 431,471 | 431,439 | 431,444 | 1.000× | 1.000–1.000 | 1.000× | 0.5 | 7.9 |
| SLH-DSA | slh-dsa-sha2-128s | verification | 430 | 431 | 441 | 1.023× | 1.022–1.024 | 1.002× | 0.5 | 7.8 |
| SLH-DSA | slh-dsa-sha2-192f | key generation | 1,298 | 1,299 | 1,300 | 1.001× | 1.001–1.001 | 1.001× | 0.5 | 0.2 |
| SLH-DSA | slh-dsa-sha2-192f | signing | 33,798 | 33,797 | 33,860 | 1.002× | 1.002–1.002 | 1.000× | 0.5 | 36.3 |
| SLH-DSA | slh-dsa-sha2-192f | verification | 1,832 | 1,834 | 1,874 | 1.022× | 1.022–1.023 | 1.000× | 0.5 | 35.2 |
| SLH-DSA | slh-dsa-sha2-192s | key generation | 82,763 | 82,767 | 82,774 | 1.000× | 1.000–1.000 | 1.000× | 0.5 | 0.2 |
| SLH-DSA | slh-dsa-sha2-192s | signing | 757,933 | 758,078 | 757,978 | 1.000× | 1.000–1.000 | 1.000× | 0.5 | 16.3 |
| SLH-DSA | slh-dsa-sha2-192s | verification | 634 | 635 | 654 | 1.032× | 1.032–1.033 | 1.002× | 0.5 | 15.9 |
| SLH-DSA | slh-dsa-sha2-256f | key generation | 3,423 | 3,424 | 3,427 | 1.001× | 1.001–1.002 | 1.000× | 0.6 | 0.2 |
| SLH-DSA | slh-dsa-sha2-256f | signing | 69,555 | 69,565 | 69,653 | 1.001× | 1.001–1.002 | 1.000× | 0.6 | 50.4 |
| SLH-DSA | slh-dsa-sha2-256f | verification | 1,873 | 1,873 | 1,924 | 1.028× | 1.027–1.028 | 1.000× | 0.6 | 49.1 |
| SLH-DSA | slh-dsa-sha2-256s | key generation | 54,658 | 54,665 | 54,667 | 1.000× | 1.000–1.000 | 1.000× | 0.5 | 0.2 |
| SLH-DSA | slh-dsa-sha2-256s | signing | 668,661 | 668,592 | 668,715 | 1.000× | 1.000–1.000 | 1.000× | 0.5 | 30.2 |
| SLH-DSA | slh-dsa-sha2-256s | verification | 930 | 931 | 976 | 1.049× | 1.047–1.049 | 1.001× | 0.6 | 29.4 |
| SLH-DSA | slh-dsa-shake-128f | key generation | 1,120 | 1,122 | 1,124 | 1.003× | 1.002–1.003 | 1.001× | 0.6 | 0.1 |
| SLH-DSA | slh-dsa-shake-128f | signing | 26,488 | 26,490 | 26,534 | 1.002× | 1.001–1.002 | 1.000× | 0.6 | 17.4 |
| SLH-DSA | slh-dsa-shake-128f | verification | 1,571 | 1,572 | 1,595 | 1.016× | 1.015–1.016 | 1.001× | 0.6 | 16.8 |
| SLH-DSA | slh-dsa-shake-128s | key generation | 71,862 | 71,893 | 71,867 | 1.000× | 1.000–1.000 | 1.000× | 0.6 | 0.1 |
| SLH-DSA | slh-dsa-shake-128s | signing | 549,142 | 548,191 | 549,846 | 1.000× | 0.997–1.005 | 1.000× | 0.6 | 7.9 |
| SLH-DSA | slh-dsa-shake-128s | verification | 544 | 545 | 556 | 1.020× | 1.019–1.022 | 1.002× | 0.6 | 7.8 |
| SLH-DSA | slh-dsa-shake-192f | key generation | 1,652 | 1,653 | 1,655 | 1.002× | 1.002–1.002 | 1.001× | 0.6 | 0.2 |
| SLH-DSA | slh-dsa-shake-192f | signing | 43,548 | 43,559 | 43,618 | 1.002× | 1.001–1.002 | 1.000× | 0.6 | 36.2 |
| SLH-DSA | slh-dsa-shake-192f | verification | 2,301 | 2,302 | 2,342 | 1.018× | 1.017–1.018 | 1.001× | 0.6 | 35.2 |
| SLH-DSA | slh-dsa-shake-192s | key generation | 104,653 | 104,792 | 104,723 | 1.000× | 0.997–1.001 | 1.000× | 0.6 | 0.2 |
| SLH-DSA | slh-dsa-shake-192s | signing | 986,579 | 986,772 | 986,092 | 1.000× | 0.999–1.003 | 1.000× | 0.6 | 16.3 |
| SLH-DSA | slh-dsa-shake-192s | verification | 794 | 795 | 814 | 1.028× | 1.025–1.032 | 1.002× | 0.6 | 15.9 |
| SLH-DSA | slh-dsa-shake-256f | key generation | 4,347 | 4,350 | 4,353 | 1.001× | 1.001–1.001 | 1.000× | 0.7 | 0.2 |
| SLH-DSA | slh-dsa-shake-256f | signing | 89,736 | 89,773 | 89,706 | 1.001× | 1.000–1.002 | 1.000× | 0.7 | 50.1 |
| SLH-DSA | slh-dsa-shake-256f | verification | 2,385 | 2,386 | 2,438 | 1.022× | 1.012–1.023 | 1.000× | 0.7 | 48.8 |
| SLH-DSA | slh-dsa-shake-256s | key generation | 69,266 | 69,286 | 69,255 | 1.000× | 0.999–1.000 | 1.000× | 0.7 | 0.2 |
| SLH-DSA | slh-dsa-shake-256s | signing | 884,324 | 885,240 | 884,707 | 1.000× | 1.000–1.002 | 1.000× | 0.7 | 30.2 |
| SLH-DSA | slh-dsa-shake-256s | verification | 1,167 | 1,169 | 1,215 | 1.041× | 1.038–1.042 | 1.001× | 0.7 | 29.4 |
| ML-KEM | ml-kem-1024 | decapsulation | 24.0 | 24.7 | 29.4 | 1.224× | 1.216–1.226 | 1.027× | 0.3 | 4.8 |
| ML-KEM | ml-kem-1024 | encapsulation | 21.4 | 21.9 | 25.2 | 1.179× | 1.174–1.184 | 1.023× | 0.3 | 3.2 |
| ML-KEM | ml-kem-1024 | key generation | 20.2 | 21.2 | 26.2 | 1.295× | 1.291–1.298 | 1.050× | 0.3 | 4.7 |
| ML-KEM | ml-kem-512 | decapsulation | 10.2 | 10.8 | 13.2 | 1.290× | 1.286–1.293 | 1.053× | 0.3 | 2.4 |
| ML-KEM | ml-kem-512 | encapsulation | 9.2 | 9.6 | 11.3 | 1.238× | 1.236–1.239 | 1.050× | 0.3 | 1.7 |
| ML-KEM | ml-kem-512 | key generation | 8.5 | 9.2 | 11.6 | 1.363× | 1.363–1.365 | 1.080× | 0.3 | 2.5 |
| ML-KEM | ml-kem-768 | decapsulation | 16.0 | 16.6 | 20.1 | 1.257× | 1.255–1.265 | 1.036× | 0.3 | 3.6 |
| ML-KEM | ml-kem-768 | encapsulation | 14.2 | 14.7 | 17.1 | 1.202× | 1.201–1.205 | 1.032× | 0.3 | 2.4 |
| ML-KEM | ml-kem-768 | key generation | 13.6 | 14.4 | 18.0 | 1.327× | 1.321–1.328 | 1.060× | 0.3 | 3.7 |
| HQC | hqc-1 | decapsulation | 2,730 | 2,732 | 2,753 | 1.008× | 1.008–1.009 | 1.001× | 0.3 | 6.8 |
| HQC | hqc-1 | encapsulation | 1,791 | 1,793 | 1,817 | 1.015× | 1.013–1.015 | 1.001× | 0.3 | 6.7 |
| HQC | hqc-1 | key generation | 896 | 914 | 920 | 1.027× | 1.025–1.029 | 1.020× | 0.3 | 4.7 |
| HQC | hqc-3 | decapsulation | 7,047 | 7,049 | 7,072 | 1.004× | 1.004–1.004 | 1.000× | 0.3 | 13.2 |
| HQC | hqc-3 | encapsulation | 4,671 | 4,673 | 4,702 | 1.007× | 1.006–1.007 | 1.000× | 0.3 | 13.4 |
| HQC | hqc-3 | key generation | 2,332 | 2,353 | 2,362 | 1.013× | 1.012–1.013 | 1.009× | 0.3 | 9.0 |
| HQC | hqc-5 | decapsulation | 17,797 | 17,801 | 17,837 | 1.002× | 1.002–1.002 | 1.000× | 0.3 | 21.7 |
| HQC | hqc-5 | encapsulation | 11,795 | 11,796 | 11,835 | 1.004× | 1.004–1.004 | 1.000× | 0.3 | 21.9 |
| HQC | hqc-5 | key generation | 5,907 | 5,928 | 5,946 | 1.006× | 1.006–1.007 | 1.004× | 0.3 | 15.2 |
| classical | dhkem-p256 | decapsulation | 100 | 115 | 115 | 1.152× | 1.151–1.153 | 1.149× | 14.5 | 0.3 |
| classical | dhkem-p256 | encapsulation | 113 | 128 | 128 | 1.136× | 1.130–1.136 | 1.133× | 14.2 | 0.2 |
| classical | dhkem-p256 | key generation | 25.6 | – | 25.7 | 1.005× | 1.005–1.005 | – | 0.0 | 0.1 |
| classical | dhkem-x25519 | decapsulation | 37.7 | 68.1 | 68.4 | 1.814× | 1.810–1.831 | 1.807× | 30.2 | 0.2 |
| classical | dhkem-x25519 | encapsulation | 67.4 | 69.8 | 70.0 | 1.040× | 1.039–1.040 | 1.036× | 2.1 | 0.2 |
| classical | dhkem-x25519 | key generation | 31.4 | – | 31.5 | 1.003× | 1.001–1.004 | – | 0.0 | 0.1 |
| classical | ecdsa-p256 | key generation | 26.4 | – | 26.5 | 1.006× | 1.006–1.008 | – | 0.0 | 0.1 |
| classical | ecdsa-p256 | signing | 20.5 | 36.3 | 36.7 | 1.786× | 1.748–1.787 | 1.768× | 14.4 | 0.2 |
| classical | ecdsa-p256 | verification | 55.7 | 70.7 | 71.0 | 1.273× | 1.272–1.277 | 1.269× | 14.2 | 0.2 |
| classical | ed25519 | key generation | 32.5 | – | 32.6 | 1.004× | 1.003–1.004 | – | 0.0 | 0.1 |
| classical | ed25519 | signing | 30.4 | 61.1 | 61.4 | 2.011× | 2.007–2.015 | 2.001× | 30.9 | 0.1 |
| classical | ed25519 | verification | 81.5 | 83.5 | 83.7 | 1.028× | 1.027–1.028 | 1.025× | 1.9 | 0.1 |
| classical | rsa3072-pss | key generation | 437,695 | – | 451,661 | 0.996× | 0.912–1.187 | – | 0.0 | 2.3 |
| classical | rsa3072-pss | signing | 1,613 | 2,455 | 2,466 | 1.530× | 1.523–1.531 | 1.519× | 291 | 2.3 |
| classical | rsa3072-pss | verification | 33.7 | 127 | 128 | 3.788× | 3.768–3.812 | 3.754× | 78.3 | 0.9 |

Generated by `analysis/paper_tables.py` from `results/combined/analysis_tables/T3_E2_integration_overhead.csv`.
