# E04  liboqs build variants on the Mac M1 (Apple M1, ARM with NEON) (µs)

Production build (b1) against the variant(s) this machine ran; medians over five repetitions, 95 % interval of the ratio, Cliff's delta.

| Variant | Family | Algorithm | Operation | Production (b1) | Variant | Variant ÷ b1 | 95 % CI | Cliff's delta |
|---|---|---|---|---:|---:|---:|---|---:|
| portable (b2) | ML-DSA | ml-dsa-44 | key generation | 30.2 | 38.8 | 1.284× | 0.938–1.305 | 0.60 |
| portable (b2) | ML-DSA | ml-dsa-44 | signing | 60.2 | 103 | 1.706× | 1.673–1.736 | 1.00 |
| portable (b2) | ML-DSA | ml-dsa-44 | verification | 28.4 | 39.0 | 1.375× | 0.925–1.420 | 0.60 |
| portable (b2) | ML-DSA | ml-dsa-65 | key generation | 53.0 | 78.3 | 1.479× | 1.471–1.501 | 1.00 |
| portable (b2) | ML-DSA | ml-dsa-65 | signing | 95.5 | 162 | 1.700× | 1.653–1.714 | 1.00 |
| portable (b2) | ML-DSA | ml-dsa-65 | verification | 47.6 | 63.7 | 1.336× | 1.307–1.362 | 1.00 |
| portable (b2) | ML-DSA | ml-dsa-87 | key generation | 90.9 | 109 | 1.197× | 1.190–1.258 | 1.00 |
| portable (b2) | ML-DSA | ml-dsa-87 | signing | 140 | 215 | 1.539× | 1.529–1.567 | 1.00 |
| portable (b2) | ML-DSA | ml-dsa-87 | verification | 83.5 | 106 | 1.274× | 1.257–1.298 | 1.00 |
| portable (b2) | Falcon | falcon-1024 | key generation | 14,858 | 32,049 | 2.157× | 2.082–2.170 | 1.00 |
| portable (b2) | Falcon | falcon-1024 | signing | 290 | 7,861 | 27.069× | 27.007–27.379 | 1.00 |
| portable (b2) | Falcon | falcon-1024 | verification | 42.6 | 53.6 | 1.259× | 1.099–1.280 | 1.00 |
| portable (b2) | Falcon | falcon-512 | key generation | 4,802 | 10,781 | 2.245× | 2.241–2.249 | 1.00 |
| portable (b2) | Falcon | falcon-512 | signing | 145 | 3,611 | 24.867× | 24.718–24.967 | 1.00 |
| portable (b2) | Falcon | falcon-512 | verification | 21.7 | 27.5 | 1.272× | 1.248–1.403 | 1.00 |
| portable (b2) | Falcon | falcon-padded-1024 | key generation | 14,814 | 31,853 | 2.150× | 2.138–2.161 | 1.00 |
| portable (b2) | Falcon | falcon-padded-1024 | signing | 287 | 7,860 | 27.354× | 26.910–27.418 | 1.00 |
| portable (b2) | Falcon | falcon-padded-1024 | verification | 41.9 | 53.3 | 1.274× | 1.252–1.275 | 1.00 |
| portable (b2) | Falcon | falcon-padded-512 | key generation | 4,803 | 10,777 | 2.244× | 2.239–2.247 | 1.00 |
| portable (b2) | Falcon | falcon-padded-512 | signing | 145 | 3,611 | 24.946× | 24.892–24.996 | 1.00 |
| portable (b2) | Falcon | falcon-padded-512 | verification | 21.8 | 27.1 | 1.245× | 1.229–1.270 | 1.00 |
| portable (b2) | SLH-DSA | slh-dsa-sha2-128f | key generation | 888 | 888 | 1.000× | 0.999–1.107 | 0.44 |
| portable (b2) | SLH-DSA | slh-dsa-sha2-128f | signing | 20,748 | 20,753 | 1.000× | 0.999–1.001 | 0.04 |
| portable (b2) | SLH-DSA | slh-dsa-sha2-128f | verification | 1,252 | 1,251 | 0.999× | 0.996–1.003 | -0.20 |
| portable (b2) | SLH-DSA | slh-dsa-sha2-128s | key generation | 56,810 | 56,804 | 1.000× | 0.999–1.000 | -0.84 |
| portable (b2) | SLH-DSA | slh-dsa-sha2-128s | signing | 431,348 | 431,342 | 1.000× | 0.999–1.000 | 0.04 |
| portable (b2) | SLH-DSA | slh-dsa-sha2-128s | verification | 428 | 430 | 1.005× | 0.989–1.012 | 0.44 |
| portable (b2) | SLH-DSA | slh-dsa-sha2-192f | key generation | 1,298 | 1,298 | 1.000× | 1.000–1.000 | 0.60 |
| portable (b2) | SLH-DSA | slh-dsa-sha2-192f | signing | 33,802 | 33,843 | 1.001× | 1.000–1.009 | 0.76 |
| portable (b2) | SLH-DSA | slh-dsa-sha2-192f | verification | 1,823 | 1,827 | 1.002× | 0.997–1.005 | 0.20 |
| portable (b2) | SLH-DSA | slh-dsa-sha2-192s | key generation | 82,750 | 82,757 | 1.000× | 1.000–1.001 | 0.68 |
| portable (b2) | SLH-DSA | slh-dsa-sha2-192s | signing | 757,581 | 757,562 | 1.000× | 0.999–1.000 | -0.20 |
| portable (b2) | SLH-DSA | slh-dsa-sha2-192s | verification | 633 | 633 | 1.000× | 0.996–1.008 | -0.04 |
| portable (b2) | SLH-DSA | slh-dsa-sha2-256f | key generation | 3,423 | 3,423 | 1.000× | 0.999–1.000 | -0.52 |
| portable (b2) | SLH-DSA | slh-dsa-sha2-256f | signing | 69,469 | 69,485 | 1.000× | 1.000–1.066 | 0.60 |
| portable (b2) | SLH-DSA | slh-dsa-sha2-256f | verification | 1,870 | 1,869 | 0.999× | 0.996–1.086 | -0.04 |
| portable (b2) | SLH-DSA | slh-dsa-sha2-256s | key generation | 54,656 | 54,657 | 1.000× | 1.000–1.007 | 0.20 |
| portable (b2) | SLH-DSA | slh-dsa-sha2-256s | signing | 667,755 | 667,873 | 1.000× | 0.996–1.000 | 0.12 |
| portable (b2) | SLH-DSA | slh-dsa-sha2-256s | verification | 928 | 928 | 1.000× | 0.995–1.007 | 0.20 |
| portable (b2) | SLH-DSA | slh-dsa-shake-128f | key generation | 1,120 | 1,120 | 1.000× | 0.996–1.041 | 0.12 |
| portable (b2) | SLH-DSA | slh-dsa-shake-128f | signing | 26,152 | 26,269 | 1.004× | 0.990–1.040 | 0.60 |
| portable (b2) | SLH-DSA | slh-dsa-shake-128f | verification | 1,558 | 1,564 | 1.004× | 0.724–1.007 | -0.04 |
| portable (b2) | SLH-DSA | slh-dsa-shake-128s | key generation | 70,822 | 71,890 | 1.015× | 0.998–1.021 | 0.60 |
| portable (b2) | SLH-DSA | slh-dsa-shake-128s | signing | 552,782 | 551,204 | 0.997× | 0.931–1.004 | -0.44 |
| portable (b2) | SLH-DSA | slh-dsa-shake-128s | verification | 539 | 540 | 1.001× | 0.994–1.007 | 0.44 |
| portable (b2) | SLH-DSA | slh-dsa-shake-192f | key generation | 1,641 | 1,638 | 0.999× | 0.996–1.003 | -0.44 |
| portable (b2) | SLH-DSA | slh-dsa-shake-192f | signing | 43,220 | 43,580 | 1.008× | 0.998–1.094 | 0.68 |
| portable (b2) | SLH-DSA | slh-dsa-shake-192f | verification | 2,285 | 2,287 | 1.001× | 0.948–1.030 | -0.04 |
| portable (b2) | SLH-DSA | slh-dsa-shake-192s | key generation | 104,123 | 103,987 | 0.999× | 0.948–1.094 | -0.28 |
| portable (b2) | SLH-DSA | slh-dsa-shake-192s | signing | 982,146 | 976,879 | 0.995× | 0.979–1.648 | -0.12 |
| portable (b2) | SLH-DSA | slh-dsa-shake-192s | verification | 786 | 788 | 1.002× | 0.991–1.007 | 0.04 |
| portable (b2) | SLH-DSA | slh-dsa-shake-256f | key generation | 4,345 | 4,328 | 0.996× | 0.927–1.002 | -0.52 |
| portable (b2) | SLH-DSA | slh-dsa-shake-256f | signing | 89,331 | 89,551 | 1.002× | 0.989–1.012 | 0.36 |
| portable (b2) | SLH-DSA | slh-dsa-shake-256f | verification | 2,335 | 2,335 | 1.000× | 0.999–1.074 | 0.20 |
| portable (b2) | SLH-DSA | slh-dsa-shake-256s | key generation | 68,979 | 69,052 | 1.001× | 0.992–1.056 | 0.28 |
| portable (b2) | SLH-DSA | slh-dsa-shake-256s | signing | 886,612 | 878,605 | 0.991× | 0.644–1.007 | -0.44 |
| portable (b2) | SLH-DSA | slh-dsa-shake-256s | verification | 1,163 | 1,162 | 0.999× | 0.970–1.004 | -0.12 |
| portable (b2) | ML-KEM | ml-kem-1024 | decapsulation | 23.7 | 33.2 | 1.404× | 1.351–1.451 | 1.00 |
| portable (b2) | ML-KEM | ml-kem-1024 | encapsulation | 21.0 | 28.8 | 1.370× | 1.354–1.396 | 1.00 |
| portable (b2) | ML-KEM | ml-kem-1024 | key generation | 20.2 | 32.7 | 1.623× | 1.519–1.658 | 1.00 |
| portable (b2) | ML-KEM | ml-kem-512 | decapsulation | 10.1 | 14.9 | 1.470× | 1.440–1.560 | 1.00 |
| portable (b2) | ML-KEM | ml-kem-512 | encapsulation | 9.1 | 13.2 | 1.448× | 1.344–1.645 | 1.00 |
| portable (b2) | ML-KEM | ml-kem-512 | key generation | 8.3 | 12.9 | 1.546× | 1.518–1.576 | 1.00 |
| portable (b2) | ML-KEM | ml-kem-768 | decapsulation | 16.1 | 22.5 | 1.400× | 1.361–1.453 | 1.00 |
| portable (b2) | ML-KEM | ml-kem-768 | encapsulation | 14.0 | 19.7 | 1.404× | 1.384–1.434 | 1.00 |
| portable (b2) | ML-KEM | ml-kem-768 | key generation | 13.3 | 21.6 | 1.619× | 1.588–1.650 | 1.00 |
| portable (b2) | HQC | hqc-1 | decapsulation | 2,728 | 2,354 | 0.863× | 0.863–0.865 | -1.00 |
| portable (b2) | HQC | hqc-1 | encapsulation | 1,791 | 1,544 | 0.862× | 0.861–0.863 | -1.00 |
| portable (b2) | HQC | hqc-1 | key generation | 894 | 774 | 0.866× | 0.854–0.876 | -1.00 |
| portable (b2) | HQC | hqc-3 | decapsulation | 7,045 | 7,039 | 0.999× | 0.998–1.000 | -1.00 |
| portable (b2) | HQC | hqc-3 | encapsulation | 4,668 | 4,667 | 1.000× | 0.997–1.001 | -0.44 |
| portable (b2) | HQC | hqc-3 | key generation | 2,345 | 2,346 | 1.001× | 0.998–1.002 | 0.20 |
| portable (b2) | HQC | hqc-5 | decapsulation | 17,809 | 17,813 | 1.000× | 1.000–1.002 | 0.36 |
| portable (b2) | HQC | hqc-5 | encapsulation | 11,813 | 11,799 | 0.999× | 0.997–1.001 | -0.52 |
| portable (b2) | HQC | hqc-5 | key generation | 5,922 | 5,915 | 0.999× | 0.997–1.001 | -0.68 |
| portable (b2) | classical | dhkem-p256 | decapsulation | 99.5 | 99.6 | 1.001× | 0.984–1.008 | 0.60 |
| portable (b2) | classical | dhkem-p256 | encapsulation | 110 | 111 | 1.004× | 0.985–1.022 | 0.68 |
| portable (b2) | classical | dhkem-p256 | key generation | 26.0 | 25.6 | 0.984× | 0.831–1.014 | -0.20 |
| portable (b2) | classical | dhkem-x25519 | decapsulation | 37.5 | 37.7 | 1.005× | 1.004–1.007 | 1.00 |
| portable (b2) | classical | dhkem-x25519 | encapsulation | 67.4 | 67.5 | 1.002× | 0.997–1.020 | 0.28 |
| portable (b2) | classical | dhkem-x25519 | key generation | 31.9 | 31.4 | 0.984× | 0.981–1.000 | -0.36 |
| portable (b2) | classical | ecdsa-p256 | key generation | 25.9 | 25.8 | 0.997× | 0.977–1.018 | -0.76 |
| portable (b2) | classical | ecdsa-p256 | signing | 19.9 | 20.0 | 1.003× | 1.001–1.009 | 0.92 |
| portable (b2) | classical | ecdsa-p256 | verification | 54.5 | 54.6 | 1.001× | 0.998–1.005 | 0.44 |
| portable (b2) | classical | ed25519 | key generation | 32.1 | 32.2 | 1.002× | 0.984–1.019 | 0.60 |
| portable (b2) | classical | ed25519 | signing | 30.3 | 30.3 | 1.002× | 0.999–1.002 | 0.60 |
| portable (b2) | classical | ed25519 | verification | 81.6 | 81.8 | 1.003× | 1.000–1.015 | 0.68 |
| portable (b2) | classical | rsa3072-pss | key generation | 401,964 | 433,148 | 1.078× | 1.000–1.178 | 0.84 |
| portable (b2) | classical | rsa3072-pss | signing | 1,605 | 1,603 | 0.998× | 0.984–1.001 | -0.20 |
| portable (b2) | classical | rsa3072-pss | verification | 33.0 | 33.2 | 1.004× | 0.987–1.022 | 0.68 |
| native (b3) | ML-DSA | ml-dsa-44 | key generation | 30.2 | 30.8 | 1.017× | 0.743–1.024 | 0.36 |
| native (b3) | ML-DSA | ml-dsa-44 | signing | 60.2 | 60.1 | 0.998× | 0.987–1.016 | -0.12 |
| native (b3) | ML-DSA | ml-dsa-44 | verification | 28.4 | 28.0 | 0.984× | 0.663–1.003 | -0.36 |
| native (b3) | ML-DSA | ml-dsa-65 | key generation | 53.0 | 52.9 | 0.999× | 0.986–1.015 | -0.04 |
| native (b3) | ML-DSA | ml-dsa-65 | signing | 95.5 | 96.0 | 1.006× | 0.978–1.015 | 0.12 |
| native (b3) | ML-DSA | ml-dsa-65 | verification | 47.6 | 47.6 | 1.000× | 0.977–1.047 | -0.28 |
| native (b3) | ML-DSA | ml-dsa-87 | key generation | 90.9 | 89.5 | 0.984× | 0.979–1.016 | -0.36 |
| native (b3) | ML-DSA | ml-dsa-87 | signing | 140 | 138 | 0.992× | 0.980–1.008 | -0.60 |
| native (b3) | ML-DSA | ml-dsa-87 | verification | 83.5 | 84.6 | 1.013× | 0.994–1.017 | 0.36 |
| native (b3) | Falcon | falcon-1024 | key generation | 14,858 | 14,772 | 0.994× | 0.993–1.002 | -0.12 |
| native (b3) | Falcon | falcon-1024 | signing | 290 | 287 | 0.990× | 0.987–1.016 | 0.04 |
| native (b3) | Falcon | falcon-1024 | verification | 42.6 | 42.0 | 0.985× | 0.860–1.006 | -0.20 |
| native (b3) | Falcon | falcon-512 | key generation | 4,802 | 4,804 | 1.000× | 0.984–1.002 | 0.36 |
| native (b3) | Falcon | falcon-512 | signing | 145 | 145 | 0.996× | 0.990–1.002 | -0.52 |
| native (b3) | Falcon | falcon-512 | verification | 21.7 | 21.6 | 0.999× | 0.985–1.020 | -0.28 |
| native (b3) | Falcon | falcon-padded-1024 | key generation | 14,814 | 14,788 | 0.998× | 0.993–1.004 | -0.36 |
| native (b3) | Falcon | falcon-padded-1024 | signing | 287 | 291 | 1.013× | 0.990–1.021 | 0.60 |
| native (b3) | Falcon | falcon-padded-1024 | verification | 41.9 | 42.0 | 1.003× | 0.985–1.019 | 0.44 |
| native (b3) | Falcon | falcon-padded-512 | key generation | 4,803 | 4,804 | 1.000× | 0.999–1.055 | 0.60 |
| native (b3) | Falcon | falcon-padded-512 | signing | 145 | 145 | 1.002× | 0.999–1.102 | 0.60 |
| native (b3) | Falcon | falcon-padded-512 | verification | 21.8 | 21.7 | 0.995× | 0.982–1.148 | 0.04 |
| native (b3) | SLH-DSA | slh-dsa-sha2-128f | key generation | 888 | 888 | 1.001× | 1.000–1.008 | 0.76 |
| native (b3) | SLH-DSA | slh-dsa-sha2-128f | signing | 20,748 | 20,733 | 0.999× | 0.999–1.015 | -0.60 |
| native (b3) | SLH-DSA | slh-dsa-sha2-128f | verification | 1,252 | 1,254 | 1.001× | 0.995–1.003 | -0.12 |
| native (b3) | SLH-DSA | slh-dsa-sha2-128s | key generation | 56,810 | 56,820 | 1.000× | 1.000–1.009 | 0.68 |
| native (b3) | SLH-DSA | slh-dsa-sha2-128s | signing | 431,348 | 431,463 | 1.000× | 0.999–1.001 | 0.36 |
| native (b3) | SLH-DSA | slh-dsa-sha2-128s | verification | 428 | 431 | 1.007× | 0.992–1.013 | 0.60 |
| native (b3) | SLH-DSA | slh-dsa-sha2-192f | key generation | 1,298 | 1,298 | 1.000× | 1.000–1.000 | 0.60 |
| native (b3) | SLH-DSA | slh-dsa-sha2-192f | signing | 33,802 | 33,797 | 1.000× | 0.998–1.001 | -0.36 |
| native (b3) | SLH-DSA | slh-dsa-sha2-192f | verification | 1,823 | 1,826 | 1.002× | 0.998–1.004 | 0.36 |
| native (b3) | SLH-DSA | slh-dsa-sha2-192s | key generation | 82,750 | 82,765 | 1.000× | 1.000–1.001 | 0.68 |
| native (b3) | SLH-DSA | slh-dsa-sha2-192s | signing | 757,581 | 757,635 | 1.000× | 0.999–1.009 | 0.20 |
| native (b3) | SLH-DSA | slh-dsa-sha2-192s | verification | 633 | 635 | 1.003× | 0.999–1.118 | 0.68 |
| native (b3) | SLH-DSA | slh-dsa-sha2-256f | key generation | 3,423 | 3,424 | 1.000× | 0.999–1.000 | 0.20 |
| native (b3) | SLH-DSA | slh-dsa-sha2-256f | signing | 69,469 | 69,474 | 1.000× | 0.999–1.000 | 0.20 |
| native (b3) | SLH-DSA | slh-dsa-sha2-256f | verification | 1,870 | 1,865 | 0.997× | 0.995–1.002 | -0.60 |
| native (b3) | SLH-DSA | slh-dsa-sha2-256s | key generation | 54,656 | 54,661 | 1.000× | 1.000–1.001 | 0.52 |
| native (b3) | SLH-DSA | slh-dsa-sha2-256s | signing | 667,755 | 667,960 | 1.000× | 0.996–1.001 | 0.04 |
| native (b3) | SLH-DSA | slh-dsa-sha2-256s | verification | 928 | 928 | 1.000× | 0.997–1.007 | 0.04 |
| native (b3) | SLH-DSA | slh-dsa-shake-128f | key generation | 1,120 | 1,118 | 0.999× | 0.995–1.006 | -0.44 |
| native (b3) | SLH-DSA | slh-dsa-shake-128f | signing | 26,152 | 26,367 | 1.008× | 0.994–1.013 | 0.28 |
| native (b3) | SLH-DSA | slh-dsa-shake-128f | verification | 1,558 | 1,562 | 1.002× | 0.723–1.005 | 0.20 |
| native (b3) | SLH-DSA | slh-dsa-shake-128s | key generation | 70,822 | 71,266 | 1.006× | 0.995–1.084 | 0.60 |
| native (b3) | SLH-DSA | slh-dsa-shake-128s | signing | 552,782 | 547,305 | 0.990× | 0.925–1.003 | -0.68 |
| native (b3) | SLH-DSA | slh-dsa-shake-128s | verification | 539 | 543 | 1.007× | 0.999–1.013 | 0.44 |
| native (b3) | SLH-DSA | slh-dsa-shake-192f | key generation | 1,641 | 1,647 | 1.004× | 0.999–1.084 | 0.52 |
| native (b3) | SLH-DSA | slh-dsa-shake-192f | signing | 43,220 | 43,267 | 1.001× | 0.992–1.011 | 0.20 |
| native (b3) | SLH-DSA | slh-dsa-shake-192f | verification | 2,285 | 2,291 | 1.003× | 0.949–1.008 | 0.28 |
| native (b3) | SLH-DSA | slh-dsa-shake-192s | key generation | 104,123 | 104,144 | 1.000× | 0.949–1.011 | -0.12 |
| native (b3) | SLH-DSA | slh-dsa-shake-192s | signing | 982,146 | 978,208 | 0.996× | 0.980–1.006 | -0.20 |
| native (b3) | SLH-DSA | slh-dsa-shake-192s | verification | 786 | 791 | 1.006× | 0.995–1.009 | 0.28 |
| native (b3) | SLH-DSA | slh-dsa-shake-256f | key generation | 4,345 | 4,335 | 0.998× | 0.929–1.004 | -0.20 |
| native (b3) | SLH-DSA | slh-dsa-shake-256f | signing | 89,331 | 89,993 | 1.007× | 0.992–1.014 | 0.44 |
| native (b3) | SLH-DSA | slh-dsa-shake-256f | verification | 2,335 | 2,330 | 0.997× | 0.995–1.004 | -0.20 |
| native (b3) | SLH-DSA | slh-dsa-shake-256s | key generation | 68,979 | 68,839 | 0.998× | 0.989–1.013 | 0.04 |
| native (b3) | SLH-DSA | slh-dsa-shake-256s | signing | 886,612 | 883,339 | 0.996× | 0.647–1.010 | -0.12 |
| native (b3) | SLH-DSA | slh-dsa-shake-256s | verification | 1,163 | 1,162 | 0.999× | 0.970–1.071 | -0.20 |
| native (b3) | ML-KEM | ml-kem-1024 | decapsulation | 23.7 | 24.0 | 1.016× | 0.995–1.023 | 0.36 |
| native (b3) | ML-KEM | ml-kem-1024 | encapsulation | 21.0 | 21.4 | 1.018× | 0.993–1.445 | 0.52 |
| native (b3) | ML-KEM | ml-kem-1024 | key generation | 20.2 | 19.9 | 0.988× | 0.925–1.011 | -0.44 |
| native (b3) | ML-KEM | ml-kem-512 | decapsulation | 10.1 | 10.1 | 1.000× | 0.980–1.000 | 0.20 |
| native (b3) | ML-KEM | ml-kem-512 | encapsulation | 9.1 | 9.2 | 1.014× | 0.941–1.106 | 0.68 |
| native (b3) | ML-KEM | ml-kem-512 | key generation | 8.3 | 8.3 | 1.001× | 0.981–1.026 | -0.04 |
| native (b3) | ML-KEM | ml-kem-768 | decapsulation | 16.1 | 15.8 | 0.982× | 0.955–1.003 | -0.28 |
| native (b3) | ML-KEM | ml-kem-768 | encapsulation | 14.0 | 14.2 | 1.015× | 0.995–1.078 | 0.20 |
| native (b3) | ML-KEM | ml-kem-768 | key generation | 13.3 | 13.5 | 1.016× | 0.982–1.401 | 0.36 |
| native (b3) | HQC | hqc-1 | decapsulation | 2,728 | 2,355 | 0.863× | 0.863–0.865 | -1.00 |
| native (b3) | HQC | hqc-1 | encapsulation | 1,791 | 1,542 | 0.861× | 0.861–0.863 | -1.00 |
| native (b3) | HQC | hqc-1 | key generation | 894 | 776 | 0.868× | 0.862–0.878 | -1.00 |
| native (b3) | HQC | hqc-3 | decapsulation | 7,045 | 7,040 | 0.999× | 0.999–1.000 | -0.92 |
| native (b3) | HQC | hqc-3 | encapsulation | 4,668 | 4,656 | 0.997× | 0.995–0.998 | -1.00 |
| native (b3) | HQC | hqc-3 | key generation | 2,345 | 2,343 | 0.999× | 0.998–1.002 | 0.04 |
| native (b3) | HQC | hqc-5 | decapsulation | 17,809 | 17,799 | 0.999× | 0.999–1.002 | -0.04 |
| native (b3) | HQC | hqc-5 | encapsulation | 11,813 | 11,790 | 0.998× | 0.997–0.999 | -1.00 |
| native (b3) | HQC | hqc-5 | key generation | 5,922 | 5,913 | 0.999× | 0.997–1.000 | -0.76 |
| native (b3) | classical | dhkem-p256 | decapsulation | 99.5 | 99.5 | 1.001× | 0.983–1.002 | 0.44 |
| native (b3) | classical | dhkem-p256 | encapsulation | 110 | 110 | 1.001× | 0.982–1.014 | 0.60 |
| native (b3) | classical | dhkem-p256 | key generation | 26.0 | 25.9 | 0.997× | 0.842–1.018 | -0.28 |
| native (b3) | classical | dhkem-x25519 | decapsulation | 37.5 | 38.2 | 1.019× | 1.002–1.026 | 1.00 |
| native (b3) | classical | dhkem-x25519 | encapsulation | 67.4 | 67.4 | 1.001× | 0.997–1.020 | 0.28 |
| native (b3) | classical | dhkem-x25519 | key generation | 31.9 | 31.5 | 0.985× | 0.981–1.006 | -0.12 |
| native (b3) | classical | ecdsa-p256 | key generation | 25.9 | 25.9 | 1.001× | 0.977–1.017 | -0.28 |
| native (b3) | classical | ecdsa-p256 | signing | 19.9 | 20.0 | 1.005× | 1.001–1.024 | 0.92 |
| native (b3) | classical | ecdsa-p256 | verification | 54.5 | 54.6 | 1.002× | 0.999–1.003 | 0.52 |
| native (b3) | classical | ed25519 | key generation | 32.1 | 32.1 | 0.998× | 0.980–1.019 | -0.44 |
| native (b3) | classical | ed25519 | signing | 30.3 | 30.4 | 1.005× | 1.000–1.021 | 0.84 |
| native (b3) | classical | ed25519 | verification | 81.6 | 81.8 | 1.003× | 0.999–1.005 | 0.36 |
| native (b3) | classical | rsa3072-pss | key generation | 401,964 | 418,854 | 1.042× | 0.931–1.172 | 0.36 |
| native (b3) | classical | rsa3072-pss | signing | 1,605 | 1,602 | 0.998× | 0.983–1.013 | 0.12 |
| native (b3) | classical | rsa3072-pss | verification | 33.0 | 33.0 | 1.001× | 0.983–1.019 | 0.68 |

Generated by `analysis/paper_tables.py` from `results/combined/analysis_tables/T7_E4_build_sensitivity.csv`.
