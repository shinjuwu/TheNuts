-- Drop indexes created in 000002_add_idempotency_constraint.up.sql
DROP INDEX IF EXISTS idx_transactions_type_created;
DROP INDEX IF EXISTS idx_transactions_wallet_created;
DROP INDEX IF EXISTS idx_transactions_idempotency_key;
