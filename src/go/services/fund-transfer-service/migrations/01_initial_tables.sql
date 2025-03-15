-- migrations/01_initial_tables.sql
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Create transfers table
CREATE TABLE IF NOT EXISTS transfers (
    id UUID PRIMARY KEY,
    transfer_id VARCHAR(50) UNIQUE NOT NULL,
    source_account_id VARCHAR(50) NOT NULL,
    destination_account_number VARCHAR(50) NOT NULL,
    destination_bank_code VARCHAR(10) NOT NULL,
    destination_account_name VARCHAR(100) NOT NULL,
    amount DECIMAL(19, 2) NOT NULL,
    currency VARCHAR(3) NOT NULL,
    status VARCHAR(20) NOT NULL,
    reference_number VARCHAR(50) NOT NULL,
    description TEXT,
    fee DECIMAL(19, 2) DEFAULT 0,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    completed_at TIMESTAMP NULL,
    transaction_ids JSONB NULL,
    ledger_journal_id VARCHAR(50) NULL
);

-- Create bulk_transfers table
CREATE TABLE IF NOT EXISTS bulk_transfers (
    id UUID PRIMARY KEY,
    bulk_transfer_id VARCHAR(50) UNIQUE NOT NULL,
    source_account_id VARCHAR(50) NOT NULL,
    total_amount DECIMAL(19, 2) NOT NULL,
    transfer_count INTEGER NOT NULL,
    currency VARCHAR(3) NOT NULL,
    status VARCHAR(20) NOT NULL,
    batch_reference VARCHAR(100) NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    completed_at TIMESTAMP NULL
);

-- Create bulk_transfer_items table
CREATE TABLE IF NOT EXISTS bulk_transfer_items (
    id UUID PRIMARY KEY,
    bulk_transfer_id VARCHAR(50) NOT NULL,
    transfer_id VARCHAR(50) NOT NULL,
    status VARCHAR(20) NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    FOREIGN KEY (bulk_transfer_id) REFERENCES bulk_transfers(bulk_transfer_id)
);

-- Create indexes
CREATE INDEX IF NOT EXISTS idx_transfers_transfer_id ON transfers(transfer_id);
CREATE INDEX IF NOT EXISTS idx_transfers_source_account_id ON transfers(source_account_id);
CREATE INDEX IF NOT EXISTS idx_transfers_status ON transfers(status);
CREATE INDEX IF NOT EXISTS idx_bulk_transfers_bulk_transfer_id ON bulk_transfers(bulk_transfer_id);
CREATE INDEX IF NOT EXISTS idx_bulk_transfers_source_account_id ON bulk_transfers(source_account_id);
CREATE INDEX IF NOT EXISTS idx_bulk_transfer_items_bulk_transfer_id ON bulk_transfer_items(bulk_transfer_id);
CREATE INDEX IF NOT EXISTS idx_bulk_transfer_items_transfer_id ON bulk_transfer_items(transfer_id);
