#!/usr/bin/env python3
"""Run reproducible device-only fake-backed S4 rows; never contacts a backend.

Example: python3 scripts/verification/k0-s4-matrix.py --row all --output /tmp/k0-s4
Fresh-process fake restart evidence does not attest Gate B production durability.
"""
import argparse
import json
from pathlib import Path
import shlex
import subprocess
import time

ROWS = {
    "1": "TestS4MatrixDurableAcceptance",
    "2": "TestS4MatrixDuplicateDelivery",
    "3": "TestS4MatrixRestartRecovery",
    "4": "TestS4MatrixOfflineDrain",
    "5": "TestS4MatrixReceiptApplication",
    "6": "TestS4MatrixForgedStaleWidened",
    "7": "TestS4MatrixInvalidFrames",
}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--row", choices=["all", *ROWS], default="all")
    parser.add_argument("--output", type=Path, required=True)
    args = parser.parse_args()
    root = Path(__file__).resolve().parents[2]
    cwd = root / "backend"
    sha = subprocess.check_output(["git", "rev-parse", "HEAD"], cwd=root, text=True).strip()
    dirty = bool(subprocess.check_output(["git", "status", "--porcelain"], cwd=root, text=True))
    args.output.mkdir(parents=True, exist_ok=True)
    selected = ROWS if args.row == "all" else {args.row: ROWS[args.row]}
    failed = False
    for row, target in selected.items():
        cmd = ["go", "test", "./internal/devicebridge", "-run", f"^{target}$", "-count=1", "-v"]
        log = args.output / f"row-{row}.log"
        start = time.monotonic()
        with log.open("w") as stream:
            result = subprocess.run(cmd, cwd=cwd, stdout=stream, stderr=subprocess.STDOUT)
        record = {"row": row, "target": target, "command": shlex.join(cmd), "cwd": str(cwd),
                  "sha": sha, "dirty": dirty, "exit": result.returncode,
                  "seconds": round(time.monotonic() - start, 6), "log": str(log.resolve()),
                  "scope": "device-only mocks; not joint K0 or Gate B proof"}
        with (args.output / "checks.jsonl").open("a") as stream:
            stream.write(json.dumps(record) + "\n")
        print(json.dumps(record), flush=True)
        failed |= result.returncode != 0
    return 1 if failed else 0


if __name__ == "__main__":
    raise SystemExit(main())
