-- Ensure UUID extension is available
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Seed bank_accounts table
INSERT INTO bank_accounts (
    id, account_number, account_name, bank_code, bank_name, proxy_type, proxy_value, created_at
)
VALUES
    (uuid_generate_v4(), '1234567890', 'John Doe', 'BBRI', 'Bank BRI', NULL, NULL, NOW()),
    (uuid_generate_v4(), '0987654321', 'Jane Smith', 'BMRI', 'Bank Mandiri', NULL, NULL, NOW()),
    (uuid_generate_v4(), '1122334455', 'Alice Johnson', 'BCA', 'Bank BCA', 'EMAIL', 'alice@example.com', NOW()),
    (uuid_generate_v4(), '6677889900', 'Bob Brown', 'BNI', 'Bank BNI', 'PHONE', '081234567890', NOW());
