# Installed as tests/conftest.py inside the liboqs source tree during the image
# build (docker/oqs-gate.sh). It makes liboqs's own ACVP test read the NIST
# vector files that oqs-gate.sh has already downloaded and checked against
# pinned SHA-256 hashes, instead of downloading them a second time. Any other
# network request from the test suite is an error.
import os

import requests

_PREFIX = "https://raw.githubusercontent.com/usnistgov/ACVP-Server/refs/tags/v1.1.0.41/gen-val/json-files/"
_DIR = os.environ.get("KPI_ACVP_DIR", "")


class _LocalResponse:
    def __init__(self, data):
        self.content = data
        self.status_code = 200

    @property
    def text(self):
        return self.content.decode("utf-8")

    def json(self):
        import json
        return json.loads(self.content)

    def raise_for_status(self):
        return None


def _local_get(url, *args, **kwargs):
    if _DIR and url.startswith(_PREFIX):
        path = os.path.join(_DIR, url[len(_PREFIX):])
        with open(path, "rb") as f:
            return _LocalResponse(f.read())
    raise RuntimeError("network access is disabled in the KPI liboqs gate: " + url)


requests.get = _local_get
