#!/usr/bin/env python3
"""Read-only legacy scope preflight. Never repairs, guesses or deletes rows."""
import argparse
import json
import sqlite3
from pathlib import Path

p = argparse.ArgumentParser()
p.add_argument("database", type=Path)
a = p.parse_args()
db = sqlite3.connect(a.database.resolve().as_uri() + "?mode=ro", uri=True)
try:
    devices = dict(db.execute("SELECT device_id,owner_id FROM device_bridge_devices"))
    report = {}
    for table in ("device_bridge_inbox_journal", "device_bridge_outbox"):
        bad = []
        count = 0
        for ident, payload in db.execute("SELECT command_id,payload FROM " + table):
            count += 1
            try:
                f = json.loads(payload)
                device = f.get("device_id") if isinstance(f, dict) else None
                owner = f.get("owner_id") if isinstance(f, dict) else None
                valid = device in devices and (owner is None or owner == devices[device])
            except (ValueError, TypeError):
                valid = False
            if not valid:
                bad.append(ident)
        report[table] = {"rows": count, "rejected_count": len(bad), "rejected_command_ids": bad}
    print(json.dumps(report, indent=2))
    raise SystemExit(2 if any(r["rejected_count"] for r in report.values()) else 0)
finally:
    db.close()
