-- Create the schema
CREATE DATABASE IF NOT EXISTS transactions;

-- Use the schema
USE transactions;

-- Create the transactions table
CREATE TABLE IF NOT EXISTS transactions (
    id VARCHAR(36) PRIMARY KEY,
    amount DECIMAL(18, 2) NOT NULL,
    currency VARCHAR(10) NOT NULL,
    status ENUM('pending', 'completed', 'failed') NOT NULL,
    user_id VARCHAR(36) NOT NULL,
    merchant_id VARCHAR(36) NOT NULL,
    payment_method VARCHAR(50) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create the index for user_id
CREATE INDEX idx_user_id ON transactions(user_id);

-- Create a user for the database if not exist
CREATE USER IF NOT EXISTS 'transaction_user'@'%' IDENTIFIED BY 'secure_password';

-- Grant privileges to the user
GRANT ALL PRIVILEGES ON transactions.* TO 'transaction_user'@'%';

-- Apply changes
FLUSH PRIVILEGES;
