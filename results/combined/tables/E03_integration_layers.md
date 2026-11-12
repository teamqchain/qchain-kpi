# E03  Integration layers on the Mac (E3): C library, Go-to-C call, Go binding, Go's own library

Medians (µs) over five repetitions. Boundary: the empty Go-to-C call; binding: the Go binding's own work (ns). Go's own library exists for the classical algorithms only.

| Family | Algorithm | Operation | C library | Go → C | Go binding | Go standard library | Boundary (ns) | Binding (ns) | Binding ÷ C | Go library ÷ binding |
|---|---|---|---:|---:|---:|---:|---:|---:|---:|---:|
| ML-DSA | ml-dsa-44 | key generation | 30.1 | 30.2 | 30.3 | – | 83 | 125 | 1.007× | – |
| ML-DSA | ml-dsa-44 | signing | 59.9 | 59.9 | 60.1 | – | 0 | 166 | 1.003× | – |
| ML-DSA | ml-dsa-44 | verification | 28.0 | 28.0 | 28.0 | – | -42 | 41 | 1.000× | – |
| ML-DSA | ml-dsa-65 | key generation | 52.4 | 52.6 | 52.8 | – | 208 | 209 | 1.008× | – |
| ML-DSA | ml-dsa-65 | signing | 96.8 | 96.2 | 96.3 | – | -583 | 22 | 0.994× | – |
| ML-DSA | ml-dsa-65 | verification | 48.5 | 48.5 | 48.4 | – | 41 | -83 | 0.999× | – |
| ML-DSA | ml-dsa-87 | key generation | 90.2 | 90.4 | 90.6 | – | 126 | 249 | 1.004× | – |
| ML-DSA | ml-dsa-87 | signing | 138 | 138 | 138 | – | 396 | 20 | 1.003× | – |
| ML-DSA | ml-dsa-87 | verification | 83.1 | 83.9 | 83.4 | – | 792 | -500 | 1.004× | – |
| Falcon | falcon-1024 | key generation | 14,823 | 14,799 | 14,848 | – | -23784 | 48538 | 1.002× | – |
| Falcon | falcon-1024 | signing | 290 | 290 | 290 | – | 167 | 416 | 1.002× | – |
| Falcon | falcon-1024 | verification | 41.0 | 41.0 | 41.1 | – | 0 | 125 | 1.003× | – |
| Falcon | falcon-512 | key generation | 4,797 | 4,799 | 4,798 | – | 2417 | -958 | 1.000× | – |
| Falcon | falcon-512 | signing | 147 | 147 | 147 | – | 41 | 167 | 1.001× | – |
| Falcon | falcon-512 | verification | 20.4 | 20.5 | 20.4 | – | 83 | -42 | 1.002× | – |
| Falcon | falcon-padded-1024 | key generation | 14,839 | 14,821 | 14,836 | – | -18436 | 14936 | 1.000× | – |
| Falcon | falcon-padded-1024 | signing | 292 | 292 | 292 | – | 42 | 83 | 1.000× | – |
| Falcon | falcon-padded-1024 | verification | 40.5 | 40.5 | 40.5 | – | 42 | -41 | 1.000× | – |
| Falcon | falcon-padded-512 | key generation | 4,795 | 4,796 | 4,802 | – | 1125 | 5666 | 1.001× | – |
| Falcon | falcon-padded-512 | signing | 147 | 147 | 147 | – | -209 | 250 | 1.000× | – |
| Falcon | falcon-padded-512 | verification | 20.4 | 20.5 | 20.5 | – | 124 | 0 | 1.006× | – |
| SLH-DSA | slh-dsa-sha2-128f | key generation | 888 | 888 | 888 | – | -41 | 500 | 1.000× | – |
| SLH-DSA | slh-dsa-sha2-128f | signing | 20,716 | 20,722 | 20,731 | – | 5707 | 9250 | 1.001× | – |
| SLH-DSA | slh-dsa-sha2-128f | verification | 1,254 | 1,255 | 1,255 | – | 1042 | -583 | 1.000× | – |
| SLH-DSA | slh-dsa-sha2-128s | key generation | 56,800 | 56,817 | 56,808 | – | 17290 | -8978 | 1.000× | – |
| SLH-DSA | slh-dsa-sha2-128s | signing | 431,406 | 431,545 | 431,414 | – | 138283 | -130159 | 1.000× | – |
| SLH-DSA | slh-dsa-sha2-128s | verification | 428 | 428 | 428 | – | 458 | 208 | 1.002× | – |
| SLH-DSA | slh-dsa-sha2-192f | key generation | 1,296 | 1,298 | 1,298 | – | 1292 | 334 | 1.001× | – |
| SLH-DSA | slh-dsa-sha2-192f | signing | 33,760 | 33,779 | 33,781 | – | 19165 | 2250 | 1.001× | – |
| SLH-DSA | slh-dsa-sha2-192f | verification | 1,829 | 1,832 | 1,829 | – | 2458 | -2958 | 1.000× | – |
| SLH-DSA | slh-dsa-sha2-192s | key generation | 82,743 | 82,742 | 82,751 | – | -999 | 8415 | 1.000× | – |
| SLH-DSA | slh-dsa-sha2-192s | signing | 757,635 | 757,568 | 757,409 | – | -66870 | -159406 | 1.000× | – |
| SLH-DSA | slh-dsa-sha2-192s | verification | 632 | 633 | 633 | – | 376 | -126 | 1.000× | – |
| SLH-DSA | slh-dsa-sha2-256f | key generation | 3,417 | 3,419 | 3,422 | – | 2750 | 2458 | 1.002× | – |
| SLH-DSA | slh-dsa-sha2-256f | signing | 69,437 | 69,434 | 69,464 | – | -3728 | 30768 | 1.000× | – |
| SLH-DSA | slh-dsa-sha2-256f | verification | 1,871 | 1,873 | 1,872 | – | 1334 | -542 | 1.000× | – |
| SLH-DSA | slh-dsa-sha2-256s | key generation | 54,643 | 54,647 | 54,651 | – | 3667 | 4791 | 1.000× | – |
| SLH-DSA | slh-dsa-sha2-256s | signing | 667,762 | 668,065 | 667,797 | – | 302916 | -267794 | 1.000× | – |
| SLH-DSA | slh-dsa-sha2-256s | verification | 927 | 928 | 928 | – | 667 | 458 | 1.001× | – |
| SLH-DSA | slh-dsa-shake-128f | key generation | 1,117 | 1,118 | 1,118 | – | 1166 | 0 | 1.001× | – |
| SLH-DSA | slh-dsa-shake-128f | signing | 26,387 | 26,363 | 26,365 | – | -23874 | 2063 | 0.999× | – |
| SLH-DSA | slh-dsa-shake-128f | verification | 1,572 | 1,573 | 1,574 | – | 1417 | 791 | 1.001× | – |
| SLH-DSA | slh-dsa-shake-128s | key generation | 70,870 | 70,824 | 70,812 | – | -46248 | -12123 | 0.999× | – |
| SLH-DSA | slh-dsa-shake-128s | signing | 547,482 | 547,516 | 546,057 | – | 34290 | -1458732 | 0.997× | – |
| SLH-DSA | slh-dsa-shake-128s | verification | 544 | 544 | 543 | – | 500 | -770 | 1.000× | – |
| SLH-DSA | slh-dsa-shake-192f | key generation | 1,657 | 1,644 | 1,643 | – | -13041 | -791 | 0.992× | – |
| SLH-DSA | slh-dsa-shake-192f | signing | 43,569 | 43,453 | 43,476 | – | -116118 | 22707 | 0.998× | – |
| SLH-DSA | slh-dsa-shake-192f | verification | 2,292 | 2,293 | 2,297 | – | 791 | 4083 | 1.002× | – |
| SLH-DSA | slh-dsa-shake-192s | key generation | 105,315 | 105,335 | 105,321 | – | 19956 | -13540 | 1.000× | – |
| SLH-DSA | slh-dsa-shake-192s | signing | 979,702 | 972,754 | 975,477 | – | -6948246 | 2722706 | 0.996× | – |
| SLH-DSA | slh-dsa-shake-192s | verification | 793 | 794 | 794 | – | 667 | 83 | 1.001× | – |
| SLH-DSA | slh-dsa-shake-256f | key generation | 4,369 | 4,371 | 4,372 | – | 2438 | 791 | 1.001× | – |
| SLH-DSA | slh-dsa-shake-256f | signing | 90,520 | 90,350 | 90,390 | – | -169280 | 39206 | 0.999× | – |
| SLH-DSA | slh-dsa-shake-256f | verification | 2,342 | 2,344 | 2,342 | – | 2458 | -2291 | 1.000× | – |
| SLH-DSA | slh-dsa-shake-256s | key generation | 69,862 | 69,665 | 69,690 | – | -196944 | 24560 | 0.998× | – |
| SLH-DSA | slh-dsa-shake-256s | signing | 889,789 | 888,175 | 888,113 | – | -1614408 | -61850 | 0.998× | – |
| SLH-DSA | slh-dsa-shake-256s | verification | 1,159 | 1,161 | 1,160 | – | 1708 | -667 | 1.001× | – |
| ML-KEM | ml-kem-1024 | decapsulation | 23.9 | 24.0 | 23.9 | – | 83 | -41 | 1.002× | – |
| ML-KEM | ml-kem-1024 | encapsulation | 21.1 | 21.2 | 21.2 | – | 83 | 42 | 1.006× | – |
| ML-KEM | ml-kem-1024 | key generation | 19.4 | 19.5 | 19.7 | – | 84 | 167 | 1.013× | – |
| ML-KEM | ml-kem-512 | decapsulation | 10.2 | 10.2 | 10.3 | – | 42 | 42 | 1.008× | – |
| ML-KEM | ml-kem-512 | encapsulation | 9.2 | 9.1 | 9.2 | – | -124 | 83 | 0.996× | – |
| ML-KEM | ml-kem-512 | key generation | 8.2 | 8.3 | 8.4 | – | 83 | 83 | 1.020× | – |
| ML-KEM | ml-kem-768 | decapsulation | 15.9 | 16.0 | 16.0 | – | 41 | 0 | 1.003× | – |
| ML-KEM | ml-kem-768 | encapsulation | 14.1 | 14.2 | 14.2 | – | 41 | 42 | 1.006× | – |
| ML-KEM | ml-kem-768 | key generation | 13.2 | 13.3 | 13.5 | – | 124 | 167 | 1.022× | – |
| HQC | hqc-1 | decapsulation | 2,732 | 2,735 | 2,736 | – | 3750 | 1084 | 1.002× | – |
| HQC | hqc-1 | encapsulation | 1,789 | 1,790 | 1,791 | – | 1333 | 792 | 1.001× | – |
| HQC | hqc-1 | key generation | 893 | 893 | 893 | – | -42 | 417 | 1.000× | – |
| HQC | hqc-3 | decapsulation | 7,045 | 7,050 | 7,047 | – | 4750 | -2375 | 1.000× | – |
| HQC | hqc-3 | encapsulation | 4,663 | 4,658 | 4,661 | – | -4250 | 2916 | 1.000× | – |
| HQC | hqc-3 | key generation | 2,334 | 2,340 | 2,342 | – | 6250 | 1958 | 1.004× | – |
| HQC | hqc-5 | decapsulation | 17,792 | 17,807 | 17,807 | – | 14873 | 417 | 1.001× | – |
| HQC | hqc-5 | encapsulation | 11,772 | 11,775 | 11,789 | – | 3010 | 13708 | 1.001× | – |
| HQC | hqc-5 | key generation | 5,897 | 5,899 | 5,900 | – | 2750 | 1125 | 1.001× | – |
| classical | dhkem-p256 | decapsulation | 105 | 105 | 105 | 44.3 | 292 | -1 | 1.003× | 0.42× |
| classical | dhkem-p256 | encapsulation | 114 | 114 | 115 | 55.8 | 333 | 208 | 1.005× | 0.49× |
| classical | dhkem-p256 | key generation | 25.4 | 25.5 | 25.7 | 12.0 | 125 | 125 | 1.010× | 0.47× |
| classical | dhkem-x25519 | decapsulation | 37.6 | 37.7 | 37.7 | 43.5 | 125 | 1 | 1.003× | 1.15× |
| classical | dhkem-x25519 | encapsulation | 67.2 | 67.3 | 67.4 | 86.7 | 167 | 42 | 1.003× | 1.29× |
| classical | dhkem-x25519 | key generation | 31.2 | 31.4 | 31.5 | 43.1 | 166 | 43 | 1.007× | 1.37× |
| classical | ecdsa-p256 | key generation | 25.8 | 25.9 | 26.4 | 13.4 | 125 | 500 | 1.024× | 0.51× |
| classical | ecdsa-p256 | signing | 20.4 | 20.6 | 20.7 | 24.0 | 167 | 125 | 1.014× | 1.16× |
| classical | ecdsa-p256 | verification | 54.5 | 54.6 | 54.7 | 56.2 | 84 | 40 | 1.002× | 1.03× |
| classical | ed25519 | key generation | 32.4 | 32.0 | 32.3 | 18.2 | -417 | 375 | 0.999× | 0.56× |
| classical | ed25519 | signing | 30.5 | 30.7 | 30.7 | 22.9 | 209 | 0 | 1.007× | 0.75× |
| classical | ed25519 | verification | 83.2 | 83.3 | 83.3 | 53.7 | 126 | 0 | 1.002× | 0.64× |
| classical | rsa3072-pss | key generation | 440,174 | 417,290 | 448,283 | 613,545 | -22884200 | 30992530 | 1.018× | 1.37× |
| classical | rsa3072-pss | signing | 1,614 | 1,620 | 1,623 | 2,618 | 6540 | 2459 | 1.006× | 1.61× |
| classical | rsa3072-pss | verification | 33.6 | 33.7 | 33.7 | 81.4 | 84 | 0 | 1.002× | 2.42× |

Generated by `analysis/paper_tables.py` from `results/combined/analysis_tables/T8_E3_integration_layers.csv`.
