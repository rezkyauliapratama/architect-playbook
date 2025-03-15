package main

import (
	"context"
	"database/sql"
	"log"

	"github.com/ClickHouse/clickhouse-go/v2"
	_ "github.com/go-sql-driver/mysql"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
)

type Transaction struct {
	ID            string  `json:"id"`
	Amount        float64 `json:"amount"`
	Currency      string  `json:"currency"`
	Status        string  `json:"status"`
	UserID        string  `json:"user_id"`
	MerchantID    string  `json:"merchant_id"`
	PaymentMethod string  `json:"payment_method"`
}

type UserTotal struct {
	UserID string  `json:"user_id"`
	Total  float64 `json:"total_amount"`
	Count  uint32  `json:"transaction_count"`
}

type TransactionHandler struct {
	mysqlDB      *sql.DB
	clickhouseDB clickhouse.Conn
}

func initDatabases() (*sql.DB, clickhouse.Conn) {
	mysqlDB, err := sql.Open("mysql", "transaction_user:secure_password@tcp(localhost:3306)/transactions")
	if err != nil {
		log.Fatal(err)
	}

	clickhouseDB, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{"localhost:9000"},
		Auth: clickhouse.Auth{
			Database: "transactions",
			Username: "transaction_user",
			Password: "secure_password",
		},
		Settings: clickhouse.Settings{
			"max_execution_time":    60,
			"max_threads":           8,
			"max_insert_block_size": 1048576,
		},
		Compression: &clickhouse.Compression{
			Method: clickhouse.CompressionLZ4,
		},
		MaxOpenConns: 5,
		MaxIdleConns: 5,
	})
	if err != nil {
		log.Fatal(err)
	}

	return mysqlDB, clickhouseDB
}

func (h *TransactionHandler) CreateTransaction(c *fiber.Ctx) error {
	tx := new(Transaction)
	if err := c.BodyParser(tx); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Cannot parse JSON"})
	}

	// Insert into MySQL
	_, err := h.mysqlDB.Exec(`
        INSERT INTO transactions 
        (id, amount, currency, status, user_id, merchant_id, payment_method) 
        VALUES (?, ?, ?, ?, ?, ?, ?)`,
		tx.ID, tx.Amount, tx.Currency, tx.Status, tx.UserID, tx.MerchantID, tx.PaymentMethod)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "MySQL insert failed"})
	}

	// Insert into ClickHouse
	err = h.clickhouseDB.Exec(context.Background(), `
        INSERT INTO transactions 
        (id, amount, currency, status, user_id, merchant_id, payment_method)
        VALUES (?, ?, ?, ?, ?, ?, ?)`,
		tx.ID, tx.Amount, tx.Currency, tx.Status, tx.UserID, tx.MerchantID, tx.PaymentMethod)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "ClickHouse insert failed"})
	}

	return c.Status(fiber.StatusCreated).JSON(tx)
}

// MySQL handler for user totals
func (h *TransactionHandler) GetMySQLUserTotals(c *fiber.Ctx) error {
	var totals []UserTotal
	rows, err := h.mysqlDB.Query(`
        SELECT user_id, 
               SUM(amount) as total_amount,
               COUNT(*) as transaction_count
        FROM transactions 
        GROUP BY user_id`)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "MySQL query failed"})
	}
	defer rows.Close()

	for rows.Next() {
		var total UserTotal
		if err := rows.Scan(&total.UserID, &total.Total, &total.Count); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "MySQL scan failed"})
		}
		totals = append(totals, total)
	}

	return c.JSON(fiber.Map{"source": "mysql", "totals": totals})
}

// ClickHouse handler for user totals
func (h *TransactionHandler) GetClickHouseUserTotals(c *fiber.Ctx) error {
	var totals []UserTotal
	rows, err := h.clickhouseDB.Query(context.Background(), `
        SELECT 
            user_id,
            toFloat64(total_amount),
            toUInt32(transaction_count)
        FROM user_totals_mv
    `)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "ClickHouse query failed"})
	}
	defer rows.Close()

	for rows.Next() {
		var total UserTotal
		if err := rows.Scan(&total.UserID, &total.Total, &total.Count); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "ClickHouse scan failed: " + err.Error(),
			})
		}
		totals = append(totals, total)
	}

	return c.JSON(fiber.Map{"source": "clickhouse", "totals": totals})
}

func setupRoutes(app *fiber.App, handler *TransactionHandler) {
	app.Post("/transactions", handler.CreateTransaction)
	app.Get("/totals/mysql", handler.GetMySQLUserTotals)
	app.Get("/totals/clickhouse", handler.GetClickHouseUserTotals)
}

func main() {
	mysqlDB, clickhouseDB := initDatabases()
	defer mysqlDB.Close()
	defer clickhouseDB.Close()

	app := fiber.New()

	// Add logger middleware
	app.Use(logger.New(logger.Config{
		Format:     "[${time}] ${status} - ${latency} ${method} ${path}\n",
		TimeFormat: "2006-01-02 15:04:05",
		TimeZone:   "Local",
	}))

	handler := &TransactionHandler{
		mysqlDB:      mysqlDB,
		clickhouseDB: clickhouseDB,
	}

	setupRoutes(app, handler)
	log.Fatal(app.Listen(":3000"))
}
