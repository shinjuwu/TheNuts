-- Add unique constraint on idempotency_key to prevent duplicate transactions
-- Using partial index (WHERE idempotency_key IS NOT NULL) to allow NULL values
CREATE UNIQUE INDEX idx_transactions_idempotency_key 
ON transactions(idempotency_key) 
WHERE idempotency_key IS NOT NULL;

-- Add composite index for better query performance on wallet transaction history
CREATE INDEX idx_transactions_wallet_created 
ON transactions(wallet_id, created_at DESC);

-- Add index for transaction type queries
CREATE INDEX idx_transactions_type_created 
ON transactions(type, created_at DESC);
