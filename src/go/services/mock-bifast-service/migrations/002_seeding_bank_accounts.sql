-- migrations/002_seeding_bank_accounts.sql
-- Seed mock bank accounts for testing

-- ============================================================
-- SEED: Bank Accounts
-- Purpose: Create test accounts for various Indonesian banks
-- ============================================================

-- BRI Accounts
INSERT INTO accounts (
    id, bank_code, account_number, account_name, account_type,
    balance, currency, status, is_active, is_blocked, created_at, updated_at
) VALUES
(
    uuid_generate_v4(),
    'BRINIDJA',  -- Bank BRI
    '1234567890',
    'John Doe',
    'savings',
    50000000.00,  -- IDR 50 juta
    'IDR',
    'active',
    true,
    false,
    NOW(),
    NOW()
),
(
    uuid_generate_v4(),
    'BRINIDJA',
    '1111222233',
    'Andi Wijaya',
    'checking',
    120000000.00,  -- IDR 120 juta
    'IDR',
    'active',
    true,
    false,
    NOW(),
    NOW()
);

-- Mandiri Accounts
INSERT INTO accounts (
    id, bank_code, account_number, account_name, account_type,
    balance, currency, status, is_active, is_blocked, created_at, updated_at
) VALUES
(
    uuid_generate_v4(),
    'BMRIIDJA',  -- Bank Mandiri
    '0987654321',
    'Jane Smith',
    'savings',
    75000000.00,  -- IDR 75 juta
    'IDR',
    'active',
    true,
    false,
    NOW(),
    NOW()
),
(
    uuid_generate_v4(),
    'BMRIIDJA',
    '5555666677',
    'Budi Santoso',
    'current',
    250000000.00,  -- IDR 250 juta
    'IDR',
    'active',
    true,
    false,
    NOW(),
    NOW()
);

-- BCA Accounts
INSERT INTO accounts (
    id, bank_code, account_number, account_name, account_type,
    balance, currency, status, is_active, is_blocked, created_at, updated_at
) VALUES
(
    uuid_generate_v4(),
    'CENAIDJA',  -- Bank BCA
    '1122334455',
    'Alice Johnson',
    'savings',
    100000000.00,  -- IDR 100 juta
    'IDR',
    'active',
    true,
    false,
    NOW(),
    NOW()
),
(
    uuid_generate_v4(),
    'CENAIDJA',
    '9988776655',
    'PT Sukses Jaya',
    'current',
    500000000.00,  -- IDR 500 juta (corporate)
    'IDR',
    'active',
    true,
    false,
    NOW(),
    NOW()
);

-- BNI Accounts
INSERT INTO accounts (
    id, bank_code, account_number, account_name, account_type,
    balance, currency, status, is_active, is_blocked, created_at, updated_at
) VALUES
(
    uuid_generate_v4(),
    'BNINIDJA',  -- Bank BNI
    '6677889900',
    'Bob Brown',
    'savings',
    30000000.00,  -- IDR 30 juta
    'IDR',
    'active',
    true,
    false,
    NOW(),
    NOW()
),
(
    uuid_generate_v4(),
    'BNINIDJA',
    '4444333322',
    'Charlie Davis',
    'checking',
    85000000.00,  -- IDR 85 juta
    'IDR',
    'active',
    true,
    false,
    NOW(),
    NOW()
);

-- Bank Sinarmas Accounts
INSERT INTO accounts (
    id, bank_code, account_number, account_name, account_type,
    balance, currency, status, is_active, is_blocked, created_at, updated_at
) VALUES
(
    uuid_generate_v4(),
    'SNIAIDJA',  -- Bank Sinarmas
    '7777888899',
    'Rezky Aulia Pratama',
    'savings',
    150000000.00,  -- IDR 150 juta
    'IDR',
    'active',
    true,
    false,
    NOW(),
    NOW()
),
(
    uuid_generate_v4(),
    'SNIAIDJA',
    '2222111100',
    'PT Teknologi Indonesia',
    'current',
    1000000000.00,  -- IDR 1 miliar (corporate)
    'IDR',
    'active',
    true,
    false,
    NOW(),
    NOW()
);

-- BankDinar Accounts (smaller bank)
INSERT INTO accounts (
    id, bank_code, account_number, account_name, account_type,
    balance, currency, status, is_active, is_blocked, created_at, updated_at
) VALUES
(
    uuid_generate_v4(),
    'BDINIDJA',  -- Bank Dinar
    '9876543210',
    'David Lee',
    'savings',
    25000000.00,  -- IDR 25 juta
    'IDR',
    'active',
    true,
    false,
    NOW(),
    NOW()
);

-- Test accounts with special statuses
INSERT INTO accounts (
    id, bank_code, account_number, account_name, account_type,
    balance, currency, status, is_active, is_blocked, created_at, updated_at
) VALUES
-- Blocked account for testing
(
    uuid_generate_v4(),
    'BRINIDJA',
    '9999888877',
    'Blocked Account Test',
    'savings',
    10000000.00,
    'IDR',
    'blocked',
    false,
    true,
    NOW(),
    NOW()
),
-- Low balance account for testing insufficient funds
(
    uuid_generate_v4(),
    'BMRIIDJA',
    '1111000099',
    'Low Balance Test',
    'savings',
    1000.00,  -- IDR 1000 only
    'IDR',
    'active',
    true,
    false,
    NOW(),
    NOW()
),
-- Closed account for testing
(
    uuid_generate_v4(),
    'CENAIDJA',
    '0000111122',
    'Closed Account Test',
    'savings',
    0.00,
    'IDR',
    'closed',
    false,
    false,
    NOW(),
    NOW()
);

-- ============================================================
-- SAMPLE TRANSACTION DATA (optional)
-- ============================================================
INSERT INTO transactions (
    transaction_id, reference_id, idempotency_key,
    source_bank_code, source_account_number,
    dest_bank_code, dest_account_number,
    amount, currency, fee, description,
    status, response_code, response_msg,
    created_at, updated_at, completed_at
) VALUES 
-- Completed transaction
(
    'BIFAST-' || uuid_generate_v4()::text,
    'REF-' || uuid_generate_v4()::text,
    'IDEM-' || uuid_generate_v4()::text,
    'CENAIDJA',
    '1122334455',
    'BDINIDJA',
    '9876543210',
    5000000.00,  -- IDR 5 juta
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
-- Another completed transaction
(
    'BIFAST-' || uuid_generate_v4()::text,
    'REF-' || uuid_generate_v4()::text,
    'IDEM-' || uuid_generate_v4()::text,
    'BMRIIDJA',
    '5555666677',
    'SNIAIDJA',
    '7777888899',
    10000000.00,  -- IDR 10 juta
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
-- Pending transaction
(
    'BIFAST-' || uuid_generate_v4()::text,
    'REF-' || uuid_generate_v4()::text,
    'IDEM-' || uuid_generate_v4()::text,
    'SNIAIDJA',
    '7777888899',
    'CENAIDJA',
    '1122334455',
    2500000.00,  -- IDR 2.5 juta
    'IDR',
    2500.00,
    'Transfer untuk testing',
    'PENDING',
    '00',
    'Transaction initiated',
    NOW() - INTERVAL '10 minutes',
    NOW() - INTERVAL '10 minutes',
    NULL
),
-- Failed transaction (insufficient funds)
(
    'BIFAST-' || uuid_generate_v4()::text,
    'REF-' || uuid_generate_v4()::text,
    'IDEM-' || uuid_generate_v4()::text,
    'BMRIIDJA',
    '1111000099',  -- Low balance account
    'BRINIDJA',
    '1234567890',
    50000000.00,  -- IDR 50 juta (too much)
    'IDR',
    25000.00,
    'Transfer gagal - saldo tidak cukup',
    'FAILED',
    '51',
    'Insufficient funds',
    NOW() - INTERVAL '1 hour',
    NOW() - INTERVAL '1 hour',
    NOW() - INTERVAL '59 minutes'
);

-- ============================================================
-- VERIFICATION QUERIES
-- ============================================================
DO $$
DECLARE
    account_count INT;
    transaction_count INT;
    total_balance DECIMAL;
BEGIN
    SELECT COUNT(*) INTO account_count FROM accounts;
    SELECT COUNT(*) INTO transaction_count FROM transactions;
    SELECT SUM(balance) INTO total_balance FROM accounts WHERE is_active = true;
    
    RAISE NOTICE '========================================';
    RAISE NOTICE 'Migration 002_seeding_bank_accounts.sql completed successfully';
    RAISE NOTICE '========================================';
    RAISE NOTICE 'Total accounts created: %', account_count;
    RAISE NOTICE 'Total transactions created: %', transaction_count;
    RAISE NOTICE 'Total active balance: IDR %', total_balance;
    RAISE NOTICE '========================================';
END $$;
