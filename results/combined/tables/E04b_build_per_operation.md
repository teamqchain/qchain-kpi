# E04b  liboqs build variants against the production build, every operation (E4)

Variant ÷ production build (b1), ratio of medians.

| Family | Algorithm | Operation | Mac M1, portable (b2) | Mac M1, native (b3) | Laptop, portable (b2) | Laptop, native (b3) | VM, no OpenSSL (b4) |
|---|---|---|---:|---:|---:|---:|---:|
| ML-DSA | ml-dsa-44 | key generation | 1.28× | 1.02× | 2.37× | 0.99× | 0.94× |
| ML-DSA | ml-dsa-44 | signing | 1.71× | 1.00× | 3.06× | 0.98× | 1.01× |
| ML-DSA | ml-dsa-44 | verification | 1.38× | 0.98× | 2.35× | 0.98× | 1.01× |
| ML-DSA | ml-dsa-65 | key generation | 1.48× | 1.00× | 2.30× | 0.99× | 1.00× |
| ML-DSA | ml-dsa-65 | signing | 1.70× | 1.01× | 3.02× | 0.99× | 1.00× |
| ML-DSA | ml-dsa-65 | verification | 1.34× | 1.00× | 2.22× | 0.99× | 1.02× |
| ML-DSA | ml-dsa-87 | key generation | 1.20× | 0.98× | 2.18× | 0.99× | 1.00× |
| ML-DSA | ml-dsa-87 | signing | 1.54× | 0.99× | 3.14× | 0.98× | 0.99× |
| ML-DSA | ml-dsa-87 | verification | 1.27× | 1.01× | 2.35× | 0.98× | 0.99× |
| Falcon | falcon-1024 | key generation | 2.16× | 0.99× | 2.09× | 0.99× | 1.00× |
| Falcon | falcon-1024 | signing | 27.07× | 0.99× | 19.76× | 0.99× | 1.00× |
| Falcon | falcon-1024 | verification | 1.26× | 0.98× | 0.96× | 1.00× | 1.03× |
| Falcon | falcon-512 | key generation | 2.25× | 1.00× | 2.12× | 1.00× | 1.00× |
| Falcon | falcon-512 | signing | 24.87× | 1.00× | 18.04× | 0.99× | 0.99× |
| Falcon | falcon-512 | verification | 1.27× | 1.00× | 0.95× | 1.01× | 1.00× |
| Falcon | falcon-padded-1024 | key generation | 2.15× | 1.00× | 2.13× | 0.99× | 1.00× |
| Falcon | falcon-padded-1024 | signing | 27.35× | 1.01× | 19.67× | 0.99× | 1.00× |
| Falcon | falcon-padded-1024 | verification | 1.27× | 1.00× | 0.96× | 1.00× | 1.01× |
| Falcon | falcon-padded-512 | key generation | 2.24× | 1.00× | 2.12× | 1.00× | 1.00× |
| Falcon | falcon-padded-512 | signing | 24.95× | 1.00× | 18.27× | 1.00× | 1.00× |
| Falcon | falcon-padded-512 | verification | 1.25× | 0.99× | 0.95× | 1.00× | 0.97× |
| SLH-DSA | slh-dsa-sha2-128f | key generation | 1.00× | 1.00× | 0.99× | 1.05× | 0.99× |
| SLH-DSA | slh-dsa-sha2-128f | signing | 1.00× | 1.00× | 1.00× | 1.05× | 0.99× |
| SLH-DSA | slh-dsa-sha2-128f | verification | 1.00× | 1.00× | 0.98× | 1.05× | 0.99× |
| SLH-DSA | slh-dsa-sha2-128s | key generation | 1.00× | 1.00× | 1.00× | 1.06× | 1.01× |
| SLH-DSA | slh-dsa-sha2-128s | signing | 1.00× | 1.00× | 1.00× | 1.05× | 1.00× |
| SLH-DSA | slh-dsa-sha2-128s | verification | 1.00× | 1.01× | 0.98× | 1.04× | 1.02× |
| SLH-DSA | slh-dsa-sha2-192f | key generation | 1.00× | 1.00× | 1.00× | 1.06× | 1.01× |
| SLH-DSA | slh-dsa-sha2-192f | signing | 1.00× | 1.00× | 1.00× | 1.04× | 1.00× |
| SLH-DSA | slh-dsa-sha2-192f | verification | 1.00× | 1.00× | 1.00× | 1.05× | 1.02× |
| SLH-DSA | slh-dsa-sha2-192s | key generation | 1.00× | 1.00× | 1.00× | 1.05× | 0.98× |
| SLH-DSA | slh-dsa-sha2-192s | signing | 1.00× | 1.00× | 1.00× | 1.03× | 1.00× |
| SLH-DSA | slh-dsa-sha2-192s | verification | 1.00× | 1.00× | 0.99× | 1.03× | 0.99× |
| SLH-DSA | slh-dsa-sha2-256f | key generation | 1.00× | 1.00× | 1.00× | 1.04× | 1.01× |
| SLH-DSA | slh-dsa-sha2-256f | signing | 1.00× | 1.00× | 1.00× | 1.02× | 0.99× |
| SLH-DSA | slh-dsa-sha2-256f | verification | 1.00× | 1.00× | 0.99× | 1.03× | 1.00× |
| SLH-DSA | slh-dsa-sha2-256s | key generation | 1.00× | 1.00× | 1.00× | 1.04× | 1.01× |
| SLH-DSA | slh-dsa-sha2-256s | signing | 1.00× | 1.00× | 1.00× | 1.01× | 0.99× |
| SLH-DSA | slh-dsa-sha2-256s | verification | 1.00× | 1.00× | 0.99× | 1.02× | 0.98× |
| SLH-DSA | slh-dsa-shake-128f | key generation | 1.00× | 1.00× | 0.97× | 0.77× | 0.99× |
| SLH-DSA | slh-dsa-shake-128f | signing | 1.00× | 1.01× | 0.94× | 0.73× | 1.00× |
| SLH-DSA | slh-dsa-shake-128f | verification | 1.00× | 1.00× | 0.99× | 0.77× | 0.99× |
| SLH-DSA | slh-dsa-shake-128s | key generation | 1.01× | 1.01× | 0.95× | 0.72× | 0.98× |
| SLH-DSA | slh-dsa-shake-128s | signing | 1.00× | 0.99× | 0.96× | 0.73× | 0.99× |
| SLH-DSA | slh-dsa-shake-128s | verification | 1.00× | 1.01× | 1.00× | 0.80× | 0.99× |
| SLH-DSA | slh-dsa-shake-192f | key generation | 1.00× | 1.00× | 0.99× | 0.77× | 0.99× |
| SLH-DSA | slh-dsa-shake-192f | signing | 1.01× | 1.00× | 0.99× | 0.74× | 0.98× |
| SLH-DSA | slh-dsa-shake-192f | verification | 1.00× | 1.00× | 1.00× | 0.75× | 0.99× |
| SLH-DSA | slh-dsa-shake-192s | key generation | 1.00× | 1.00× | 0.98× | 0.72× | 1.00× |
| SLH-DSA | slh-dsa-shake-192s | signing | 0.99× | 1.00× | 0.98× | 0.75× | 0.98× |
| SLH-DSA | slh-dsa-shake-192s | verification | 1.00× | 1.01× | 1.00× | 0.79× | 0.99× |
| SLH-DSA | slh-dsa-shake-256f | key generation | 1.00× | 1.00× | 1.00× | 0.83× | 1.00× |
| SLH-DSA | slh-dsa-shake-256f | signing | 1.00× | 1.01× | 1.00× | 0.83× | 0.99× |
| SLH-DSA | slh-dsa-shake-256f | verification | 1.00× | 1.00× | 1.00× | 0.83× | 0.98× |
| SLH-DSA | slh-dsa-shake-256s | key generation | 1.00× | 1.00× | 1.00× | 0.83× | 0.99× |
| SLH-DSA | slh-dsa-shake-256s | signing | 0.99× | 1.00× | 0.99× | 0.83× | 0.98× |
| SLH-DSA | slh-dsa-shake-256s | verification | 1.00× | 1.00× | 1.00× | 0.84× | 1.00× |
| ML-KEM | ml-kem-1024 | decapsulation | 1.40× | 1.02× | 1.71× | 0.99× | 1.03× |
| ML-KEM | ml-kem-1024 | encapsulation | 1.37× | 1.02× | 1.76× | 0.99× | 0.95× |
| ML-KEM | ml-kem-1024 | key generation | 1.62× | 0.99× | 1.80× | 0.98× | 0.97× |
| ML-KEM | ml-kem-512 | decapsulation | 1.47× | 1.00× | 1.89× | 0.98× | 1.03× |
| ML-KEM | ml-kem-512 | encapsulation | 1.45× | 1.01× | 1.84× | 0.98× | 0.90× |
| ML-KEM | ml-kem-512 | key generation | 1.55× | 1.00× | 1.80× | 0.98× | 0.89× |
| ML-KEM | ml-kem-768 | decapsulation | 1.40× | 0.98× | 1.72× | 0.98× | 1.00× |
| ML-KEM | ml-kem-768 | encapsulation | 1.40× | 1.01× | 1.73× | 0.98× | 0.91× |
| ML-KEM | ml-kem-768 | key generation | 1.62× | 1.02× | 1.71× | 0.99× | 0.95× |
| HQC | hqc-1 | decapsulation | 0.86× | 0.86× | 1.00× | 0.96× | 0.99× |
| HQC | hqc-1 | encapsulation | 0.86× | 0.86× | 1.00× | 0.97× | 0.97× |
| HQC | hqc-1 | key generation | 0.87× | 0.87× | 1.00× | 0.97× | 0.99× |
| HQC | hqc-3 | decapsulation | 1.00× | 1.00× | 1.00× | 0.97× | 1.02× |
| HQC | hqc-3 | encapsulation | 1.00× | 1.00× | 1.00× | 0.97× | 1.00× |
| HQC | hqc-3 | key generation | 1.00× | 1.00× | 1.00× | 0.97× | 1.01× |
| HQC | hqc-5 | decapsulation | 1.00× | 1.00× | 1.00× | 0.98× | 1.02× |
| HQC | hqc-5 | encapsulation | 1.00× | 1.00× | 1.00× | 0.98× | 1.02× |
| HQC | hqc-5 | key generation | 1.00× | 1.00× | 1.00× | 0.98× | 1.02× |
| classical | dhkem-p256 | decapsulation | 1.00× | 1.00× | 1.00× | 1.00× | 0.97× |
| classical | dhkem-p256 | encapsulation | 1.00× | 1.00× | 1.00× | 1.01× | 1.05× |
| classical | dhkem-p256 | key generation | 0.98× | 1.00× | 0.99× | 1.00× | 0.99× |
| classical | dhkem-x25519 | decapsulation | 1.00× | 1.02× | 0.99× | 1.00× | 1.02× |
| classical | dhkem-x25519 | encapsulation | 1.00× | 1.00× | 0.99× | 0.99× | 1.00× |
| classical | dhkem-x25519 | key generation | 0.98× | 0.98× | 1.00× | 1.00× | 1.05× |
| classical | ecdsa-p256 | key generation | 1.00× | 1.00× | 0.99× | 0.99× | 0.96× |
| classical | ecdsa-p256 | signing | 1.00× | 1.00× | 1.00× | 0.99× | 1.01× |
| classical | ecdsa-p256 | verification | 1.00× | 1.00× | 1.00× | 1.00× | 0.93× |
| classical | ed25519 | key generation | 1.00× | 1.00× | 1.00× | 1.00× | 0.98× |
| classical | ed25519 | signing | 1.00× | 1.00× | 1.00× | 1.00× | 1.00× |
| classical | ed25519 | verification | 1.00× | 1.00× | 1.00× | 1.00× | 0.96× |
| classical | rsa3072-pss | key generation | 1.08× | 1.04× | 0.94× | 1.02× | 1.08× |
| classical | rsa3072-pss | signing | 1.00× | 1.00× | 1.00× | 1.00× | 0.99× |
| classical | rsa3072-pss | verification | 1.00× | 1.00× | 1.00× | 1.00× | 1.00× |

Generated by `analysis/paper_tables.py` from `results/combined/analysis_tables/T7_E4_build_sensitivity.csv`.
