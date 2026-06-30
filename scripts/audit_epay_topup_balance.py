#!/usr/bin/env python3
"""
Audit epay top-up balances: compare (quota + used) vs sum(face amount) at 1:1 QuotaPerUnit.

Usage (on origin host):
  python3 scripts/audit_epay_topup_balance.py
  python3 scripts/audit_epay_topup_balance.py --min-mult 1.05 --max-mult 0.95

Requires: docker access to newapi-postgres container.
"""

from __future__ import annotations

import argparse
import subprocess
import sys

QUOTA_PER_UNIT = 500_000


def run_sql(sql: str) -> str:
    cmd = [
        "docker",
        "exec",
        "newapi-postgres",
        "psql",
        "-U",
        "root",
        "-d",
        "new-api",
        "-t",
        "-A",
        "-F",
        "\t",
        "-c",
        sql,
    ]
    return subprocess.check_output(cmd, text=True)


def main() -> int:
    parser = argparse.ArgumentParser(description="Audit epay user balance vs top-up face value")
    parser.add_argument("--over", type=float, default=1.05, help="Flag mult above this (default 1.05)")
    parser.add_argument("--under", type=float, default=0.95, help="Flag mult below this (default 0.95)")
    parser.add_argument("--username", type=str, default="", help="Filter single username")
    args = parser.parse_args()

    user_filter = ""
    if args.username:
        user_filter = f"AND us.username = '{args.username.replace(chr(39), chr(39)+chr(39))}'"

    sql = f"""
WITH t AS (
  SELECT user_id, SUM(amount) AS face_cny, SUM(money) AS paid_cny, COUNT(*) AS orders,
         MIN(create_time) AS first_ts, MAX(create_time) AS last_ts
  FROM top_ups
  WHERE status = 'success' AND payment_provider = 'epay'
  GROUP BY user_id
)
SELECT us.id, us.username, t.face_cny, t.paid_cny, t.orders,
       us.quota, us.used_quota,
       round((us.quota + us.used_quota)::numeric / {QUOTA_PER_UNIT}, 4) AS total_yuan,
       round((us.quota + us.used_quota)::numeric / NULLIF(t.face_cny * {QUOTA_PER_UNIT}, 0), 4) AS mult,
       to_char(to_timestamp(t.first_ts), 'YYYY-MM-DD') AS first_day,
       to_char(to_timestamp(t.last_ts), 'YYYY-MM-DD') AS last_day
FROM t
JOIN users us ON us.id = t.user_id
WHERE t.face_cny > 0 {user_filter}
ORDER BY mult DESC NULLS LAST;
"""
    raw = run_sql(sql).strip()
    if not raw:
        print("No epay top-up users found.")
        return 0

    rows = []
    for line in raw.splitlines():
        parts = line.split("\t")
        if len(parts) < 11:
            continue
        rows.append(
            {
                "id": int(parts[0]),
                "username": parts[1],
                "face_cny": float(parts[2]),
                "paid_cny": float(parts[3]),
                "orders": int(parts[4]),
                "quota": int(parts[5]),
                "used_quota": int(parts[6]),
                "total_yuan": float(parts[7]),
                "mult": float(parts[8]),
                "first_day": parts[9],
                "last_day": parts[10],
            }
        )

    over = [r for r in rows if r["mult"] > args.over]
    under = [r for r in rows if r["mult"] < args.under]
    ok = [r for r in rows if args.under <= r["mult"] <= args.over]

    print(f"=== Epay balance audit (1:1 expected mult≈1.0, QuotaPerUnit={QUOTA_PER_UNIT}) ===")
    print(f"Total users: {len(rows)}  OK [{args.under},{args.over}]: {len(ok)}  OVER >{args.over}: {len(over)}  UNDER <{args.under}: {len(under)}")
    print()

    def print_table(title: str, items: list[dict]) -> None:
        if not items:
            return
        print(title)
        print(f"{'username':<20} {'face':>8} {'total¥':>10} {'mult':>8} {'remain¥':>10} {'used¥':>8} {'period'}")
        for r in items:
            remain = r["quota"] / QUOTA_PER_UNIT
            used = r["used_quota"] / QUOTA_PER_UNIT
            print(
                f"{r['username']:<20} {r['face_cny']:>8.0f} {r['total_yuan']:>10.2f} {r['mult']:>8.3f} "
                f"{remain:>10.2f} {used:>8.2f} {r['first_day']}~{r['last_day']}"
            )
        print()

    print_table(f"--- OVER-CREDITED (mult > {args.over}) ---", over)
    print_table(f"--- UNDER-CREDITED (mult < {args.under}) ---", under)

    return 0


if __name__ == "__main__":
    raise SystemExit(main())
