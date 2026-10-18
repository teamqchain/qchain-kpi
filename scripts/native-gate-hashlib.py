
# ---------------------------------------------------------------------------
# Native macOS gate only: appended to docker/acvp_conftest.py by
# scripts/native-setup.sh (the image's gate does not use it).
#
# macOS's /usr/bin/python3 links LibreSSL 2.8.3, whose hashlib has no
# SHA-512/224 or SHA-512/256. liboqs's ACVP test needs both to pre-hash the
# HashML-DSA vectors (hashlib.new("sha512_224" | "sha512_256")); Debian's
# Python in the image has them. Here they are computed by the OpenSSL 3.0.22
# built for the native harness (KPI_LIBCRYPTO), checked first against the
# FIPS 180-4 examples for "abc". Every other hash stays Python's own.
import ctypes
import hashlib

_crypto = ctypes.CDLL(os.environ["KPI_LIBCRYPTO"])
_crypto.EVP_get_digestbyname.restype = ctypes.c_void_p
_crypto.EVP_get_digestbyname.argtypes = [ctypes.c_char_p]
_crypto.EVP_Digest.restype = ctypes.c_int
_crypto.EVP_Digest.argtypes = [ctypes.c_char_p, ctypes.c_size_t, ctypes.c_char_p,
                               ctypes.POINTER(ctypes.c_uint), ctypes.c_void_p, ctypes.c_void_p]

_OPENSSL_NAMES = {"sha512_224": b"SHA512-224", "sha512_256": b"SHA512-256"}


class _OpenSSLDigest:
    def __init__(self, name, data=b""):
        self._md = _crypto.EVP_get_digestbyname(_OPENSSL_NAMES[name])
        if not self._md:
            raise ValueError("OpenSSL has no digest " + name)
        self._buf = bytearray(data)

    def update(self, data):
        self._buf += data

    def digest(self):
        out = ctypes.create_string_buffer(64)
        n = ctypes.c_uint(0)
        data = bytes(self._buf)
        if _crypto.EVP_Digest(data, len(data), out, ctypes.byref(n), self._md, None) != 1:
            raise ValueError("EVP_Digest failed")
        return out.raw[:n.value]

    def hexdigest(self):
        return self.digest().hex()


_hashlib_new = hashlib.new


def _new(name, data=b"", **kwargs):
    if name.lower() in _OPENSSL_NAMES:
        return _OpenSSLDigest(name.lower(), data)
    return _hashlib_new(name, data, **kwargs)


hashlib.new = _new

for _name, _want in (("sha512_224", "4634270f707b6a54daae7530460842e20e37ed265ceee9a43e8924aa"),
                     ("sha512_256", "53048e2681941ef99b2e29b76b4c7dabe4c2d0c634fc6d46e0e2f13107e7af23")):
    if hashlib.new(_name, b"abc").hexdigest() != _want:
        raise RuntimeError(_name + " from OpenSSL does not match FIPS 180-4's example")
