# E01d  The first call in a fresh process (µs)

Every block is a new process; its first call is timed apart from the warm calls. Extra = first call − warm median.

| Family | Algorithm | Operation | First call, Mac M1 | First call, Laptop | First call, VM | Extra, Mac M1 | Extra, Laptop | Extra, VM |
|---|---|---|---:|---:|---:|---:|---:|---:|
| ML-DSA | ml-dsa-44 | key generation | 926 | 1,768 | 1,823 | 896 | 1,736 | 1,746 |
| ML-DSA | ml-dsa-44 | signing | 967 | 1,944 | 2,225 | 906 | 1,865 | 2,019 |
| ML-DSA | ml-dsa-44 | verification | 49.8 | 118 | 212 | 21.4 | 85.5 | 133 |
| ML-DSA | ml-dsa-65 | key generation | 938 | 1,972 | 1,855 | 885 | 1,917 | 1,722 |
| ML-DSA | ml-dsa-65 | signing | 997 | 1,948 | 2,236 | 902 | 1,823 | 1,915 |
| ML-DSA | ml-dsa-65 | verification | 75.3 | 185 | 250 | 27.7 | 132 | 142 |
| ML-DSA | ml-dsa-87 | key generation | 1,033 | 1,910 | 1,925 | 942 | 1,825 | 1,741 |
| ML-DSA | ml-dsa-87 | signing | 1,037 | 2,082 | 2,291 | 897 | 1,929 | 1,876 |
| ML-DSA | ml-dsa-87 | verification | 110 | 234 | 333 | 26.9 | 153 | 149 |
| Falcon | falcon-1024 | key generation | 15,942 | 17,941 | 28,839 | 1,084 | 2,102 | -4130.2 |
| Falcon | falcon-1024 | signing | 1,184 | 2,267 | 10,208 | 894 | 1,850 | 2,297 |
| Falcon | falcon-1024 | verification | 66.6 | 173 | 168 | 24.0 | 92.1 | 90.1 |
| Falcon | falcon-512 | key generation | 5,335 | 6,798 | 12,284 | 533 | 1,178 | 703 |
| Falcon | falcon-512 | signing | 1,058 | 1,924 | 5,595 | 912 | 1,714 | 1,928 |
| Falcon | falcon-512 | verification | 35.6 | 88.5 | 97.5 | 14.0 | 47.5 | 61.3 |
| Falcon | falcon-padded-1024 | key generation | 16,504 | 16,269 | 60,493 | 1,690 | 471 | 27,416 |
| Falcon | falcon-padded-1024 | signing | 1,170 | 2,289 | 10,316 | 882 | 1,870 | 2,385 |
| Falcon | falcon-padded-1024 | verification | 57.0 | 147 | 170 | 15.1 | 66.4 | 90.9 |
| Falcon | falcon-padded-512 | key generation | 5,494 | 7,635 | 15,947 | 691 | 2,021 | 4,438 |
| Falcon | falcon-padded-512 | signing | 1,052 | 1,961 | 5,766 | 907 | 1,753 | 2,110 |
| Falcon | falcon-padded-512 | verification | 34.6 | 87.5 | 83.5 | 12.8 | 46.4 | 45.5 |
| SLH-DSA | slh-dsa-sha2-128f | key generation | 1,804 | 2,678 | 2,868 | 916 | 1,788 | 1,887 |
| SLH-DSA | slh-dsa-sha2-128f | signing | 21,741 | 22,710 | 24,558 | 992 | 1,893 | 1,908 |
| SLH-DSA | slh-dsa-sha2-128f | verification | 1,263 | 1,323 | 1,460 | 11.0 | 59.1 | 66.3 |
| SLH-DSA | slh-dsa-sha2-128s | key generation | 57,753 | 57,999 | 61,776 | 942 | 2,023 | 719 |
| SLH-DSA | slh-dsa-sha2-128s | signing | 432,702 | 433,792 | 464,950 | 1,354 | 3,847 | -5163.8 |
| SLH-DSA | slh-dsa-sha2-128s | verification | 444 | 508 | 552 | 16.3 | 70.2 | 81.2 |
| SLH-DSA | slh-dsa-sha2-192f | key generation | 2,216 | 3,234 | 3,027 | 918 | 1,947 | 1,623 |
| SLH-DSA | slh-dsa-sha2-192f | signing | 34,793 | 36,439 | 38,271 | 991 | 2,092 | 1,526 |
| SLH-DSA | slh-dsa-sha2-192f | verification | 1,903 | 1,871 | 2,051 | 79.8 | 31.5 | 55.5 |
| SLH-DSA | slh-dsa-sha2-192s | key generation | 83,642 | 83,984 | 90,279 | 893 | 2,419 | 267 |
| SLH-DSA | slh-dsa-sha2-192s | signing | 759,151 | 775,933 | 848,441 | 1,570 | 1,557 | 10,171 |
| SLH-DSA | slh-dsa-sha2-192s | verification | 636 | 728 | 780 | 2.6 | 86.8 | 55.1 |
| SLH-DSA | slh-dsa-sha2-256f | key generation | 4,433 | 5,267 | 5,044 | 1,010 | 1,805 | 1,370 |
| SLH-DSA | slh-dsa-sha2-256f | signing | 70,385 | 73,037 | 80,654 | 916 | 1,578 | 3,999 |
| SLH-DSA | slh-dsa-sha2-256f | verification | 1,924 | 1,970 | 2,151 | 53.3 | 60.0 | 84.0 |
| SLH-DSA | slh-dsa-sha2-256s | key generation | 55,618 | 57,871 | 60,911 | 962 | 3,004 | 2,234 |
| SLH-DSA | slh-dsa-sha2-256s | signing | 668,942 | 707,663 | 755,337 | 1,187 | 5,393 | 6,399 |
| SLH-DSA | slh-dsa-sha2-256s | verification | 941 | 1,035 | 1,125 | 13.1 | 82.1 | 82.9 |
| SLH-DSA | slh-dsa-shake-128f | key generation | 2,053 | 3,488 | 3,327 | 934 | 1,792 | 1,706 |
| SLH-DSA | slh-dsa-shake-128f | signing | 27,076 | 42,996 | 39,703 | 924 | 1,682 | 2,138 |
| SLH-DSA | slh-dsa-shake-128f | verification | 1,572 | 2,609 | 2,368 | 13.4 | 236 | 121 |
| SLH-DSA | slh-dsa-shake-128s | key generation | 71,921 | 114,661 | 102,131 | 1,099 | 1,343 | -377.0 |
| SLH-DSA | slh-dsa-shake-128s | signing | 554,052 | 866,438 | 788,370 | 1,270 | 4,937 | -773.2 |
| SLH-DSA | slh-dsa-shake-128s | verification | 560 | 854 | 832 | 21.1 | 74.7 | 56.0 |
| SLH-DSA | slh-dsa-shake-192f | key generation | 2,610 | 4,311 | 4,178 | 969 | 1,828 | 1,817 |
| SLH-DSA | slh-dsa-shake-192f | signing | 44,146 | 68,767 | 63,455 | 927 | 1,797 | 1,774 |
| SLH-DSA | slh-dsa-shake-192f | verification | 2,340 | 3,445 | 3,274 | 55.3 | -90.2 | 4.6 |
| SLH-DSA | slh-dsa-shake-192s | key generation | 104,835 | 168,127 | 150,866 | 712 | 2,061 | 543 |
| SLH-DSA | slh-dsa-shake-192s | signing | 971,472 | 1,509,705 | 1,416,279 | -10673.6 | 6,947 | 12,004 |
| SLH-DSA | slh-dsa-shake-192s | verification | 785 | 1,230 | 1,215 | -1.5 | 68.5 | 81.2 |
| SLH-DSA | slh-dsa-shake-256f | key generation | 5,259 | 7,926 | 7,854 | 914 | 1,871 | 1,653 |
| SLH-DSA | slh-dsa-shake-256f | signing | 90,101 | 126,235 | 128,414 | 769 | 1,558 | 1,506 |
| SLH-DSA | slh-dsa-shake-256f | verification | 2,370 | 3,320 | 3,351 | 34.4 | 54.5 | -12.8 |
| SLH-DSA | slh-dsa-shake-256s | key generation | 69,900 | 98,234 | 100,224 | 921 | 2,159 | 203 |
| SLH-DSA | slh-dsa-shake-256s | signing | 875,281 | 1,229,477 | 1,266,923 | -11331.5 | 5,515 | 19,008 |
| SLH-DSA | slh-dsa-shake-256s | verification | 1,147 | 1,623 | 1,780 | -15.9 | 5.4 | 113 |
| ML-KEM | ml-kem-1024 | key generation | 927 | 1,719 | 1,752 | 906 | 1,691 | 1,697 |
| ML-KEM | ml-kem-1024 | encapsulation | 894 | 1,732 | 1,721 | 873 | 1,702 | 1,667 |
| ML-KEM | ml-kem-1024 | decapsulation | 47.1 | 113 | 171 | 23.4 | 79.5 | 112 |
| ML-KEM | ml-kem-512 | key generation | 879 | 1,768 | 1,682 | 871 | 1,756 | 1,658 |
| ML-KEM | ml-kem-512 | encapsulation | 886 | 1,854 | 1,706 | 877 | 1,842 | 1,680 |
| ML-KEM | ml-kem-512 | decapsulation | 30.2 | 85.0 | 133 | 20.1 | 70.5 | 107 |
| ML-KEM | ml-kem-768 | key generation | 897 | 1,715 | 2,010 | 884 | 1,695 | 1,972 |
| ML-KEM | ml-kem-768 | encapsulation | 869 | 1,827 | 1,785 | 855 | 1,807 | 1,746 |
| ML-KEM | ml-kem-768 | decapsulation | 37.9 | 92.8 | 144 | 21.8 | 69.1 | 104 |
| HQC | hqc-1 | key generation | 1,781 | 2,822 | 3,098 | 886 | 1,745 | 1,867 |
| HQC | hqc-1 | encapsulation | 2,675 | 4,015 | 4,522 | 884 | 1,850 | 2,044 |
| HQC | hqc-1 | decapsulation | 2,723 | 3,488 | 4,088 | -5.0 | 186 | 359 |
| HQC | hqc-3 | key generation | 3,220 | 5,131 | 5,666 | 875 | 1,856 | 2,007 |
| HQC | hqc-3 | encapsulation | 5,570 | 8,187 | 9,471 | 902 | 1,675 | 2,201 |
| HQC | hqc-3 | decapsulation | 6,994 | 10,045 | 11,088 | -50.4 | 246 | 146 |
| HQC | hqc-5 | key generation | 6,856 | 9,848 | 10,983 | 935 | 1,885 | 2,129 |
| HQC | hqc-5 | encapsulation | 12,633 | 18,316 | 20,341 | 820 | 2,419 | 2,638 |
| HQC | hqc-5 | decapsulation | 17,907 | 24,384 | 27,793 | 98.7 | 483 | 1,010 |
| classical | dhkem-p256 | key generation | 421 | 791 | 981 | 395 | 754 | 942 |
| classical | dhkem-p256 | encapsulation | 483 | 886 | 977 | 372 | 756 | 839 |
| classical | dhkem-p256 | decapsulation | 145 | 290 | 286 | 46.0 | 173 | 160 |
| classical | dhkem-x25519 | key generation | 400 | 763 | 801 | 368 | 724 | 759 |
| classical | dhkem-x25519 | encapsulation | 431 | 792 | 820 | 363 | 714 | 730 |
| classical | dhkem-x25519 | decapsulation | 72.6 | 127 | 153 | 35.1 | 87.7 | 108 |
| classical | ecdsa-p256 | key generation | 400 | 857 | 1,037 | 374 | 819 | 997 |
| classical | ecdsa-p256 | signing | 378 | 794 | 722 | 358 | 770 | 691 |
| classical | ecdsa-p256 | verification | 92.2 | 142 | 251 | 37.7 | 83.0 | 172 |
| classical | ed25519 | key generation | 393 | 733 | 842 | 360 | 691 | 795 |
| classical | ed25519 | signing | 52.7 | 82.5 | 93.0 | 22.4 | 44.5 | 52.7 |
| classical | ed25519 | verification | 109 | 159 | 166 | 27.8 | 57.3 | 64.3 |
| classical | rsa3072-pss | key generation | 544,980 | 585,699 | 1,021,128 | 143,016 | 62,000 | 510,764 |
| classical | rsa3072-pss | signing | 2,503 | 2,857 | 3,731 | 898 | 1,330 | 1,555 |
| classical | rsa3072-pss | verification | 84.3 | 138 | 211 | 51.3 | 101 | 158 |

Generated by `analysis/paper_tables.py` from `results/combined/analysis_tables/T2_E1_primitive_latency.csv`.
