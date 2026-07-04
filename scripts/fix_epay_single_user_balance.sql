-- 单笔易支付用户余额校正（勿批量执行）
-- 用法（务必带 username 与 trade_no 双条件）：
--   psql ... -v username=normanmises -v trade_no=USR129NOQLUFCK1783060917 -f fix_epay_single_user_balance.sql

\set ON_ERROR_STOP on

BEGIN;

WITH topup AS (
  SELECT COALESCE(SUM(t.amount), 0) AS face_cny
  FROM top_ups t
  JOIN users u ON u.id = t.user_id
  WHERE u.username = :'username'
    AND t.trade_no = :'trade_no'
    AND t.status = 'success'
    AND t.payment_provider = 'epay'
),
calc AS (
  SELECT u.id,
         u.username,
         u.quota AS old_quota,
         u.used_quota,
         topup.face_cny,
         GREATEST(topup.face_cny * 500000 - u.used_quota, 0) AS new_quota
  FROM users u
  CROSS JOIN topup
  WHERE u.username = :'username'
    AND topup.face_cny > 0
)
UPDATE users u
SET quota = calc.new_quota
FROM calc
WHERE u.id = calc.id
RETURNING u.id, u.username, calc.face_cny, calc.old_quota, u.quota AS new_quota, calc.used_quota;

-- 无限额度令牌的 remain/used 与用户钱包解耦后归零，避免脏账本继续累积
UPDATE tokens t
SET remain_quota = 0,
    used_quota = 0
FROM users u
WHERE t.user_id = u.id
  AND u.username = :'username'
  AND t.unlimited_quota = true;

COMMIT;
