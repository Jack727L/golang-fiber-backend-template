-- Revert YOUR_SQITCH_PROJECT:create_user_tokens from pg

BEGIN;

DROP TABLE IF EXISTS user_tokens;

COMMIT;
