# E05b  Native macOS against the container, every operation (µs)

Medians over five repetitions; 95 % interval of the ratio.

| Family | Algorithm | Operation | Container | Native | Native ÷ container | 95 % CI |
|---|---|---|---:|---:|---:|---|
| ML-DSA | ml-dsa-44 | key generation | 30.2 | 30.3 | 1.002× | 0.732–1.027 |
| ML-DSA | ml-dsa-44 | signing | 60.2 | 60.3 | 1.001× | 0.990–1.019 |
| ML-DSA | ml-dsa-44 | verification | 28.4 | 27.5 | 0.970× | 0.653–0.990 |
| ML-DSA | ml-dsa-65 | key generation | 53.0 | 54.8 | 1.033× | 1.028–1.049 |
| ML-DSA | ml-dsa-65 | signing | 95.5 | 99.1 | 1.038× | 1.010–1.045 |
| ML-DSA | ml-dsa-65 | verification | 47.6 | 49.7 | 1.043× | 1.020–1.053 |
| ML-DSA | ml-dsa-87 | key generation | 90.9 | 94.0 | 1.034× | 1.031–1.057 |
| ML-DSA | ml-dsa-87 | signing | 140 | 142 | 1.018× | 1.012–1.033 |
| ML-DSA | ml-dsa-87 | verification | 83.5 | 87.8 | 1.050× | 1.036–1.055 |
| Falcon | falcon-1024 | key generation | 14,858 | 13,990 | 0.942× | 0.938–0.947 |
| Falcon | falcon-1024 | signing | 290 | 311 | 1.072× | 1.061–1.084 |
| Falcon | falcon-1024 | verification | 42.6 | 55.3 | 1.299× | 1.134–1.321 |
| Falcon | falcon-512 | key generation | 4,802 | 4,487 | 0.934× | 0.933–0.936 |
| Falcon | falcon-512 | signing | 145 | 159 | 1.094× | 1.083–1.099 |
| Falcon | falcon-512 | verification | 21.7 | 26.5 | 1.224× | 1.207–1.235 |
| Falcon | falcon-padded-1024 | key generation | 14,814 | 13,932 | 0.940× | 0.936–0.947 |
| Falcon | falcon-padded-1024 | signing | 287 | 310 | 1.078× | 1.061–1.088 |
| Falcon | falcon-padded-1024 | verification | 41.9 | 55.4 | 1.323× | 1.300–1.324 |
| Falcon | falcon-padded-512 | key generation | 4,803 | 4,496 | 0.936× | 0.928–0.937 |
| Falcon | falcon-padded-512 | signing | 145 | 159 | 1.099× | 1.096–1.101 |
| Falcon | falcon-padded-512 | verification | 21.8 | 26.5 | 1.217× | 1.201–1.232 |
| SLH-DSA | slh-dsa-sha2-128f | key generation | 888 | 879 | 0.990× | 0.989–1.003 |
| SLH-DSA | slh-dsa-sha2-128f | signing | 20,748 | 20,385 | 0.982× | 0.982–0.983 |
| SLH-DSA | slh-dsa-sha2-128f | verification | 1,252 | 1,244 | 0.993× | 0.989–0.998 |
| SLH-DSA | slh-dsa-sha2-128s | key generation | 56,810 | 55,697 | 0.980× | 0.980–0.981 |
| SLH-DSA | slh-dsa-sha2-128s | signing | 431,348 | 424,554 | 0.984× | 0.983–0.985 |
| SLH-DSA | slh-dsa-sha2-128s | verification | 428 | 430 | 1.006× | 0.991–1.013 |
| SLH-DSA | slh-dsa-sha2-192f | key generation | 1,298 | 1,274 | 0.982× | 0.981–0.982 |
| SLH-DSA | slh-dsa-sha2-192f | signing | 33,802 | 33,336 | 0.986× | 0.985–0.992 |
| SLH-DSA | slh-dsa-sha2-192f | verification | 1,823 | 1,804 | 0.989× | 0.985–0.997 |
| SLH-DSA | slh-dsa-sha2-192s | key generation | 82,750 | 80,987 | 0.979× | 0.979–0.979 |
| SLH-DSA | slh-dsa-sha2-192s | signing | 757,581 | 749,501 | 0.989× | 0.988–0.990 |
| SLH-DSA | slh-dsa-sha2-192s | verification | 633 | 633 | 0.999× | 0.991–1.007 |
| SLH-DSA | slh-dsa-sha2-256f | key generation | 3,423 | 3,354 | 0.980× | 0.979–0.996 |
| SLH-DSA | slh-dsa-sha2-256f | signing | 69,469 | 68,484 | 0.986× | 0.985–0.987 |
| SLH-DSA | slh-dsa-sha2-256f | verification | 1,870 | 1,844 | 0.986× | 0.984–0.990 |
| SLH-DSA | slh-dsa-sha2-256s | key generation | 54,656 | 53,505 | 0.979× | 0.979–0.979 |
| SLH-DSA | slh-dsa-sha2-256s | signing | 667,755 | 664,592 | 0.995× | 0.991–0.996 |
| SLH-DSA | slh-dsa-sha2-256s | verification | 928 | 923 | 0.995× | 0.990–1.006 |
| SLH-DSA | slh-dsa-shake-128f | key generation | 1,120 | 1,078 | 0.962× | 0.949–0.969 |
| SLH-DSA | slh-dsa-shake-128f | signing | 26,152 | 24,890 | 0.952× | 0.938–0.969 |
| SLH-DSA | slh-dsa-shake-128f | verification | 1,558 | 1,498 | 0.961× | 0.694–0.969 |
| SLH-DSA | slh-dsa-shake-128s | key generation | 70,822 | 68,727 | 0.970× | 0.951–0.971 |
| SLH-DSA | slh-dsa-shake-128s | signing | 552,782 | 520,523 | 0.942× | 0.880–0.954 |
| SLH-DSA | slh-dsa-shake-128s | verification | 539 | 521 | 0.967× | 0.961–0.977 |
| SLH-DSA | slh-dsa-shake-192f | key generation | 1,641 | 1,564 | 0.953× | 0.950–0.967 |
| SLH-DSA | slh-dsa-shake-192f | signing | 43,220 | 41,470 | 0.960× | 0.949–0.973 |
| SLH-DSA | slh-dsa-shake-192f | verification | 2,285 | 2,205 | 0.965× | 0.914–0.971 |
| SLH-DSA | slh-dsa-shake-192s | key generation | 104,123 | 99,386 | 0.955× | 0.906–0.973 |
| SLH-DSA | slh-dsa-shake-192s | signing | 982,146 | 934,120 | 0.951× | 0.936–0.965 |
| SLH-DSA | slh-dsa-shake-192s | verification | 786 | 760 | 0.967× | 0.955–0.978 |
| SLH-DSA | slh-dsa-shake-256f | key generation | 4,345 | 4,159 | 0.957× | 0.891–0.970 |
| SLH-DSA | slh-dsa-shake-256f | signing | 89,331 | 85,817 | 0.961× | 0.947–0.975 |
| SLH-DSA | slh-dsa-shake-256f | verification | 2,335 | 2,247 | 0.962× | 0.952–0.973 |
| SLH-DSA | slh-dsa-shake-256s | key generation | 68,979 | 65,884 | 0.955× | 0.947–0.976 |
| SLH-DSA | slh-dsa-shake-256s | signing | 886,612 | 849,129 | 0.958× | 0.622–0.971 |
| SLH-DSA | slh-dsa-shake-256s | verification | 1,163 | 1,119 | 0.962× | 0.934–0.976 |
| ML-KEM | ml-kem-1024 | decapsulation | 23.7 | 23.6 | 0.999× | 0.978–1.052 |
| ML-KEM | ml-kem-1024 | encapsulation | 21.0 | 20.8 | 0.993× | 0.971–1.036 |
| ML-KEM | ml-kem-1024 | key generation | 20.2 | 19.6 | 0.972× | 0.910–0.992 |
| ML-KEM | ml-kem-512 | decapsulation | 10.1 | 10.3 | 1.017× | 0.996–1.103 |
| ML-KEM | ml-kem-512 | encapsulation | 9.1 | 8.9 | 0.973× | 0.903–1.083 |
| ML-KEM | ml-kem-512 | key generation | 8.3 | 8.0 | 0.966× | 0.946–0.976 |
| ML-KEM | ml-kem-768 | decapsulation | 16.1 | 16.0 | 0.998× | 0.970–1.034 |
| ML-KEM | ml-kem-768 | encapsulation | 14.0 | 13.9 | 0.989× | 0.974–1.051 |
| ML-KEM | ml-kem-768 | key generation | 13.3 | 13.0 | 0.979× | 0.964–0.997 |
| HQC | hqc-1 | decapsulation | 2,728 | 2,384 | 0.874× | 0.874–0.874 |
| HQC | hqc-1 | encapsulation | 1,791 | 1,564 | 0.874× | 0.873–0.874 |
| HQC | hqc-1 | key generation | 894 | 787 | 0.880× | 0.874–0.890 |
| HQC | hqc-3 | decapsulation | 7,045 | 7,139 | 1.013× | 1.013–1.014 |
| HQC | hqc-3 | encapsulation | 4,668 | 4,714 | 1.010× | 1.007–1.010 |
| HQC | hqc-3 | key generation | 2,345 | 2,373 | 1.012× | 1.011–1.014 |
| HQC | hqc-5 | decapsulation | 17,809 | 17,361 | 0.975× | 0.974–0.976 |
| HQC | hqc-5 | encapsulation | 11,813 | 11,509 | 0.974× | 0.973–0.974 |
| HQC | hqc-5 | key generation | 5,922 | 5,762 | 0.973× | 0.972–0.975 |
| classical | dhkem-p256 | decapsulation | 99.5 | 102 | 1.024× | 1.005–1.025 |
| classical | dhkem-p256 | encapsulation | 110 | 112 | 1.015× | 0.996–1.019 |
| classical | dhkem-p256 | key generation | 26.0 | 22.0 | 0.845× | 0.714–0.866 |
| classical | dhkem-x25519 | decapsulation | 37.5 | 35.6 | 0.949× | 0.948–0.961 |
| classical | dhkem-x25519 | encapsulation | 67.4 | 64.5 | 0.957× | 0.954–0.958 |
| classical | dhkem-x25519 | key generation | 31.9 | 29.5 | 0.922× | 0.908–0.944 |
| classical | ecdsa-p256 | key generation | 25.9 | 22.1 | 0.856× | 0.838–0.862 |
| classical | ecdsa-p256 | signing | 19.9 | 20.6 | 1.032× | 1.030–1.034 |
| classical | ecdsa-p256 | verification | 54.5 | 57.8 | 1.060× | 1.057–1.067 |
| classical | ed25519 | key generation | 32.1 | 29.7 | 0.923× | 0.906–0.940 |
| classical | ed25519 | signing | 30.3 | 29.7 | 0.980× | 0.977–0.981 |
| classical | ed25519 | verification | 81.6 | 88.4 | 1.083× | 1.080–1.084 |
| classical | rsa3072-pss | key generation | 401,964 | 362,858 | 0.903× | 0.783–0.967 |
| classical | rsa3072-pss | signing | 1,605 | 1,585 | 0.988× | 0.972–0.992 |
| classical | rsa3072-pss | verification | 33.0 | 33.8 | 1.024× | 1.007–1.036 |

Generated by `analysis/paper_tables.py` from `results/combined/analysis_tables/T9_E5_native_vs_container.csv`.
