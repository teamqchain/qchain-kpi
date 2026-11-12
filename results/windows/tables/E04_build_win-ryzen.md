# E04  liboqs build variants on the Laptop (AMD Ryzen 5 7520U, x86 with AVX2) (µs)

Production build (b1) against the variant(s) this machine ran; medians over five repetitions, 95 % interval of the ratio, Cliff's delta.

| Variant | Family | Algorithm | Operation | Production (b1) | Variant | Variant ÷ b1 | 95 % CI | Cliff's delta |
|---|---|---|---|---:|---:|---:|---|---:|
| portable (b2) | ML-DSA | ml-dsa-44 | key generation | 32.0 | 75.9 | 2.372× | 2.277–2.387 | 1.00 |
| portable (b2) | ML-DSA | ml-dsa-44 | signing | 79.2 | 242 | 3.062× | 2.993–3.118 | 1.00 |
| portable (b2) | ML-DSA | ml-dsa-44 | verification | 32.0 | 75.2 | 2.347× | 2.330–2.364 | 1.00 |
| portable (b2) | ML-DSA | ml-dsa-65 | key generation | 55.0 | 127 | 2.301× | 2.294–2.318 | 1.00 |
| portable (b2) | ML-DSA | ml-dsa-65 | signing | 125 | 379 | 3.025× | 3.008–3.072 | 1.00 |
| portable (b2) | ML-DSA | ml-dsa-65 | verification | 53.1 | 118 | 2.221× | 2.201–2.231 | 1.00 |
| portable (b2) | ML-DSA | ml-dsa-87 | key generation | 85.2 | 186 | 2.182× | 2.111–2.206 | 1.00 |
| portable (b2) | ML-DSA | ml-dsa-87 | signing | 153 | 481 | 3.137× | 3.117–3.183 | 1.00 |
| portable (b2) | ML-DSA | ml-dsa-87 | verification | 81.3 | 191 | 2.347× | 2.326–2.368 | 1.00 |
| portable (b2) | Falcon | falcon-1024 | key generation | 15,838 | 33,060 | 2.087× | 2.066–2.109 | 1.00 |
| portable (b2) | Falcon | falcon-1024 | signing | 417 | 8,238 | 19.759× | 19.514–19.854 | 1.00 |
| portable (b2) | Falcon | falcon-1024 | verification | 81.1 | 77.9 | 0.961× | 0.626–0.970 | -1.00 |
| portable (b2) | Falcon | falcon-512 | key generation | 5,620 | 11,931 | 2.123× | 2.097–2.143 | 1.00 |
| portable (b2) | Falcon | falcon-512 | signing | 210 | 3,795 | 18.044× | 17.961–18.345 | 1.00 |
| portable (b2) | Falcon | falcon-512 | verification | 41.0 | 38.8 | 0.946× | 0.942–0.955 | -1.00 |
| portable (b2) | Falcon | falcon-padded-1024 | key generation | 15,798 | 33,579 | 2.126× | 2.075–2.143 | 1.00 |
| portable (b2) | Falcon | falcon-padded-1024 | signing | 419 | 8,249 | 19.667× | 19.572–19.870 | 1.00 |
| portable (b2) | Falcon | falcon-padded-1024 | verification | 80.7 | 77.9 | 0.965× | 0.962–0.969 | -1.00 |
| portable (b2) | Falcon | falcon-padded-512 | key generation | 5,614 | 11,886 | 2.117× | 2.105–2.140 | 1.00 |
| portable (b2) | Falcon | falcon-padded-512 | signing | 208 | 3,798 | 18.267× | 18.036–18.357 | 1.00 |
| portable (b2) | Falcon | falcon-padded-512 | verification | 41.0 | 38.9 | 0.948× | 0.942–0.953 | -1.00 |
| portable (b2) | SLH-DSA | slh-dsa-sha2-128f | key generation | 890 | 882 | 0.991× | 0.982–1.004 | -0.52 |
| portable (b2) | SLH-DSA | slh-dsa-sha2-128f | signing | 20,817 | 20,759 | 0.997× | 0.989–1.000 | -0.60 |
| portable (b2) | SLH-DSA | slh-dsa-sha2-128f | verification | 1,264 | 1,242 | 0.983× | 0.977–0.993 | -0.92 |
| portable (b2) | SLH-DSA | slh-dsa-sha2-128s | key generation | 55,977 | 55,871 | 0.998× | 0.993–1.006 | -0.20 |
| portable (b2) | SLH-DSA | slh-dsa-sha2-128s | signing | 429,945 | 430,926 | 1.002× | 0.994–1.006 | -0.04 |
| portable (b2) | SLH-DSA | slh-dsa-sha2-128s | verification | 438 | 431 | 0.984× | 0.975–0.993 | -1.00 |
| portable (b2) | SLH-DSA | slh-dsa-sha2-192f | key generation | 1,286 | 1,282 | 0.997× | 0.995–1.001 | -0.68 |
| portable (b2) | SLH-DSA | slh-dsa-sha2-192f | signing | 34,347 | 34,279 | 0.998× | 0.996–1.003 | -0.28 |
| portable (b2) | SLH-DSA | slh-dsa-sha2-192f | verification | 1,840 | 1,832 | 0.996× | 0.989–1.005 | -0.60 |
| portable (b2) | SLH-DSA | slh-dsa-sha2-192s | key generation | 81,565 | 81,285 | 0.997× | 0.993–1.038 | -0.60 |
| portable (b2) | SLH-DSA | slh-dsa-sha2-192s | signing | 774,376 | 775,048 | 1.001× | 0.997–1.007 | 0.04 |
| portable (b2) | SLH-DSA | slh-dsa-sha2-192s | verification | 641 | 637 | 0.994× | 0.985–0.998 | -0.84 |
| portable (b2) | SLH-DSA | slh-dsa-sha2-256f | key generation | 3,462 | 3,463 | 1.000× | 0.997–1.004 | 0.12 |
| portable (b2) | SLH-DSA | slh-dsa-sha2-256f | signing | 71,459 | 71,351 | 0.998× | 0.996–1.000 | -0.76 |
| portable (b2) | SLH-DSA | slh-dsa-sha2-256f | verification | 1,910 | 1,899 | 0.994× | 0.990–0.996 | -1.00 |
| portable (b2) | SLH-DSA | slh-dsa-sha2-256s | key generation | 54,867 | 54,844 | 1.000× | 0.998–1.002 | 0.12 |
| portable (b2) | SLH-DSA | slh-dsa-sha2-256s | signing | 702,270 | 704,840 | 1.004× | 0.997–1.005 | 0.44 |
| portable (b2) | SLH-DSA | slh-dsa-sha2-256s | verification | 953 | 944 | 0.991× | 0.983–1.001 | -0.76 |
| portable (b2) | SLH-DSA | slh-dsa-shake-128f | key generation | 1,696 | 1,651 | 0.974× | 0.950–0.979 | -1.00 |
| portable (b2) | SLH-DSA | slh-dsa-shake-128f | signing | 41,314 | 38,628 | 0.935× | 0.924–0.941 | -1.00 |
| portable (b2) | SLH-DSA | slh-dsa-shake-128f | verification | 2,373 | 2,359 | 0.994× | 0.977–1.016 | -0.68 |
| portable (b2) | SLH-DSA | slh-dsa-shake-128s | key generation | 113,318 | 107,967 | 0.953× | 0.916–0.960 | -1.00 |
| portable (b2) | SLH-DSA | slh-dsa-shake-128s | signing | 861,502 | 828,479 | 0.962× | 0.958–0.965 | -1.00 |
| portable (b2) | SLH-DSA | slh-dsa-shake-128s | verification | 780 | 782 | 1.003× | 0.994–1.014 | 0.36 |
| portable (b2) | SLH-DSA | slh-dsa-shake-192f | key generation | 2,483 | 2,459 | 0.990× | 0.987–0.995 | -1.00 |
| portable (b2) | SLH-DSA | slh-dsa-shake-192f | signing | 66,970 | 66,413 | 0.992× | 0.985–1.000 | -0.84 |
| portable (b2) | SLH-DSA | slh-dsa-shake-192f | verification | 3,535 | 3,539 | 1.001× | 0.995–1.004 | -0.12 |
| portable (b2) | SLH-DSA | slh-dsa-shake-192s | key generation | 166,066 | 162,175 | 0.977× | 0.976–0.993 | -1.00 |
| portable (b2) | SLH-DSA | slh-dsa-shake-192s | signing | 1,502,757 | 1,478,415 | 0.984× | 0.977–0.994 | -1.00 |
| portable (b2) | SLH-DSA | slh-dsa-shake-192s | verification | 1,162 | 1,160 | 0.999× | 0.992–1.003 | -0.36 |
| portable (b2) | SLH-DSA | slh-dsa-shake-256f | key generation | 6,054 | 6,045 | 0.998× | 0.997–1.001 | -0.52 |
| portable (b2) | SLH-DSA | slh-dsa-shake-256f | signing | 124,677 | 124,149 | 0.996× | 0.994–0.998 | -1.00 |
| portable (b2) | SLH-DSA | slh-dsa-shake-256f | verification | 3,266 | 3,253 | 0.996× | 0.992–0.999 | -0.92 |
| portable (b2) | SLH-DSA | slh-dsa-shake-256s | key generation | 96,074 | 95,931 | 0.999× | 0.997–1.022 | -0.44 |
| portable (b2) | SLH-DSA | slh-dsa-shake-256s | signing | 1,223,962 | 1,210,544 | 0.989× | 0.988–0.991 | -1.00 |
| portable (b2) | SLH-DSA | slh-dsa-shake-256s | verification | 1,617 | 1,617 | 1.000× | 0.994–1.003 | -0.28 |
| portable (b2) | ML-KEM | ml-kem-1024 | decapsulation | 34.0 | 58.2 | 1.711× | 1.681–1.759 | 1.00 |
| portable (b2) | ML-KEM | ml-kem-1024 | encapsulation | 29.4 | 51.7 | 1.756× | 1.719–1.826 | 1.00 |
| portable (b2) | ML-KEM | ml-kem-1024 | key generation | 28.5 | 51.5 | 1.804× | 1.792–1.820 | 1.00 |
| portable (b2) | ML-KEM | ml-kem-512 | decapsulation | 14.5 | 27.4 | 1.885× | 1.867–1.896 | 1.00 |
| portable (b2) | ML-KEM | ml-kem-512 | encapsulation | 12.8 | 23.6 | 1.838× | 1.822–1.850 | 1.00 |
| portable (b2) | ML-KEM | ml-kem-512 | key generation | 11.9 | 21.5 | 1.801× | 1.785–1.810 | 1.00 |
| portable (b2) | ML-KEM | ml-kem-768 | decapsulation | 23.7 | 40.8 | 1.720× | 1.716–1.775 | 1.00 |
| portable (b2) | ML-KEM | ml-kem-768 | encapsulation | 20.6 | 35.5 | 1.728× | 1.721–1.797 | 1.00 |
| portable (b2) | ML-KEM | ml-kem-768 | key generation | 20.3 | 34.6 | 1.707× | 1.691–1.721 | 1.00 |
| portable (b2) | HQC | hqc-1 | decapsulation | 3,301 | 3,292 | 0.997× | 0.996–1.002 | -0.60 |
| portable (b2) | HQC | hqc-1 | encapsulation | 2,165 | 2,161 | 0.998× | 0.997–1.001 | -0.52 |
| portable (b2) | HQC | hqc-1 | key generation | 1,078 | 1,079 | 1.001× | 0.997–1.003 | 0.04 |
| portable (b2) | HQC | hqc-3 | decapsulation | 9,799 | 9,800 | 1.000× | 0.998–1.002 | 0.12 |
| portable (b2) | HQC | hqc-3 | encapsulation | 6,512 | 6,515 | 1.001× | 0.998–1.004 | -0.12 |
| portable (b2) | HQC | hqc-3 | key generation | 3,275 | 3,267 | 0.998× | 0.997–1.000 | -0.76 |
| portable (b2) | HQC | hqc-5 | decapsulation | 23,901 | 23,920 | 1.001× | 0.998–1.002 | 0.12 |
| portable (b2) | HQC | hqc-5 | encapsulation | 15,897 | 15,895 | 1.000× | 0.997–1.002 | -0.20 |
| portable (b2) | HQC | hqc-5 | key generation | 7,963 | 7,967 | 1.000× | 0.997–1.002 | -0.04 |
| portable (b2) | classical | dhkem-p256 | decapsulation | 117 | 116 | 0.997× | 0.989–1.007 | -0.20 |
| portable (b2) | classical | dhkem-p256 | encapsulation | 131 | 131 | 0.999× | 0.994–1.008 | -0.20 |
| portable (b2) | classical | dhkem-p256 | key generation | 37.5 | 37.1 | 0.991× | 0.981–1.010 | -0.52 |
| portable (b2) | classical | dhkem-x25519 | decapsulation | 39.2 | 38.9 | 0.994× | 0.986–1.011 | -0.28 |
| portable (b2) | classical | dhkem-x25519 | encapsulation | 77.8 | 77.3 | 0.993× | 0.970–1.015 | -0.36 |
| portable (b2) | classical | dhkem-x25519 | key generation | 39.9 | 40.0 | 1.003× | 0.993–1.006 | 0.20 |
| portable (b2) | classical | ecdsa-p256 | key generation | 38.4 | 38.2 | 0.994× | 0.984–0.999 | -0.92 |
| portable (b2) | classical | ecdsa-p256 | signing | 24.4 | 24.3 | 0.997× | 0.989–1.016 | 0.04 |
| portable (b2) | classical | ecdsa-p256 | verification | 58.9 | 58.9 | 1.000× | 0.985–1.006 | -0.28 |
| portable (b2) | classical | ed25519 | key generation | 41.7 | 41.7 | 1.000× | 0.992–1.008 | 0.12 |
| portable (b2) | classical | ed25519 | signing | 38.1 | 37.9 | 0.996× | 0.992–0.999 | -0.92 |
| portable (b2) | classical | ed25519 | verification | 101 | 101 | 1.000× | 0.998–1.008 | 0.20 |
| portable (b2) | classical | rsa3072-pss | key generation | 523,698 | 493,638 | 0.943× | 0.809–1.062 | -0.52 |
| portable (b2) | classical | rsa3072-pss | signing | 1,527 | 1,525 | 0.999× | 0.997–1.004 | 0.12 |
| portable (b2) | classical | rsa3072-pss | verification | 36.9 | 36.8 | 0.998× | 0.993–1.009 | -0.36 |
| native (b3) | ML-DSA | ml-dsa-44 | key generation | 32.0 | 31.6 | 0.987× | 0.982–0.994 | -1.00 |
| native (b3) | ML-DSA | ml-dsa-44 | signing | 79.2 | 77.8 | 0.983× | 0.960–1.044 | -0.28 |
| native (b3) | ML-DSA | ml-dsa-44 | verification | 32.0 | 31.4 | 0.982× | 0.975–0.997 | -0.92 |
| native (b3) | ML-DSA | ml-dsa-65 | key generation | 55.0 | 54.3 | 0.987× | 0.983–0.990 | -1.00 |
| native (b3) | ML-DSA | ml-dsa-65 | signing | 125 | 123 | 0.986× | 0.981–0.996 | -0.92 |
| native (b3) | ML-DSA | ml-dsa-65 | verification | 53.1 | 52.4 | 0.986× | 0.983–0.991 | -1.00 |
| native (b3) | ML-DSA | ml-dsa-87 | key generation | 85.2 | 84.2 | 0.988× | 0.956–1.009 | -0.44 |
| native (b3) | ML-DSA | ml-dsa-87 | signing | 153 | 151 | 0.984× | 0.977–0.994 | -1.00 |
| native (b3) | ML-DSA | ml-dsa-87 | verification | 81.3 | 80.1 | 0.985× | 0.979–0.994 | -1.00 |
| native (b3) | Falcon | falcon-1024 | key generation | 15,838 | 15,715 | 0.992× | 0.987–0.997 | -1.00 |
| native (b3) | Falcon | falcon-1024 | signing | 417 | 414 | 0.993× | 0.980–1.006 | -0.76 |
| native (b3) | Falcon | falcon-1024 | verification | 81.1 | 80.8 | 0.996× | 0.649–1.007 | -0.20 |
| native (b3) | Falcon | falcon-512 | key generation | 5,620 | 5,612 | 0.999× | 0.995–1.007 | -0.12 |
| native (b3) | Falcon | falcon-512 | signing | 210 | 208 | 0.988× | 0.983–1.004 | -0.44 |
| native (b3) | Falcon | falcon-512 | verification | 41.0 | 41.4 | 1.008× | 1.000–1.018 | 0.72 |
| native (b3) | Falcon | falcon-padded-1024 | key generation | 15,798 | 15,706 | 0.994× | 0.984–1.002 | -0.52 |
| native (b3) | Falcon | falcon-padded-1024 | signing | 419 | 414 | 0.988× | 0.981–1.012 | -0.44 |
| native (b3) | Falcon | falcon-padded-1024 | verification | 80.7 | 80.9 | 1.002× | 0.999–1.007 | 0.52 |
| native (b3) | Falcon | falcon-padded-512 | key generation | 5,614 | 5,635 | 1.004× | 1.000–1.010 | 0.84 |
| native (b3) | Falcon | falcon-padded-512 | signing | 208 | 207 | 0.998× | 0.985–1.010 | -0.04 |
| native (b3) | Falcon | falcon-padded-512 | verification | 41.0 | 41.1 | 1.003× | 0.996–1.008 | 0.52 |
| native (b3) | SLH-DSA | slh-dsa-sha2-128f | key generation | 890 | 935 | 1.050× | 1.040–1.062 | 1.00 |
| native (b3) | SLH-DSA | slh-dsa-sha2-128f | signing | 20,817 | 21,834 | 1.049× | 1.047–1.052 | 1.00 |
| native (b3) | SLH-DSA | slh-dsa-sha2-128f | verification | 1,264 | 1,331 | 1.053× | 1.043–1.061 | 1.00 |
| native (b3) | SLH-DSA | slh-dsa-sha2-128s | key generation | 55,977 | 59,525 | 1.063× | 1.058–1.066 | 1.00 |
| native (b3) | SLH-DSA | slh-dsa-sha2-128s | signing | 429,945 | 453,423 | 1.055× | 1.047–1.058 | 1.00 |
| native (b3) | SLH-DSA | slh-dsa-sha2-128s | verification | 438 | 454 | 1.037× | 1.031–1.050 | 1.00 |
| native (b3) | SLH-DSA | slh-dsa-sha2-192f | key generation | 1,286 | 1,360 | 1.058× | 1.055–1.061 | 1.00 |
| native (b3) | SLH-DSA | slh-dsa-sha2-192f | signing | 34,347 | 35,579 | 1.036× | 1.034–1.042 | 1.00 |
| native (b3) | SLH-DSA | slh-dsa-sha2-192f | verification | 1,840 | 1,925 | 1.046× | 1.042–1.048 | 1.00 |
| native (b3) | SLH-DSA | slh-dsa-sha2-192s | key generation | 81,565 | 86,068 | 1.055× | 1.053–1.059 | 1.00 |
| native (b3) | SLH-DSA | slh-dsa-sha2-192s | signing | 774,376 | 797,966 | 1.030× | 1.026–1.032 | 1.00 |
| native (b3) | SLH-DSA | slh-dsa-sha2-192s | verification | 641 | 662 | 1.033× | 1.028–1.038 | 1.00 |
| native (b3) | SLH-DSA | slh-dsa-sha2-256f | key generation | 3,462 | 3,605 | 1.041× | 1.037–1.061 | 1.00 |
| native (b3) | SLH-DSA | slh-dsa-sha2-256f | signing | 71,459 | 73,258 | 1.025× | 1.023–1.027 | 1.00 |
| native (b3) | SLH-DSA | slh-dsa-sha2-256f | verification | 1,910 | 1,975 | 1.034× | 1.029–1.048 | 1.00 |
| native (b3) | SLH-DSA | slh-dsa-sha2-256s | key generation | 54,867 | 57,103 | 1.041× | 1.038–1.058 | 1.00 |
| native (b3) | SLH-DSA | slh-dsa-sha2-256s | signing | 702,270 | 710,393 | 1.012× | 1.007–1.027 | 1.00 |
| native (b3) | SLH-DSA | slh-dsa-sha2-256s | verification | 953 | 977 | 1.025× | 1.018–1.032 | 1.00 |
| native (b3) | SLH-DSA | slh-dsa-shake-128f | key generation | 1,696 | 1,300 | 0.766× | 0.761–0.769 | -1.00 |
| native (b3) | SLH-DSA | slh-dsa-shake-128f | signing | 41,314 | 30,265 | 0.733× | 0.732–0.742 | -1.00 |
| native (b3) | SLH-DSA | slh-dsa-shake-128f | verification | 2,373 | 1,817 | 0.766× | 0.753–0.770 | -1.00 |
| native (b3) | SLH-DSA | slh-dsa-shake-128s | key generation | 113,318 | 82,127 | 0.725× | 0.724–0.728 | -1.00 |
| native (b3) | SLH-DSA | slh-dsa-shake-128s | signing | 861,502 | 630,422 | 0.732× | 0.730–0.733 | -1.00 |
| native (b3) | SLH-DSA | slh-dsa-shake-128s | verification | 780 | 627 | 0.804× | 0.799–0.814 | -1.00 |
| native (b3) | SLH-DSA | slh-dsa-shake-192f | key generation | 2,483 | 1,904 | 0.767× | 0.766–0.769 | -1.00 |
| native (b3) | SLH-DSA | slh-dsa-shake-192f | signing | 66,970 | 49,891 | 0.745× | 0.743–0.747 | -1.00 |
| native (b3) | SLH-DSA | slh-dsa-shake-192f | verification | 3,535 | 2,651 | 0.750× | 0.747–0.754 | -1.00 |
| native (b3) | SLH-DSA | slh-dsa-shake-192s | key generation | 166,066 | 120,371 | 0.725× | 0.724–0.726 | -1.00 |
| native (b3) | SLH-DSA | slh-dsa-shake-192s | signing | 1,502,757 | 1,120,718 | 0.746× | 0.744–0.748 | -1.00 |
| native (b3) | SLH-DSA | slh-dsa-shake-192s | verification | 1,162 | 913 | 0.786× | 0.782–0.789 | -1.00 |
| native (b3) | SLH-DSA | slh-dsa-shake-256f | key generation | 6,054 | 5,043 | 0.833× | 0.829–0.835 | -1.00 |
| native (b3) | SLH-DSA | slh-dsa-shake-256f | signing | 124,677 | 103,434 | 0.830× | 0.827–0.831 | -1.00 |
| native (b3) | SLH-DSA | slh-dsa-shake-256f | verification | 3,266 | 2,718 | 0.832× | 0.829–0.835 | -1.00 |
| native (b3) | SLH-DSA | slh-dsa-shake-256s | key generation | 96,074 | 79,984 | 0.833× | 0.830–0.834 | -1.00 |
| native (b3) | SLH-DSA | slh-dsa-shake-256s | signing | 1,223,962 | 1,013,352 | 0.828× | 0.827–0.830 | -1.00 |
| native (b3) | SLH-DSA | slh-dsa-shake-256s | verification | 1,617 | 1,354 | 0.837× | 0.832–0.839 | -1.00 |
| native (b3) | ML-KEM | ml-kem-1024 | decapsulation | 34.0 | 33.6 | 0.987× | 0.970–1.007 | -0.44 |
| native (b3) | ML-KEM | ml-kem-1024 | encapsulation | 29.4 | 29.0 | 0.987× | 0.966–1.011 | -0.36 |
| native (b3) | ML-KEM | ml-kem-1024 | key generation | 28.5 | 28.1 | 0.985× | 0.980–0.991 | -1.00 |
| native (b3) | ML-KEM | ml-kem-512 | decapsulation | 14.5 | 14.2 | 0.979× | 0.967–0.983 | -1.00 |
| native (b3) | ML-KEM | ml-kem-512 | encapsulation | 12.8 | 12.6 | 0.979× | 0.972–0.984 | -1.00 |
| native (b3) | ML-KEM | ml-kem-512 | key generation | 11.9 | 11.7 | 0.982× | 0.975–0.998 | -0.88 |
| native (b3) | ML-KEM | ml-kem-768 | decapsulation | 23.7 | 23.3 | 0.983× | 0.977–0.985 | -1.00 |
| native (b3) | ML-KEM | ml-kem-768 | encapsulation | 20.6 | 20.1 | 0.980× | 0.976–0.988 | -1.00 |
| native (b3) | ML-KEM | ml-kem-768 | key generation | 20.3 | 20.1 | 0.991× | 0.983–0.996 | -1.00 |
| native (b3) | HQC | hqc-1 | decapsulation | 3,301 | 3,183 | 0.964× | 0.962–0.967 | -1.00 |
| native (b3) | HQC | hqc-1 | encapsulation | 2,165 | 2,093 | 0.967× | 0.965–0.970 | -1.00 |
| native (b3) | HQC | hqc-1 | key generation | 1,078 | 1,042 | 0.967× | 0.964–0.968 | -1.00 |
| native (b3) | HQC | hqc-3 | decapsulation | 9,799 | 9,501 | 0.970× | 0.968–0.972 | -1.00 |
| native (b3) | HQC | hqc-3 | encapsulation | 6,512 | 6,323 | 0.971× | 0.969–0.973 | -1.00 |
| native (b3) | HQC | hqc-3 | key generation | 3,275 | 3,179 | 0.971× | 0.968–0.972 | -1.00 |
| native (b3) | HQC | hqc-5 | decapsulation | 23,901 | 23,398 | 0.979× | 0.976–0.979 | -1.00 |
| native (b3) | HQC | hqc-5 | encapsulation | 15,897 | 15,557 | 0.979× | 0.976–0.980 | -1.00 |
| native (b3) | HQC | hqc-5 | key generation | 7,963 | 7,788 | 0.978× | 0.976–0.980 | -1.00 |
| native (b3) | classical | dhkem-p256 | decapsulation | 117 | 117 | 0.999× | 0.991–1.010 | -0.04 |
| native (b3) | classical | dhkem-p256 | encapsulation | 131 | 131 | 1.006× | 1.000–1.014 | 0.84 |
| native (b3) | classical | dhkem-p256 | key generation | 37.5 | 37.5 | 1.000× | 0.994–1.015 | 0.24 |
| native (b3) | classical | dhkem-x25519 | decapsulation | 39.2 | 39.1 | 1.000× | 0.995–1.017 | -0.04 |
| native (b3) | classical | dhkem-x25519 | encapsulation | 77.8 | 76.7 | 0.986× | 0.963–1.008 | -0.68 |
| native (b3) | classical | dhkem-x25519 | key generation | 39.9 | 40.0 | 1.002× | 0.996–1.006 | 0.28 |
| native (b3) | classical | ecdsa-p256 | key generation | 38.4 | 38.2 | 0.994× | 0.981–1.006 | -0.68 |
| native (b3) | classical | ecdsa-p256 | signing | 24.4 | 24.1 | 0.990× | 0.981–1.004 | -0.52 |
| native (b3) | classical | ecdsa-p256 | verification | 58.9 | 58.6 | 0.996× | 0.983–1.008 | -0.28 |
| native (b3) | classical | ed25519 | key generation | 41.7 | 41.8 | 1.004× | 0.994–1.012 | 0.44 |
| native (b3) | classical | ed25519 | signing | 38.1 | 38.0 | 0.997× | 0.990–1.002 | -0.68 |
| native (b3) | classical | ed25519 | verification | 101 | 101 | 1.000× | 0.998–1.005 | 0.04 |
| native (b3) | classical | rsa3072-pss | key generation | 523,698 | 532,939 | 1.018× | 0.824–1.146 | -0.12 |
| native (b3) | classical | rsa3072-pss | signing | 1,527 | 1,525 | 0.999× | 0.997–1.003 | -0.04 |
| native (b3) | classical | rsa3072-pss | verification | 36.9 | 36.9 | 1.001× | 0.999–1.004 | 0.32 |

Generated by `analysis/paper_tables.py` from `results/combined/analysis_tables/T7_E4_build_sensitivity.csv`.
