-- Create database
-- CREATE DATABASE inventory_db;

-- Connect to database
\c inventory_db;

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- =====================================================
-- PRODUCTS TABLE
-- =====================================================
CREATE TABLE IF NOT EXISTS products (
    product_id VARCHAR(50) PRIMARY KEY,
    product_name VARCHAR(255) NOT NULL,
    stock_quantity INTEGER NOT NULL DEFAULT 0,
    reserved_quantity INTEGER NOT NULL DEFAULT 0,
    available_quantity INTEGER GENERATED ALWAYS AS (stock_quantity - reserved_quantity) STORED,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    
    CONSTRAINT positive_stock CHECK (stock_quantity >= 0),
    CONSTRAINT positive_reserved CHECK (reserved_quantity >= 0),
    CONSTRAINT valid_reserved CHECK (reserved_quantity <= stock_quantity)
);

CREATE INDEX idx_products_available ON products(available_quantity) WHERE available_quantity > 0;

-- =====================================================
-- PROCESSED_ORDERS TABLE
-- =====================================================
CREATE TABLE IF NOT EXISTS processed_orders (
    idempotency_key VARCHAR(255) PRIMARY KEY,
    order_id VARCHAR(100) NOT NULL,
    user_id VARCHAR(100) NOT NULL,
    product_id VARCHAR(50) NOT NULL REFERENCES products(product_id),
    quantity INTEGER NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'completed',
    processed_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    
    CONSTRAINT positive_quantity CHECK (quantity > 0)
);

CREATE INDEX idx_processed_orders_order_id ON processed_orders(order_id);
CREATE INDEX idx_processed_orders_processed_at ON processed_orders(processed_at DESC);

-- =====================================================
-- INVENTORY_LOGS TABLE
-- =====================================================
CREATE TABLE IF NOT EXISTS inventory_logs (
    log_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    product_id VARCHAR(50) NOT NULL REFERENCES products(product_id),
    order_id VARCHAR(100),
    change_type VARCHAR(50) NOT NULL,
    quantity_change INTEGER NOT NULL,
    stock_before INTEGER NOT NULL,
    stock_after INTEGER NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_inventory_logs_product_id ON inventory_logs(product_id);
CREATE INDEX idx_inventory_logs_created_at ON inventory_logs(created_at DESC);

-- =====================================================
-- AUTO-UPDATE TRIGGER
-- =====================================================
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_products_updated_at
    BEFORE UPDATE ON products
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- =====================================================
-- SEED DATA
-- =====================================================
INSERT INTO products (product_id, product_name, stock_quantity) VALUES
    ('prd_laptop_001', 'ASUS ROG Gaming Laptop', 100),
    ('prd_mouse_002', 'Logitech G Pro Wireless', 500),
    ('prd_keyboard_003', 'Keychron K8 Mechanical', 200),
    ('prd_monitor_004', 'LG 4K UltraGear Monitor', 50)
ON CONFLICT (product_id) DO NOTHING;

-- =====================================================
-- VIEWS
-- =====================================================
CREATE OR REPLACE VIEW inventory_status AS
SELECT 
    product_id,
    product_name,
    stock_quantity,
    reserved_quantity,
    available_quantity,
    ROUND((available_quantity::DECIMAL / NULLIF(stock_quantity, 0)) * 100, 2) AS availability_percent,
    updated_at
FROM products
ORDER BY product_name;
