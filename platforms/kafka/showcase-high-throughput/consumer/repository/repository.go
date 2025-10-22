package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrProductNotFound   = errors.New("product not found")
	ErrInsufficientStock = errors.New("insufficient stock")
	ErrAlreadyProcessed  = errors.New("order already processed")
)

// Product represents inventory product
type Product struct {
	ProductID         string
	ProductName       string
	StockQuantity     int
	ReservedQuantity  int
	AvailableQuantity int
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// ProcessedOrder represents idempotency tracking
type ProcessedOrder struct {
	IdempotencyKey string
	OrderID        string
	UserID         string
	ProductID      string
	Quantity       int
	Status         string
	ProcessedAt    time.Time
}

// InventoryRepository handles database operations
type InventoryRepository struct {
	pool *pgxpool.Pool
}

// NewInventoryRepository creates new repository
func NewInventoryRepository(pool *pgxpool.Pool) *InventoryRepository {
	return &InventoryRepository{pool: pool}
}

// CheckIdempotency checks if order already processed
func (r *InventoryRepository) CheckIdempotency(ctx context.Context, idempotencyKey string) (*ProcessedOrder, error) {
	query := `
        SELECT idempotency_key, order_id, user_id, product_id, quantity, status, processed_at
        FROM processed_orders
        WHERE idempotency_key = $1
    `

	var po ProcessedOrder
	err := r.pool.QueryRow(ctx, query, idempotencyKey).Scan(
		&po.IdempotencyKey,
		&po.OrderID,
		&po.UserID,
		&po.ProductID,
		&po.Quantity,
		&po.Status,
		&po.ProcessedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("check idempotency failed: %w", err)
	}

	return &po, nil
}

// GetProduct retrieves product by ID
func (r *InventoryRepository) GetProduct(ctx context.Context, productID string) (*Product, error) {
	query := `
        SELECT product_id, product_name, stock_quantity, reserved_quantity, 
               available_quantity, created_at, updated_at
        FROM products
        WHERE product_id = $1
    `

	var p Product
	err := r.pool.QueryRow(ctx, query, productID).Scan(
		&p.ProductID,
		&p.ProductName,
		&p.StockQuantity,
		&p.ReservedQuantity,
		&p.AvailableQuantity,
		&p.CreatedAt,
		&p.UpdatedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrProductNotFound
	}

	if err != nil {
		return nil, fmt.Errorf("get product failed: %w", err)
	}

	return &p, nil
}

// ReserveInventory reserves stock (with transaction)
func (r *InventoryRepository) ReserveInventory(
	ctx context.Context,
	orderID, userID, productID string,
	quantity int,
	idempotencyKey string,
) error {
	// Begin transaction
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction failed: %w", err)
	}
	defer tx.Rollback(ctx)

	// Check idempotency within transaction
	var existingOrder string
	err = tx.QueryRow(
		ctx,
		"SELECT order_id FROM processed_orders WHERE idempotency_key = $1 FOR UPDATE",
		idempotencyKey,
	).Scan(&existingOrder)

	if err == nil {
		return ErrAlreadyProcessed
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("check idempotency in tx failed: %w", err)
	}

	// Get current stock (with row lock)
	var stockBefore, reserved int
	query := `
        SELECT stock_quantity, reserved_quantity
        FROM products
        WHERE product_id = $1
        FOR UPDATE
    `
	err = tx.QueryRow(ctx, query, productID).Scan(&stockBefore, &reserved)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrProductNotFound
	}
	if err != nil {
		return fmt.Errorf("get stock failed: %w", err)
	}

	// Check availability
	available := stockBefore - reserved
	if available < quantity {
		return ErrInsufficientStock
	}

	// Reserve inventory
	updateQuery := `
        UPDATE products
        SET reserved_quantity = reserved_quantity + $1
        WHERE product_id = $2
    `
	_, err = tx.Exec(ctx, updateQuery, quantity, productID)
	if err != nil {
		return fmt.Errorf("reserve inventory failed: %w", err)
	}

	// Record processed order
	insertOrder := `
        INSERT INTO processed_orders (idempotency_key, order_id, user_id, product_id, quantity, status)
        VALUES ($1, $2, $3, $4, $5, 'completed')
    `
	_, err = tx.Exec(ctx, insertOrder, idempotencyKey, orderID, userID, productID, quantity)
	if err != nil {
		return fmt.Errorf("insert processed order failed: %w", err)
	}

	// Log inventory change
	insertLog := `
        INSERT INTO inventory_logs (product_id, order_id, change_type, quantity_change, stock_before, stock_after)
        VALUES ($1, $2, 'reserve', $3, $4, $5)
    `
	_, err = tx.Exec(ctx, insertLog, productID, orderID, -quantity, stockBefore, stockBefore)
	if err != nil {
		return fmt.Errorf("insert inventory log failed: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction failed: %w", err)
	}

	return nil
}

// GetInventoryStats returns aggregate statistics
func (r *InventoryRepository) GetInventoryStats(ctx context.Context) (map[string]interface{}, error) {
	query := `
        SELECT 
            COUNT(*) as total_products,
            COALESCE(SUM(stock_quantity), 0) as total_stock,
            COALESCE(SUM(reserved_quantity), 0) as total_reserved,
            COALESCE(SUM(available_quantity), 0) as total_available
        FROM products
    `

	var stats struct {
		TotalProducts  int
		TotalStock     int
		TotalReserved  int
		TotalAvailable int
	}

	err := r.pool.QueryRow(ctx, query).Scan(
		&stats.TotalProducts,
		&stats.TotalStock,
		&stats.TotalReserved,
		&stats.TotalAvailable,
	)

	if err != nil {
		return nil, fmt.Errorf("get stats failed: %w", err)
	}

	return map[string]interface{}{
		"total_products":  stats.TotalProducts,
		"total_stock":     stats.TotalStock,
		"total_reserved":  stats.TotalReserved,
		"total_available": stats.TotalAvailable,
	}, nil
}
