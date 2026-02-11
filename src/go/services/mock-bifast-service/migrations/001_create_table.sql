-- migrations/001_create_transactions.sql
-- Create transactions table for Mock BI-FAST Service

-- Enable UUID extension (if not already enabled)
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Create transactions table
CREATE TABLE IF NOT EXISTS transactions (
    transaction_id VARCHAR(100) PRIMARY KEY,
    reference_id VARCHAR(100) NOT NULL,
    idempotency_key VARCHAR(255) NOT NULL,
    
    source_bank_code VARCHAR(8) NOT NULL,
    source_account_number VARCHAR(50) NOT NULL,
    dest_bank_code VARCHAR(8) NOT NULL,
    dest_account_number VARCHAR(50) NOT NULL,
    
    amount DECIMAL(20, 2) NOT NULL,
    currency VARCHAR(3) NOT NULL DEFAULT 'IDR',
    fee DECIMAL(20, 2) NOT NULL DEFAULT 0,
    description TEXT,
    
    status VARCHAR(20) NOT NULL DEFAULT 'PENDING',
    response_code VARCHAR(5) NOT NULL,
    response_msg VARCHAR(255) NOT NULL,
    
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMP,
    
    CONSTRAINT check_amount_positive CHECK (amount > 0),
    CONSTRAINT check_fee_positive CHECK (fee >= 0),
    CONSTRAINT check_status_valid CHECK (status IN ('PENDING', 'PROCESSING', 'COMPLETED', 'FAILED', 'EXPIRED'))
);

-- Create indexes for performance
CREATE INDEX idx_transactions_reference_id ON transactions(reference_id);
CREATE INDEX idx_transactions_idempotency_key ON transactions(idempotency_key);
CREATE INDEX idx_transactions_status ON transactions(status);
CREATE INDEX idx_transactions_created_at ON transactions(created_at DESC);
CREATE INDEX idx_transactions_source_account ON transactions(source_bank_code, source_account_number);
CREATE INDEX idx_transactions_dest_account ON transactions(dest_bank_code, dest_account_number);

-- Create composite index for common queries
CREATE INDEX idx_transactions_status_created ON transactions(status, created_at DESC);

-- Add comments
COMMENT ON TABLE transactions IS 'Stores all BI-FAST transaction records';
COMMENT ON COLUMN transactions.transaction_id IS 'Unique transaction identifier (BIFAST-UUID format)';
COMMENT ON COLUMN transactions.reference_id IS 'Client reference ID for tracking';
COMMENT ON COLUMN transactions.idempotency_key IS 'Idempotency key to prevent duplicate transactions';
COMMENT ON COLUMN transactions.status IS 'Transaction status: PENDING, PROCESSING, COMPLETED, FAILED, EXPIRED';
COMMENT ON COLUMN transactions.response_code IS 'BI-FAST standard response code (00 = success)';
COMMENT ON COLUMN transactions.completed_at IS 'Timestamp when transaction was completed or failed';

-- Sample mock data (optional, for testing)
INSERT INTO transactions (
    transaction_id, reference_id, idempotency_key,
    source_bank_code, source_account_number,
    dest_bank_code, dest_account_number,
    amount, currency, fee, description,
    status, response_code, response_msg,
    created_at, updated_at, completed_at
) VALUES 
(
    'BIFAST-' || uuid_generate_v4()::text,
    'REF-' || uuid_generate_v4()::text,
    'IDEM-' || uuid_generate_v4()::text,
    'CENAIDJA',
    '1234567890',
    'BDINIDJA',
    '9876543210',
    5000000.00,
    'IDR',
    2500.00,
    'Transfer untuk pembayaran invoice #INV-001',
    'COMPLETED',
    '00',
    'Transaction successful',
    NOW() - INTERVAL '5 hours',
    NOW() - INTERVAL '5 hours',
    NOW() - INTERVAL '4 hours 59 minutes'
),
(
    'BIFAST-' || uuid_generate_v4()::text,
    'REF-' || uuid_generate_v4()::text,
    'IDEM-' || uuid_generate_v4()::text,
    'BMRIIDJA',
    '1111222233',
    'SNIAIDJA',
    '5555666677',
    10000000.00,
    'IDR',
    10000.00,
    'Transfer gaji karyawan bulan Februari 2026',
    'COMPLETED',
    '00',
    'Transaction successful',
    NOW() - INTERVAL '2 hours',
    NOW() - INTERVAL '2 hours',
    NOW() - INTERVAL '1 hour 58 minutes'
),
(
    'BIFAST-' || uuid_generate_v4()::text,
    'REF-' || uuid_generate_v4()::text,
    'IDEM-' || uuid_generate_v4()::text,
    'SNIAIDJA',
    '5555666677',
    'CENAIDJA',
    '1234567890',
    2500000.00,
    'IDR',
    2500.00,
    'Transfer untuk testing',
    'PENDING',
    '00',
    'Transaction initiated',
    NOW() - INTERVAL '10 minutes',
    NOW() - INTERVAL '10 minutes',
    NULL
);
