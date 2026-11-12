# E09  Memory per call (E9) on the Laptop (AMD Ryzen 5 7520U, x86 with AVX2)

Stack high-water mark (KiB) and C heap of one call through the C library; Go allocation of one call through QChain's production code; peak resident memory of the process (MiB). Heap and Go figures are identical on the three machines.

| Family | Algorithm | Operation | Stack (KiB) | C heap peak (B) | C heap allocated (B) | Go bytes (production) | Go allocations (production) | Resident peak (MiB) |
|---|---|---|---:|---:|---:|---:|---:|---:|
| ML-DSA | ml-dsa-44 | key generation | 31.7 | 872 | 5,760 | 20,336 | 10 | 20.7 |
| ML-DSA | ml-dsa-44 | signing | 49.3 | 872 | 8,216 | 15,296 | 10 | 20.1 |
| ML-DSA | ml-dsa-44 | verification | 29.7 | 872 | 4,544 | 4,272 | 7 | 17.6 |
| ML-DSA | ml-dsa-65 | key generation | 48.7 | 872 | 9,776 | 30,832 | 10 | 21.0 |
| ML-DSA | ml-dsa-65 | signing | 73.3 | 872 | 13,816 | 21,312 | 10 | 20.3 |
| ML-DSA | ml-dsa-65 | verification | 44.8 | 872 | 7,688 | 5,680 | 7 | 17.7 |
| ML-DSA | ml-dsa-87 | key generation | 78.7 | 872 | 16,224 | 39,408 | 10 | 21.6 |
| ML-DSA | ml-dsa-87 | signing | 111.3 | 872 | 19,552 | 29,376 | 10 | 20.6 |
| ML-DSA | ml-dsa-87 | verification | 73.0 | 872 | 13,264 | 7,728 | 7 | 17.8 |
| Falcon | falcon-1024 | key generation | 35.0 | 264 | 264 | 22,800 | 10 | 20.8 |
| Falcon | falcon-1024 | signing | 84.2 | 264 | 528 | 9,824 | 10 | 20.0 |
| Falcon | falcon-1024 | verification | 8.4 | 264 | 264 | 3,536 | 7 | 17.5 |
| Falcon | falcon-512 | key generation | 18.5 | 264 | 264 | 12,048 | 10 | 20.3 |
| Falcon | falcon-512 | signing | 44.0 | 264 | 528 | 5,216 | 10 | 19.9 |
| Falcon | falcon-512 | verification | 4.4 | 264 | 264 | 1,936 | 7 | 17.6 |
| Falcon | falcon-padded-1024 | key generation | 35.0 | 264 | 264 | 22,808 | 10 | 20.8 |
| Falcon | falcon-padded-1024 | signing | 84.2 | 264 | 528 | 9,576 | 10 | 20.0 |
| Falcon | falcon-padded-1024 | verification | 8.4 | 264 | 264 | 3,544 | 7 | 17.5 |
| Falcon | falcon-padded-512 | key generation | 18.5 | 264 | 264 | 12,056 | 10 | 20.3 |
| Falcon | falcon-padded-512 | signing | 44.0 | 264 | 528 | 5,160 | 10 | 19.9 |
| Falcon | falcon-padded-512 | verification | 4.4 | 264 | 264 | 1,944 | 7 | 17.5 |
| SLH-DSA | slh-dsa-sha2-128f | key generation | 4.0 | 0 | 0 | 616 | 10 | 20.6 |
| SLH-DSA | slh-dsa-sha2-128f | signing | 4.4 | 0 | 0 | 100,616 | 10 | 22.4 |
| SLH-DSA | slh-dsa-sha2-128f | verification | 4.4 | 0 | 0 | 18,664 | 7 | 18.4 |
| SLH-DSA | slh-dsa-sha2-128s | key generation | 4.0 | 0 | 0 | 616 | 10 | 19.8 |
| SLH-DSA | slh-dsa-sha2-128s | signing | 4.4 | 0 | 0 | 41,224 | 10 | 20.1 |
| SLH-DSA | slh-dsa-sha2-128s | verification | 4.4 | 0 | 0 | 8,424 | 7 | 17.5 |
| SLH-DSA | slh-dsa-sha2-192f | key generation | 4.5 | 0 | 0 | 856 | 10 | 21.7 |
| SLH-DSA | slh-dsa-sha2-192f | signing | 4.9 | 0 | 0 | 188,712 | 10 | 23.7 |
| SLH-DSA | slh-dsa-sha2-192f | verification | 4.8 | 0 | 0 | 41,208 | 7 | 19.6 |
| SLH-DSA | slh-dsa-sha2-192s | key generation | 4.5 | 0 | 0 | 856 | 10 | 20.0 |
| SLH-DSA | slh-dsa-sha2-192s | signing | 4.9 | 0 | 0 | 82,216 | 10 | 20.4 |
| SLH-DSA | slh-dsa-sha2-192s | verification | 4.8 | 0 | 0 | 16,632 | 7 | 17.8 |
| SLH-DSA | slh-dsa-sha2-256f | key generation | 4.5 | 0 | 0 | 1,096 | 10 | 23.1 |
| SLH-DSA | slh-dsa-sha2-256f | signing | 4.9 | 0 | 0 | 270,664 | 10 | 26.5 |
| SLH-DSA | slh-dsa-sha2-256f | verification | 4.8 | 0 | 0 | 57,608 | 7 | 20.9 |
| SLH-DSA | slh-dsa-sha2-256s | key generation | 4.5 | 0 | 0 | 1,096 | 10 | 20.5 |
| SLH-DSA | slh-dsa-sha2-256s | signing | 4.9 | 0 | 0 | 164,168 | 10 | 21.4 |
| SLH-DSA | slh-dsa-sha2-256s | verification | 4.8 | 0 | 0 | 33,032 | 7 | 18.3 |
| SLH-DSA | slh-dsa-shake-128f | key generation | 3.8 | 0 | 0 | 616 | 10 | 20.6 |
| SLH-DSA | slh-dsa-shake-128f | signing | 4.2 | 0 | 0 | 100,616 | 10 | 22.4 |
| SLH-DSA | slh-dsa-shake-128f | verification | 4.1 | 0 | 0 | 18,664 | 7 | 18.4 |
| SLH-DSA | slh-dsa-shake-128s | key generation | 3.8 | 0 | 0 | 616 | 10 | 19.8 |
| SLH-DSA | slh-dsa-shake-128s | signing | 4.2 | 0 | 0 | 41,224 | 10 | 19.9 |
| SLH-DSA | slh-dsa-shake-128s | verification | 4.1 | 0 | 0 | 8,424 | 7 | 17.6 |
| SLH-DSA | slh-dsa-shake-192f | key generation | 3.8 | 0 | 0 | 856 | 10 | 21.7 |
| SLH-DSA | slh-dsa-shake-192f | signing | 4.2 | 0 | 0 | 188,712 | 10 | 23.7 |
| SLH-DSA | slh-dsa-shake-192f | verification | 4.1 | 0 | 0 | 41,208 | 7 | 19.6 |
| SLH-DSA | slh-dsa-shake-192s | key generation | 3.8 | 0 | 0 | 856 | 10 | 20.0 |
| SLH-DSA | slh-dsa-shake-192s | signing | 4.2 | 0 | 0 | 82,216 | 10 | 20.2 |
| SLH-DSA | slh-dsa-shake-192s | verification | 4.1 | 0 | 0 | 16,632 | 7 | 17.8 |
| SLH-DSA | slh-dsa-shake-256f | key generation | 3.8 | 0 | 0 | 1,096 | 10 | 23.0 |
| SLH-DSA | slh-dsa-shake-256f | signing | 4.2 | 0 | 0 | 270,664 | 10 | 26.5 |
| SLH-DSA | slh-dsa-shake-256f | verification | 4.1 | 0 | 0 | 57,608 | 7 | 20.9 |
| SLH-DSA | slh-dsa-shake-256s | key generation | 3.8 | 0 | 0 | 1,096 | 10 | 20.5 |
| SLH-DSA | slh-dsa-shake-256s | signing | 4.2 | 0 | 0 | 164,168 | 10 | 20.9 |
| SLH-DSA | slh-dsa-shake-256s | verification | 4.1 | 0 | 0 | 33,032 | 7 | 18.3 |
| ML-KEM | ml-kem-1024 | decapsulation | 23.9 | 872 | 6,288 | 5,280 | 9 | 17.5 |
| ML-KEM | ml-kem-1024 | encapsulation | 22.4 | 872 | 6,024 | 10,272 | 11 | 20.3 |
| ML-KEM | ml-kem-1024 | key generation | 18.9 | 872 | 5,760 | 24,576 | 10 | 20.7 |
| ML-KEM | ml-kem-512 | decapsulation | 12.9 | 872 | 2,800 | 2,848 | 9 | 17.4 |
| ML-KEM | ml-kem-512 | encapsulation | 12.2 | 872 | 2,536 | 5,024 | 11 | 19.9 |
| ML-KEM | ml-kem-512 | key generation | 9.6 | 872 | 2,272 | 13,312 | 10 | 20.2 |
| ML-KEM | ml-kem-768 | decapsulation | 18.0 | 872 | 4,544 | 4,128 | 9 | 17.4 |
| ML-KEM | ml-kem-768 | encapsulation | 16.9 | 872 | 4,280 | 7,328 | 11 | 20.1 |
| ML-KEM | ml-kem-768 | key generation | 13.9 | 872 | 4,280 | 19,200 | 10 | 20.5 |
| HQC | hqc-1 | decapsulation | 64.3 | 528 | 1,584 | 7,872 | 9 | 17.6 |
| HQC | hqc-1 | encapsulation | 57.7 | 528 | 1,056 | 26,432 | 11 | 20.7 |
| HQC | hqc-1 | key generation | 51.2 | 264 | 1,056 | 24,608 | 10 | 20.8 |
| HQC | hqc-3 | decapsulation | 128.9 | 528 | 1,584 | 14,656 | 9 | 17.8 |
| HQC | hqc-3 | encapsulation | 115.5 | 528 | 1,056 | 51,520 | 11 | 21.6 |
| HQC | hqc-3 | key generation | 102.5 | 264 | 1,056 | 47,776 | 10 | 21.9 |
| HQC | hqc-5 | decapsulation | 205.9 | 528 | 1,584 | 24,896 | 9 | 18.3 |
| HQC | hqc-5 | encapsulation | 184.6 | 528 | 1,056 | 90,432 | 11 | 23.6 |
| HQC | hqc-5 | key generation | 163.5 | 264 | 1,056 | 82,080 | 10 | 23.5 |
| classical | dhkem-p256 | decapsulation | 3.0 | 5,240 | 29,000 | 448 | 8 | 19.7 |
| classical | dhkem-p256 | encapsulation | 3.1 | 6,808 | 31,800 | 720 | 10 | 19.9 |
| classical | dhkem-p256 | key generation | 2.8 | 6,712 | 15,320 | 752 | 12 | 19.8 |
| classical | dhkem-x25519 | decapsulation | 3.0 | 3,152 | 13,632 | 352 | 8 | 19.5 |
| classical | dhkem-x25519 | encapsulation | 3.1 | 3,936 | 16,856 | 464 | 10 | 19.7 |
| classical | dhkem-x25519 | key generation | 2.0 | 3,840 | 7,704 | 544 | 12 | 19.6 |
| classical | ecdsa-p256 | key generation | 2.8 | 6,712 | 15,320 | 5,760 | 12 | 19.9 |
| classical | ecdsa-p256 | signing | 3.1 | 3,232 | 7,192 | 656 | 9 | 19.9 |
| classical | ecdsa-p256 | verification | 1.8 | 5,416 | 8,560 | 320 | 5 | 19.6 |
| classical | ed25519 | key generation | 3.1 | 3,840 | 8,016 | 5,552 | 12 | 19.6 |
| classical | ed25519 | signing | 3.1 | 2,512 | 3,864 | 528 | 8 | 19.5 |
| classical | ed25519 | verification | 5.1 | 2,512 | 3,432 | 256 | 5 | 19.4 |
| classical | rsa3072-pss | key generation | 3.1 | 29,264 | 2,069,328 | 17,712 | 12 | 20.0 |
| classical | rsa3072-pss | signing | 3.1 | 10,600 | 27,344 | 4,144 | 8 | 19.9 |
| classical | rsa3072-pss | verification | 2.0 | 4,968 | 10,048 | 992 | 5 | 19.7 |

Generated by `analysis/paper_tables.py` from `results/combined/analysis_tables/T4_E9_memory.csv`.
