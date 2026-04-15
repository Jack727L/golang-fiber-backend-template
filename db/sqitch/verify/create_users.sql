-- Verify YOUR_SQITCH_PROJECT:create_users on pg

SELECT id, email, name, hashed_password, created_at, updated_at, last_active_at
  FROM users
 WHERE FALSE;
