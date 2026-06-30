-- Correct a user's remaining quota to 1:1 epay face value minus used_quota.
-- Run inside newapi-postgres after reviewing audit_epay_topup_balance.py output.
--
-- Example (cc22pp):
--   psql ... -v username=cc22pp -f fix_epay_user_balance.sql

\set ON_ERROR_STOP on

BEGIN;

WITH topup AS (
  SELECT COALESCE(SUM(amount), 0) AS face_cny
  FROM top_ups t
  JOIN users u ON u.id = t.user_id
  WHERE u.username = :'username'
    AND t.status = 'success'
    AND t.payment_provider = 'epay'
),
calc AS (
  SELECT u.id,
         u.username,
         u.quota AS old_quota,
         u.used_quota,
         topup.face_cny,
         (topup.face_cny * 500000 - u.used_quota) AS new_quota
  FROM users u
  CROSS JOIN topup
  WHERE u.username = :'username'
)
UPDATE users u
SET quota = GREATEST(calc.new_quota, 0)
FROM calc
WHERE u.id = calc.id
RETURNING u.id, u.username, calc.face_cny, calc.old_quota, u.quota AS new_quota, calc.used_quota;

COMMIT;
