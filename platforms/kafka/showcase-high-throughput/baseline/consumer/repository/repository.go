package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type InventoryRepository struct {
	pool *pgxpool.Pool
}

func NewInventoryRepository(pool *pgxpool.Pool) *InventoryRepository {
	return &InventoryRepository{pool: pool}
}

// ❌ BASELINE: NO idempotency check method
// This function doesn't exist in baseline

// ReserveInventory - ❌ VULNERABLE: No idempotency check
func (r *InventoryRepository) ReserveInventory(
	ctx context.Context,
	orderID, userID, productID string,
	quantity int,
	idempotencyKey string,
) error {
	// ❌ NO TRANSACTION - each operation separate
	// ❌ NO IDEMPOTENCY CHECK - process every message

	// Step 1: Read current stock
	var currentStock, currentReserved int
	err := r.pool.QueryRow(ctx, `
		SELECT stock_quantity, reserved_quantity
		FROM products
		WHERE product_id = $1
	`, productID).Scan(&currentStock, &currentReserved)

	if err != nil {
		return fmt.Errorf("product not found: %w", err)
	}

	// Check availability
	available := currentStock - currentReserved
	if available < quantity {
		return fmt.Errorf("insufficient stock: available=%d, requested=%d", available, quantity)
	}

	// Step 2: Update reservation
	newReserved := currentReserved + quantity
	_, err = r.pool.Exec(ctx, `
		UPDATE products
		SET reserved_quantity = $1
		WHERE product_id = $2
	`, newReserved, productID)

	if err != nil {
		return fmt.Errorf("failed to update reservation: %w", err)
	}

	// Step 3: Record order (may fail with duplicate key, but reservation already done!)
	_, err = r.pool.Exec(ctx, `
		INSERT INTO processed_orders (
			idempotency_key, order_id, user_id, product_id, quantity, processed_at
		) VALUES ($1, $2, $3, $4, $5, NOW())
	`, idempotencyKey, orderID, userID, productID, quantity)

	// ❌ CRITICAL BUG: If INSERT fails (duplicate key), reservation already committed!
	if err != nil {
		// Just log, don't rollback (because no transaction)
		return nil // Swallow error
	}

	return nil
}
