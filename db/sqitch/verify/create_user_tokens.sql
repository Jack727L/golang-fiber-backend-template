-- Verify YOUR_SQITCH_PROJECT:create_user_tokens on pg

SELECT id, user_id, access_token, refresh_token,
       access_token_expires_at, refresh_token_expires_at,
       created_at, is_active
  FROM user_tokens
 WHERE FALSE;
