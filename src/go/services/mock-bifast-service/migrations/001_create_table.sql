-- migrations/001_create_table.sql
-- Create tables for Mock BI-FAST Service

-- Enable UUID extension (if not already enabled)
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- ============================================================
-- TABLE: accounts
-- Purpose: Mock bank accounts for BI-FAST testing
-- ============================================================
CREATE TABLE IF NOT EXISTS accounts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    
    -- Account identification
    bank_code VARCHAR(8) NOT NULL,
    account_number VARCHAR(50) NOT NULL,
    account_name VARCHAR(255) NOT NULL,
    account_type VARCHAR(20) NOT NULL DEFAULT 'savings',
    
    -- Balance information
    balance DECIMAL(20, 2) NOT NULL DEFAULT 0,
    currency VARCHAR(3) NOT NULL DEFAULT 'IDR',
    
    -- Account status
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    is_active BOOLEAN NOT NULL DEFAULT true,
    is_blocked BOOLEAN NOT NULL DEFAULT false,
    
    -- Timestamps
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    
    -- Constraints
    CONSTRAINT uq_accounts_bank_account UNIQUE (bank_code, account_number),
    CONSTRAINT check_balance_positive CHECK (balance >= 0),
    CONSTRAINT check_status_valid CHECK (status IN ('active', 'blocked', 'closed')),
    CONSTRAINT check_account_type_valid CHECK (account_type IN ('savings', 'checking', 'current', 'deposit'))
);

-- Indexes for accounts table
CREATE INDEX idx_accounts_bank_code ON accounts(bank_code);
CREATE INDEX idx_accounts_account_number ON accounts(account_number);
CREATE INDEX idx_accounts_status ON accounts(status);
CREATE INDEX idx_accounts_is_active ON accounts(is_active);
CREATE INDEX idx_accounts_created_at ON accounts(created_at DESC);

-- Composite index for account lookup (most common query)
CREATE INDEX idx_accounts_lookup ON accounts(bank_code, account_number, is_active);

-- Comments for accounts table
COMMENT ON TABLE accounts IS 'Mock bank accounts for BI-FAST transaction testing';
COMMENT ON COLUMN accounts.bank_code IS 'Bank code (8 characters SWIFT format, e.g., CENAIDJA)';
COMMENT ON COLUMN accounts.account_number IS 'Account number (1-50 alphanumeric)';
COMMENT ON COLUMN accounts.account_type IS 'Account type: savings, checking, current, deposit';
COMMENT ON COLUMN accounts.balance IS 'Current account balance for testing purposes';
COMMENT ON COLUMN accounts.status IS 'Account status: active, blocked, closed';
COMMENT ON COLUMN accounts.is_active IS 'Quick flag to check if account is active';
COMMENT ON COLUMN accounts.is_blocked IS 'Quick flag to check if account is blocked';

-- ============================================================
-- TABLE: transactions
-- Purpose: Store all BI-FAST transaction records
-- ============================================================
CREATE TABLE IF NOT EXISTS transactions (
    transaction_id VARCHAR(100) PRIMARY KEY,
    reference_id VARCHAR(100) NOT NULL,
    idempotency_key VARCHAR(255) NOT NULL,
    
    -- Source account
    source_bank_code VARCHAR(8) NOT NULL,
    source_account_number VARCHAR(50) NOT NULL,
    
    -- Destination account
    dest_bank_code VARCHAR(8) NOT NULL,
    dest_account_number VARCHAR(50) NOT NULL,
    
    -- Transaction amounts
    amount DECIMAL(20, 2) NOT NULL,
    currency VARCHAR(3) NOT NULL DEFAULT 'IDR',
    fee DECIMAL(20, 2) NOT NULL DEFAULT 0,
    description TEXT,
    
    -- Transaction status
    status VARCHAR(20) NOT NULL DEFAULT 'PENDING',
    response_code VARCHAR(5) NOT NULL,
    response_msg VARCHAR(255) NOT NULL,
    
    -- Timestamps
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMP,
    
    -- Constraints
    CONSTRAINT uq_transactions_idempotency UNIQUE (idempotency_key),
    CONSTRAINT check_amount_positive CHECK (amount > 0),
    CONSTRAINT check_fee_positive CHECK (fee >= 0),
    CONSTRAINT check_status_valid CHECK (status IN ('PENDING', 'PROCESSING', 'COMPLETED', 'FAILED', 'EXPIRED'))
);

-- Indexes for transactions table
CREATE INDEX idx_transactions_reference_id ON transactions(reference_id);
CREATE INDEX idx_transactions_idempotency_key ON transactions(idempotency_key);
CREATE INDEX idx_transactions_status ON transactions(status);
CREATE INDEX idx_transactions_created_at ON transactions(created_at DESC);
CREATE INDEX idx_transactions_source_account ON transactions(source_bank_code, source_account_number);
CREATE INDEX idx_transactions_dest_account ON transactions(dest_bank_code, dest_account_number);

-- Composite indexes for common queries
CREATE INDEX idx_transactions_status_created ON transactions(status, created_at DESC);
CREATE INDEX idx_transactions_source_lookup ON transactions(source_bank_code, source_account_number, created_at DESC);
CREATE INDEX idx_transactions_dest_lookup ON transactions(dest_bank_code, dest_account_number, created_at DESC);

-- Comments for transactions table
COMMENT ON TABLE transactions IS 'Stores all BI-FAST transaction records';
COMMENT ON COLUMN transactions.transaction_id IS 'Unique transaction identifier (BIFAST-UUID format)';
COMMENT ON COLUMN transactions.reference_id IS 'Client reference ID for tracking';
COMMENT ON COLUMN transactions.idempotency_key IS 'Idempotency key to prevent duplicate transactions';
COMMENT ON COLUMN transactions.status IS 'Transaction status: PENDING, PROCESSING, COMPLETED, FAILED, EXPIRED';
COMMENT ON COLUMN transactions.response_code IS 'BI-FAST standard response code (00 = success)';
COMMENT ON COLUMN transactions.completed_at IS 'Timestamp when transaction was completed or failed';

-- ============================================================
-- TRIGGER: Update updated_at timestamp automatically
-- ============================================================
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Apply trigger to accounts table
CREATE TRIGGER trigger_accounts_updated_at
    BEFORE UPDATE ON accounts
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Apply trigger to transactions table
CREATE TRIGGER trigger_transactions_updated_at
    BEFORE UPDATE ON transactions
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- ============================================================
-- SUCCESS MESSAGE
-- ============================================================
DO $$
BEGIN
    RAISE NOTICE 'Migration 001_create_table.sql completed successfully';
    RAISE NOTICE 'Tables created: accounts, transactions';
    RAISE NOTICE 'Triggers created: auto update_at timestamp';
END $$;
