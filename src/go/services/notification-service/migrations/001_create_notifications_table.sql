-- migrations/001_create_notifications_table.sql
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Notifications table
CREATE TABLE IF NOT EXISTS notifications (
    id UUID PRIMARY KEY,  -- UUID v7 generated in code
    notification_id VARCHAR(50) UNIQUE NOT NULL,
    recipient_id VARCHAR(50) NOT NULL,
    type VARCHAR(10) NOT NULL,  -- EMAIL, SMS, PUSH
    title VARCHAR(255) NOT NULL,
    message TEXT NOT NULL,
    status VARCHAR(20) NOT NULL,  -- PENDING, SENT, FAILED
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    sent_at TIMESTAMP NULL
);

-- Indexes for optimized queries
CREATE INDEX idx_notifications_recipient_id ON notifications(recipient_id);
CREATE INDEX idx_notifications_status ON notifications(status);
CREATE INDEX idx_notifications_type ON notifications(type);
CREATE INDEX idx_notifications_created_at ON notifications(created_at);
