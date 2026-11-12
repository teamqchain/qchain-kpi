# E09  Memory per call (E9) on the Mac M1 (Apple M1, ARM with NEON)

Stack high-water mark (KiB) and C heap of one call through the C library; Go allocation of one call through QChain's production code; peak resident memory of the process (MiB). Heap and Go figures are identical on the three machines.

| Family | Algorithm | Operation | Stack (KiB) | C heap peak (B) | C heap allocated (B) | Go bytes (production) | Go allocations (production) | Resident peak (MiB) |
|---|---|---|---:|---:|---:|---:|---:|---:|
| ML-DSA | ml-dsa-44 | key generation | 31.5 | 872 | 5,760 | 20,336 | 10 | 19.3 |
| ML-DSA | ml-dsa-44 | signing | 49.0 | 872 | 8,216 | 15,296 | 10 | 18.8 |
| ML-DSA | ml-dsa-44 | verification | 29.4 | 872 | 4,544 | 4,272 | 7 | 16.2 |
| ML-DSA | ml-dsa-65 | key generation | 48.5 | 872 | 9,776 | 30,832 | 10 | 19.7 |
| ML-DSA | ml-dsa-65 | signing | 72.9 | 872 | 13,816 | 21,312 | 10 | 18.9 |
| ML-DSA | ml-dsa-65 | verification | 44.4 | 872 | 7,688 | 5,680 | 7 | 16.3 |
| ML-DSA | ml-dsa-87 | key generation | 78.5 | 872 | 16,224 | 39,408 | 10 | 20.2 |
| ML-DSA | ml-dsa-87 | signing | 111.0 | 872 | 19,552 | 29,376 | 10 | 19.1 |
| ML-DSA | ml-dsa-87 | verification | 72.7 | 872 | 13,264 | 7,728 | 7 | 16.3 |
| Falcon | falcon-1024 | key generation | 34.9 | 264 | 264 | 22,800 | 10 | 19.5 |
| Falcon | falcon-1024 | signing | 82.8 | 264 | 528 | 9,824 | 10 | 18.7 |
| Falcon | falcon-1024 | verification | 9.2 | 264 | 264 | 3,536 | 7 | 16.3 |
| Falcon | falcon-512 | key generation | 18.6 | 264 | 264 | 12,048 | 10 | 18.9 |
| Falcon | falcon-512 | signing | 42.6 | 264 | 528 | 5,216 | 10 | 18.5 |
| Falcon | falcon-512 | verification | 5.4 | 264 | 264 | 1,936 | 7 | 16.2 |
| Falcon | falcon-padded-1024 | key generation | 34.9 | 264 | 264 | 22,808 | 10 | 19.4 |
| Falcon | falcon-padded-1024 | signing | 82.7 | 264 | 528 | 9,576 | 10 | 18.6 |
| Falcon | falcon-padded-1024 | verification | 9.2 | 264 | 264 | 3,544 | 7 | 16.2 |
| Falcon | falcon-padded-512 | key generation | 18.6 | 264 | 264 | 12,056 | 10 | 18.9 |
| Falcon | falcon-padded-512 | signing | 42.6 | 264 | 528 | 5,160 | 10 | 18.5 |
| Falcon | falcon-padded-512 | verification | 5.4 | 264 | 264 | 1,944 | 7 | 16.2 |
| SLH-DSA | slh-dsa-sha2-128f | key generation | 4.0 | 0 | 0 | 616 | 10 | 19.4 |
| SLH-DSA | slh-dsa-sha2-128f | signing | 4.4 | 0 | 0 | 100,616 | 10 | 21.1 |
| SLH-DSA | slh-dsa-sha2-128f | verification | 4.3 | 0 | 0 | 18,664 | 7 | 17.1 |
| SLH-DSA | slh-dsa-sha2-128s | key generation | 4.0 | 0 | 0 | 616 | 10 | 18.5 |
| SLH-DSA | slh-dsa-sha2-128s | signing | 4.4 | 0 | 0 | 41,224 | 10 | 18.9 |
| SLH-DSA | slh-dsa-sha2-128s | verification | 4.3 | 0 | 0 | 8,424 | 7 | 16.2 |
| SLH-DSA | slh-dsa-sha2-192f | key generation | 4.5 | 0 | 0 | 856 | 10 | 20.5 |
| SLH-DSA | slh-dsa-sha2-192f | signing | 4.9 | 0 | 0 | 188,712 | 10 | 22.4 |
| SLH-DSA | slh-dsa-sha2-192f | verification | 4.7 | 0 | 0 | 41,208 | 7 | 18.3 |
| SLH-DSA | slh-dsa-sha2-192s | key generation | 4.5 | 0 | 0 | 856 | 10 | 18.8 |
| SLH-DSA | slh-dsa-sha2-192s | signing | 4.9 | 0 | 0 | 82,216 | 10 | 19.2 |
| SLH-DSA | slh-dsa-sha2-192s | verification | 4.7 | 0 | 0 | 16,632 | 7 | 16.5 |
| SLH-DSA | slh-dsa-sha2-256f | key generation | 4.5 | 0 | 0 | 1,096 | 10 | 21.8 |
| SLH-DSA | slh-dsa-sha2-256f | signing | 4.9 | 0 | 0 | 270,664 | 10 | 25.2 |
| SLH-DSA | slh-dsa-sha2-256f | verification | 4.7 | 0 | 0 | 57,608 | 7 | 19.6 |
| SLH-DSA | slh-dsa-sha2-256s | key generation | 4.5 | 0 | 0 | 1,096 | 10 | 19.3 |
| SLH-DSA | slh-dsa-sha2-256s | signing | 4.9 | 0 | 0 | 164,168 | 10 | 20.1 |
| SLH-DSA | slh-dsa-sha2-256s | verification | 4.7 | 0 | 0 | 33,032 | 7 | 17.0 |
| SLH-DSA | slh-dsa-shake-128f | key generation | 3.8 | 0 | 0 | 616 | 10 | 19.4 |
| SLH-DSA | slh-dsa-shake-128f | signing | 4.2 | 0 | 0 | 100,616 | 10 | 21.1 |
| SLH-DSA | slh-dsa-shake-128f | verification | 4.0 | 0 | 0 | 18,664 | 7 | 17.1 |
| SLH-DSA | slh-dsa-shake-128s | key generation | 3.8 | 0 | 0 | 616 | 10 | 18.5 |
| SLH-DSA | slh-dsa-shake-128s | signing | 4.2 | 0 | 0 | 41,224 | 10 | 18.8 |
| SLH-DSA | slh-dsa-shake-128s | verification | 4.0 | 0 | 0 | 8,424 | 7 | 16.2 |
| SLH-DSA | slh-dsa-shake-192f | key generation | 3.8 | 0 | 0 | 856 | 10 | 20.5 |
| SLH-DSA | slh-dsa-shake-192f | signing | 4.2 | 0 | 0 | 188,712 | 10 | 22.6 |
| SLH-DSA | slh-dsa-shake-192f | verification | 4.0 | 0 | 0 | 41,208 | 7 | 18.3 |
| SLH-DSA | slh-dsa-shake-192s | key generation | 3.8 | 0 | 0 | 856 | 10 | 18.8 |
| SLH-DSA | slh-dsa-shake-192s | signing | 4.2 | 0 | 0 | 82,216 | 10 | 19.1 |
| SLH-DSA | slh-dsa-shake-192s | verification | 4.0 | 0 | 0 | 16,632 | 7 | 16.5 |
| SLH-DSA | slh-dsa-shake-256f | key generation | 3.8 | 0 | 0 | 1,096 | 10 | 21.8 |
| SLH-DSA | slh-dsa-shake-256f | signing | 4.2 | 0 | 0 | 270,664 | 10 | 25.4 |
| SLH-DSA | slh-dsa-shake-256f | verification | 4.0 | 0 | 0 | 57,608 | 7 | 19.6 |
| SLH-DSA | slh-dsa-shake-256s | key generation | 3.8 | 0 | 0 | 1,096 | 10 | 19.3 |
| SLH-DSA | slh-dsa-shake-256s | signing | 4.2 | 0 | 0 | 164,168 | 10 | 19.9 |
| SLH-DSA | slh-dsa-shake-256s | verification | 4.0 | 0 | 0 | 33,032 | 7 | 17.0 |
| ML-KEM | ml-kem-1024 | decapsulation | 23.2 | 872 | 6,288 | 5,280 | 9 | 16.1 |
| ML-KEM | ml-kem-1024 | encapsulation | 21.8 | 872 | 6,024 | 10,272 | 11 | 18.9 |
| ML-KEM | ml-kem-1024 | key generation | 18.2 | 872 | 5,760 | 24,576 | 10 | 19.4 |
| ML-KEM | ml-kem-512 | decapsulation | 11.9 | 872 | 2,800 | 2,848 | 9 | 16.1 |
| ML-KEM | ml-kem-512 | encapsulation | 11.2 | 872 | 2,536 | 5,024 | 11 | 18.6 |
| ML-KEM | ml-kem-512 | key generation | 8.7 | 872 | 2,272 | 13,312 | 10 | 18.8 |
| ML-KEM | ml-kem-768 | decapsulation | 17.3 | 872 | 4,544 | 4,128 | 9 | 16.1 |
| ML-KEM | ml-kem-768 | encapsulation | 16.3 | 872 | 4,280 | 7,328 | 11 | 18.8 |
| ML-KEM | ml-kem-768 | key generation | 13.2 | 872 | 4,280 | 19,200 | 10 | 19.2 |
| HQC | hqc-1 | decapsulation | 64.1 | 528 | 1,584 | 7,872 | 9 | 16.3 |
| HQC | hqc-1 | encapsulation | 57.5 | 528 | 1,056 | 26,432 | 11 | 19.4 |
| HQC | hqc-1 | key generation | 51.0 | 264 | 1,056 | 24,608 | 10 | 19.5 |
| HQC | hqc-3 | decapsulation | 128.6 | 528 | 1,584 | 14,656 | 9 | 16.6 |
| HQC | hqc-3 | encapsulation | 115.3 | 528 | 1,056 | 51,520 | 11 | 20.4 |
| HQC | hqc-3 | key generation | 102.2 | 264 | 1,056 | 47,776 | 10 | 20.7 |
| HQC | hqc-5 | decapsulation | 205.7 | 528 | 1,584 | 24,896 | 9 | 17.1 |
| HQC | hqc-5 | encapsulation | 184.3 | 528 | 1,056 | 90,432 | 11 | 22.4 |
| HQC | hqc-5 | key generation | 163.3 | 264 | 1,056 | 82,080 | 10 | 22.2 |
| classical | dhkem-p256 | decapsulation | 2.8 | 5,240 | 29,000 | 448 | 8 | 18.4 |
| classical | dhkem-p256 | encapsulation | 2.8 | 6,776 | 31,672 | 720 | 10 | 18.6 |
| classical | dhkem-p256 | key generation | 2.8 | 6,680 | 15,256 | 752 | 12 | 18.5 |
| classical | dhkem-x25519 | decapsulation | 2.8 | 3,152 | 13,632 | 352 | 8 | 18.2 |
| classical | dhkem-x25519 | encapsulation | 2.8 | 3,904 | 16,728 | 464 | 10 | 18.4 |
| classical | dhkem-x25519 | key generation | 1.9 | 3,808 | 7,640 | 544 | 12 | 18.4 |
| classical | ecdsa-p256 | key generation | 2.9 | 6,680 | 15,256 | 5,760 | 12 | 18.5 |
| classical | ecdsa-p256 | signing | 2.5 | 3,200 | 7,160 | 656 | 9 | 18.5 |
| classical | ecdsa-p256 | verification | 1.6 | 5,416 | 8,560 | 320 | 5 | 18.4 |
| classical | ed25519 | key generation | 1.8 | 3,808 | 7,952 | 5,552 | 12 | 18.4 |
| classical | ed25519 | signing | 1.8 | 2,512 | 3,864 | 528 | 8 | 18.2 |
| classical | ed25519 | verification | 3.5 | 2,512 | 3,432 | 256 | 5 | 18.2 |
| classical | rsa3072-pss | key generation | 2.7 | 41,168 | 3,707,392 | 17,712 | 12 | 18.6 |
| classical | rsa3072-pss | signing | 2.2 | 16,680 | 39,472 | 4,144 | 8 | 18.6 |
| classical | rsa3072-pss | verification | 1.9 | 4,968 | 10,048 | 992 | 5 | 18.2 |

Generated by `analysis/paper_tables.py` from `results/combined/analysis_tables/T4_E9_memory.csv`.
