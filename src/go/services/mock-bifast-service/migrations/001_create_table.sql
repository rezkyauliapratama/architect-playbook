-- migrations/001_create_tables.sql
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Bank accounts table for inquiry operations
CREATE TABLE IF NOT EXISTS bank_accounts (
    id UUID PRIMARY KEY,
    account_number VARCHAR(50) UNIQUE NOT NULL,
    account_name VARCHAR(100) NOT NULL,
    bank_code VARCHAR(10) NOT NULL,
    bank_name VARCHAR(100) NOT NULL,
    proxy_type VARCHAR(10) NULL, -- EMAIL or PHONE
    proxy_value VARCHAR(100) NULL, -- Email address or phone number
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Create optimized indexes for account lookup
CREATE INDEX idx_bank_accounts_account_number ON bank_accounts(account_number);
CREATE INDEX idx_bank_accounts_proxy ON bank_accounts(proxy_type, proxy_value) WHERE proxy_type IS NOT NULL;
CREATE INDEX idx_bank_accounts_bank_code ON bank_accounts(bank_code);

-- Transaction tracking table
CREATE TABLE IF NOT EXISTS bifast_transactions (
    id UUID PRIMARY KEY,
    transaction_id VARCHAR(50) UNIQUE NOT NULL,
    source_account_number VARCHAR(50) NOT NULL,
    source_account_name VARCHAR(100) NOT NULL,
    source_bank_code VARCHAR(10) NOT NULL,
    destination_account_number VARCHAR(50) NOT NULL,
    destination_account_name VARCHAR(100) NOT NULL,
    destination_bank_code VARCHAR(10) NOT NULL,
    amount DECIMAL(19, 2) NOT NULL,
    fee DECIMAL(19, 2) NOT NULL DEFAULT 2500,
    currency VARCHAR(3) NOT NULL DEFAULT 'IDR',
    status VARCHAR(20) NOT NULL, -- PENDING, COMPLETED, FAILED
    reference_id VARCHAR(50) NOT NULL,
    description TEXT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    completed_at TIMESTAMP NULL
);

-- Optimized indexes for transaction queries
CREATE INDEX idx_bifast_transactions_transaction_id ON bifast_transactions(transaction_id);
CREATE INDEX idx_bifast_transactions_source_account ON bifast_transactions(source_account_number);
CREATE INDEX idx_bifast_transactions_destination_account ON bifast_transactions(destination_account_number);
CREATE INDEX idx_bifast_transactions_status ON bifast_transactions(status);
CREATE INDEX idx_bifast_transactions_created_at ON bifast_transactions(created_at);
CREATE INDEX idx_bifast_transactions_reference_id ON bifast_transactions(reference_id);
