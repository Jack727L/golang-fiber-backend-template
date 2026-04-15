-- Revert YOUR_SQITCH_PROJECT:create_users from pg

BEGIN;

DROP TABLE IF EXISTS users CASCADE;
DROP FUNCTION IF EXISTS update_updated_at_column();

COMMIT;
