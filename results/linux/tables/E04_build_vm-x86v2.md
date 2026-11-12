# E04  liboqs build variants on the VM (virtualised x86 server without AVX2) (µs)

Production build (b1) against the variant(s) this machine ran; medians over five repetitions, 95 % interval of the ratio, Cliff's delta.

| Variant | Family | Algorithm | Operation | Production (b1) | Variant | Variant ÷ b1 | 95 % CI | Cliff's delta |
|---|---|---|---|---:|---:|---:|---|---:|
| no OpenSSL (b4) | ML-DSA | ml-dsa-44 | key generation | 77.3 | 72.8 | 0.941× | 0.872–0.998 | -0.92 |
| no OpenSSL (b4) | ML-DSA | ml-dsa-44 | signing | 206 | 207 | 1.006× | 0.954–1.035 | -0.04 |
| no OpenSSL (b4) | ML-DSA | ml-dsa-44 | verification | 78.7 | 79.4 | 1.009× | 0.998–1.090 | 0.68 |
| no OpenSSL (b4) | ML-DSA | ml-dsa-65 | key generation | 132 | 133 | 1.002× | 0.959–1.033 | 0.04 |
| no OpenSSL (b4) | ML-DSA | ml-dsa-65 | signing | 321 | 320 | 0.997× | 0.955–1.006 | -0.36 |
| no OpenSSL (b4) | ML-DSA | ml-dsa-65 | verification | 108 | 110 | 1.016× | 0.963–1.054 | 0.28 |
| no OpenSSL (b4) | ML-DSA | ml-dsa-87 | key generation | 183 | 184 | 1.002× | 0.952–1.023 | -0.04 |
| no OpenSSL (b4) | ML-DSA | ml-dsa-87 | signing | 415 | 412 | 0.993× | 0.982–1.024 | 0.12 |
| no OpenSSL (b4) | ML-DSA | ml-dsa-87 | verification | 184 | 182 | 0.992× | 0.964–1.041 | 0.20 |
| no OpenSSL (b4) | Falcon | falcon-1024 | key generation | 32,969 | 32,877 | 0.997× | 0.984–1.017 | -0.20 |
| no OpenSSL (b4) | Falcon | falcon-1024 | signing | 7,911 | 7,906 | 0.999× | 0.982–1.001 | -0.68 |
| no OpenSSL (b4) | Falcon | falcon-1024 | verification | 78.3 | 80.9 | 1.033× | 0.947–1.069 | 0.28 |
| no OpenSSL (b4) | Falcon | falcon-512 | key generation | 11,581 | 11,530 | 0.996× | 0.990–1.008 | -0.28 |
| no OpenSSL (b4) | Falcon | falcon-512 | signing | 3,667 | 3,646 | 0.994× | 0.991–1.002 | -0.76 |
| no OpenSSL (b4) | Falcon | falcon-512 | verification | 36.2 | 36.4 | 1.004× | 0.932–1.089 | 0.04 |
| no OpenSSL (b4) | Falcon | falcon-padded-1024 | key generation | 33,077 | 32,967 | 0.997× | 0.966–1.030 | -0.04 |
| no OpenSSL (b4) | Falcon | falcon-padded-1024 | signing | 7,931 | 7,923 | 0.999× | 0.965–1.030 | -0.28 |
| no OpenSSL (b4) | Falcon | falcon-padded-1024 | verification | 79.4 | 80.3 | 1.011× | 0.955–1.047 | 0.04 |
| no OpenSSL (b4) | Falcon | falcon-padded-512 | key generation | 11,508 | 11,562 | 1.005× | 0.945–1.018 | 0.04 |
| no OpenSSL (b4) | Falcon | falcon-padded-512 | signing | 3,655 | 3,639 | 0.996× | 0.969–1.000 | -0.68 |
| no OpenSSL (b4) | Falcon | falcon-padded-512 | verification | 38.0 | 36.8 | 0.969× | 0.947–1.002 | -0.76 |
| no OpenSSL (b4) | SLH-DSA | slh-dsa-sha2-128f | key generation | 981 | 974 | 0.993× | 0.932–1.014 | -0.36 |
| no OpenSSL (b4) | SLH-DSA | slh-dsa-sha2-128f | signing | 22,651 | 22,485 | 0.993× | 0.958–1.023 | -0.36 |
| no OpenSSL (b4) | SLH-DSA | slh-dsa-sha2-128f | verification | 1,394 | 1,376 | 0.987× | 0.977–1.022 | -0.60 |
| no OpenSSL (b4) | SLH-DSA | slh-dsa-sha2-128s | key generation | 61,058 | 61,420 | 1.006× | 0.993–1.024 | 0.52 |
| no OpenSSL (b4) | SLH-DSA | slh-dsa-sha2-128s | signing | 470,114 | 471,075 | 1.002× | 0.955–1.019 | -0.04 |
| no OpenSSL (b4) | SLH-DSA | slh-dsa-sha2-128s | verification | 471 | 480 | 1.020× | 0.990–1.044 | 0.76 |
| no OpenSSL (b4) | SLH-DSA | slh-dsa-sha2-192f | key generation | 1,404 | 1,420 | 1.011× | 0.996–1.015 | 0.60 |
| no OpenSSL (b4) | SLH-DSA | slh-dsa-sha2-192f | signing | 36,745 | 36,777 | 1.001× | 0.980–1.012 | -0.20 |
| no OpenSSL (b4) | SLH-DSA | slh-dsa-sha2-192f | verification | 1,996 | 2,028 | 1.016× | 1.002–1.026 | 1.00 |
| no OpenSSL (b4) | SLH-DSA | slh-dsa-sha2-192s | key generation | 90,012 | 88,513 | 0.983× | 0.967–1.006 | -0.52 |
| no OpenSSL (b4) | SLH-DSA | slh-dsa-sha2-192s | signing | 838,269 | 841,193 | 1.003× | 0.977–1.015 | 0.04 |
| no OpenSSL (b4) | SLH-DSA | slh-dsa-sha2-192s | verification | 724 | 721 | 0.995× | 0.962–1.026 | -0.28 |
| no OpenSSL (b4) | SLH-DSA | slh-dsa-sha2-256f | key generation | 3,674 | 3,706 | 1.009× | 0.998–1.016 | 0.76 |
| no OpenSSL (b4) | SLH-DSA | slh-dsa-sha2-256f | signing | 76,655 | 76,139 | 0.993× | 0.963–1.014 | -0.36 |
| no OpenSSL (b4) | SLH-DSA | slh-dsa-sha2-256f | verification | 2,067 | 2,061 | 0.997× | 0.969–1.009 | -0.04 |
| no OpenSSL (b4) | SLH-DSA | slh-dsa-sha2-256s | key generation | 58,677 | 59,018 | 1.006× | 0.987–1.027 | 0.20 |
| no OpenSSL (b4) | SLH-DSA | slh-dsa-sha2-256s | signing | 748,938 | 741,319 | 0.990× | 0.964–1.017 | -0.44 |
| no OpenSSL (b4) | SLH-DSA | slh-dsa-sha2-256s | verification | 1,042 | 1,020 | 0.978× | 0.969–0.990 | -1.00 |
| no OpenSSL (b4) | SLH-DSA | slh-dsa-shake-128f | key generation | 1,621 | 1,602 | 0.988× | 0.958–1.001 | -0.68 |
| no OpenSSL (b4) | SLH-DSA | slh-dsa-shake-128f | signing | 37,564 | 37,528 | 0.999× | 0.979–1.014 | -0.20 |
| no OpenSSL (b4) | SLH-DSA | slh-dsa-shake-128f | verification | 2,248 | 2,226 | 0.990× | 0.970–1.007 | -0.52 |
| no OpenSSL (b4) | SLH-DSA | slh-dsa-shake-128s | key generation | 102,508 | 100,912 | 0.984× | 0.975–1.002 | -0.68 |
| no OpenSSL (b4) | SLH-DSA | slh-dsa-shake-128s | signing | 789,143 | 781,661 | 0.991× | 0.979–1.007 | -0.68 |
| no OpenSSL (b4) | SLH-DSA | slh-dsa-shake-128s | verification | 776 | 767 | 0.988× | 0.967–1.043 | -0.04 |
| no OpenSSL (b4) | SLH-DSA | slh-dsa-shake-192f | key generation | 2,361 | 2,340 | 0.991× | 0.985–1.006 | -0.44 |
| no OpenSSL (b4) | SLH-DSA | slh-dsa-shake-192f | signing | 61,681 | 60,634 | 0.983× | 0.960–0.997 | -0.92 |
| no OpenSSL (b4) | SLH-DSA | slh-dsa-shake-192f | verification | 3,269 | 3,233 | 0.989× | 0.974–1.001 | -0.76 |
| no OpenSSL (b4) | SLH-DSA | slh-dsa-shake-192s | key generation | 150,323 | 150,083 | 0.998× | 0.968–1.007 | -0.20 |
| no OpenSSL (b4) | SLH-DSA | slh-dsa-shake-192s | signing | 1,404,275 | 1,379,366 | 0.982× | 0.952–0.994 | -0.92 |
| no OpenSSL (b4) | SLH-DSA | slh-dsa-shake-192s | verification | 1,134 | 1,126 | 0.994× | 0.972–1.008 | -0.36 |
| no OpenSSL (b4) | SLH-DSA | slh-dsa-shake-256f | key generation | 6,201 | 6,175 | 0.996× | 0.980–1.020 | -0.44 |
| no OpenSSL (b4) | SLH-DSA | slh-dsa-shake-256f | signing | 126,908 | 125,793 | 0.991× | 0.957–1.012 | -0.44 |
| no OpenSSL (b4) | SLH-DSA | slh-dsa-shake-256f | verification | 3,364 | 3,308 | 0.983× | 0.976–1.007 | -0.60 |
| no OpenSSL (b4) | SLH-DSA | slh-dsa-shake-256s | key generation | 100,021 | 98,977 | 0.990× | 0.974–1.019 | -0.36 |
| no OpenSSL (b4) | SLH-DSA | slh-dsa-shake-256s | signing | 1,247,915 | 1,224,579 | 0.981× | 0.932–1.038 | -0.68 |
| no OpenSSL (b4) | SLH-DSA | slh-dsa-shake-256s | verification | 1,667 | 1,669 | 1.001× | 0.982–1.022 | 0.12 |
| no OpenSSL (b4) | ML-KEM | ml-kem-1024 | decapsulation | 58.6 | 60.2 | 1.027× | 0.974–1.067 | 0.60 |
| no OpenSSL (b4) | ML-KEM | ml-kem-1024 | encapsulation | 53.7 | 51.3 | 0.954× | 0.939–1.036 | -0.28 |
| no OpenSSL (b4) | ML-KEM | ml-kem-1024 | key generation | 55.4 | 54.0 | 0.974× | 0.950–0.999 | -0.84 |
| no OpenSSL (b4) | ML-KEM | ml-kem-512 | decapsulation | 26.2 | 26.9 | 1.026× | 0.977–1.076 | 0.52 |
| no OpenSSL (b4) | ML-KEM | ml-kem-512 | encapsulation | 26.1 | 23.3 | 0.895× | 0.827–0.944 | -1.00 |
| no OpenSSL (b4) | ML-KEM | ml-kem-512 | key generation | 24.0 | 21.4 | 0.892× | 0.881–0.992 | -0.92 |
| no OpenSSL (b4) | ML-KEM | ml-kem-768 | decapsulation | 40.3 | 40.4 | 1.002× | 0.956–1.034 | 0.04 |
| no OpenSSL (b4) | ML-KEM | ml-kem-768 | encapsulation | 38.9 | 35.2 | 0.905× | 0.881–0.960 | -1.00 |
| no OpenSSL (b4) | ML-KEM | ml-kem-768 | key generation | 37.1 | 35.4 | 0.955× | 0.897–1.003 | -0.76 |
| no OpenSSL (b4) | HQC | hqc-1 | decapsulation | 3,730 | 3,677 | 0.986× | 0.981–1.007 | -0.60 |
| no OpenSSL (b4) | HQC | hqc-1 | encapsulation | 2,478 | 2,414 | 0.974× | 0.963–0.998 | -0.92 |
| no OpenSSL (b4) | HQC | hqc-1 | key generation | 1,231 | 1,215 | 0.987× | 0.968–0.994 | -1.00 |
| no OpenSSL (b4) | HQC | hqc-3 | decapsulation | 10,942 | 11,200 | 1.024× | 1.014–1.032 | 1.00 |
| no OpenSSL (b4) | HQC | hqc-3 | encapsulation | 7,270 | 7,306 | 1.005× | 0.964–1.009 | -0.28 |
| no OpenSSL (b4) | HQC | hqc-3 | key generation | 3,659 | 3,681 | 1.006× | 0.991–1.012 | 0.20 |
| no OpenSSL (b4) | HQC | hqc-5 | decapsulation | 26,783 | 27,464 | 1.025× | 0.997–1.031 | 0.76 |
| no OpenSSL (b4) | HQC | hqc-5 | encapsulation | 17,703 | 18,113 | 1.023× | 1.003–1.029 | 0.84 |
| no OpenSSL (b4) | HQC | hqc-5 | key generation | 8,854 | 9,075 | 1.025× | 1.009–1.036 | 1.00 |
| no OpenSSL (b4) | classical | dhkem-p256 | decapsulation | 126 | 122 | 0.974× | 0.951–1.041 | -0.04 |
| no OpenSSL (b4) | classical | dhkem-p256 | encapsulation | 138 | 145 | 1.047× | 0.961–1.055 | 0.52 |
| no OpenSSL (b4) | classical | dhkem-p256 | key generation | 38.5 | 38.2 | 0.994× | 0.969–1.051 | 0.12 |
| no OpenSSL (b4) | classical | dhkem-x25519 | decapsulation | 45.1 | 45.9 | 1.019× | 1.009–1.050 | 1.00 |
| no OpenSSL (b4) | classical | dhkem-x25519 | encapsulation | 90.3 | 90.7 | 1.004× | 0.984–1.025 | 0.20 |
| no OpenSSL (b4) | classical | dhkem-x25519 | key generation | 42.6 | 44.6 | 1.047× | 0.973–1.061 | 0.52 |
| no OpenSSL (b4) | classical | ecdsa-p256 | key generation | 40.8 | 39.1 | 0.958× | 0.947–1.008 | -0.60 |
| no OpenSSL (b4) | classical | ecdsa-p256 | signing | 30.9 | 31.3 | 1.012× | 0.976–1.066 | 0.52 |
| no OpenSSL (b4) | classical | ecdsa-p256 | verification | 78.4 | 72.6 | 0.926× | 0.913–1.046 | -0.36 |
| no OpenSSL (b4) | classical | ed25519 | key generation | 46.7 | 45.9 | 0.983× | 0.953–1.001 | -0.68 |
| no OpenSSL (b4) | classical | ed25519 | signing | 40.3 | 40.3 | 1.001× | 0.993–1.058 | -0.04 |
| no OpenSSL (b4) | classical | ed25519 | verification | 102 | 98.0 | 0.962× | 0.948–1.007 | -0.68 |
| no OpenSSL (b4) | classical | rsa3072-pss | key generation | 510,364 | 550,772 | 1.079× | 0.931–1.220 | 0.68 |
| no OpenSSL (b4) | classical | rsa3072-pss | signing | 2,176 | 2,159 | 0.992× | 0.989–1.004 | -0.36 |
| no OpenSSL (b4) | classical | rsa3072-pss | verification | 53.3 | 53.3 | 1.000× | 0.973–1.061 | 0.12 |

Generated by `analysis/paper_tables.py` from `results/combined/analysis_tables/T7_E4_build_sensitivity.csv`.
