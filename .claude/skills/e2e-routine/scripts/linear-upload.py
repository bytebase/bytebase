#!/usr/bin/env python3
"""Upload one evidence file to a Linear issue.

    linear-upload.py stat <file>   -> {"filename", "contentType", "size"} for prepare_attachment_upload
    linear-upload.py put <file>    -> PUTs the file to the uploadRequest read as JSON on stdin

`put` sends every header of uploadRequest verbatim, as the signed URL
requires, and must run within 60 seconds of prepare_attachment_upload. Exit 0
means the bytes landed; then call create_attachment_from_upload with the
assetUrl. On failure it prints the HTTP status and the upload host, so a host
the cloud environment's network policy blocks is named in the report.
"""
import json
import os
import subprocess
import sys
from urllib.parse import urlparse

CONTENT_TYPES = {".png": "image/png", ".md": "text/markdown", ".zip": "application/zip", ".log": "text/plain"}


def upload_name(path):
    # Playwright names every screenshot test-failed-1.png; the test-results
    # directory (spec, title hash, project, retry) is what tells them apart.
    parent = os.path.basename(os.path.dirname(os.path.abspath(path)))
    return f"{parent[:90]}-{os.path.basename(path)}"


def stat(path):
    ext = os.path.splitext(path)[1].lower()
    print(json.dumps({"filename": upload_name(path),
                      "contentType": CONTENT_TYPES.get(ext, "application/octet-stream"),
                      "size": os.path.getsize(path)}))
    return 0


def put(path):
    req = json.load(sys.stdin)
    req = req.get("uploadRequest", req)
    url, headers = req["url"], req.get("headers") or {}
    if isinstance(headers, list):  # [{"key": ..., "value": ...}]
        headers = {h["key"]: h["value"] for h in headers}
    # The signed URL lives 60 seconds, so a host the network policy drops must
    # fail inside that window instead of hanging.
    cmd = ["curl", "-sS", "--connect-timeout", "10", "--max-time", "45", "-o", "/dev/null", "-w", "%{http_code}",
           "-X", "PUT", "--data-binary", f"@{path}"]
    for k, v in headers.items():
        cmd += ["-H", f"{k}: {v}"]
    proc = subprocess.run(cmd + [url], capture_output=True, text=True)
    host = urlparse(url).hostname
    code = proc.stdout.strip()
    if proc.returncode == 0 and code.startswith("2"):
        print(f"uploaded {os.path.getsize(path)} bytes to {host}")
        return 0
    print(f"upload failed: host {host}, HTTP {code or 'none'}, curl exit {proc.returncode}: {proc.stderr.strip()[:300]}",
          file=sys.stderr)
    return 1


if __name__ == "__main__":
    if len(sys.argv) != 3 or sys.argv[1] not in ("stat", "put") or not os.path.isfile(sys.argv[2]):
        print(__doc__)
        sys.exit(2)
    sys.exit(stat(sys.argv[2]) if sys.argv[1] == "stat" else put(sys.argv[2]))
