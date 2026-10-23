package main

import (
	"database/sql"
	"log"
	"os"
	"time"

	_ "github.com/lib/pq"
)

func main() {
	connStr := os.Getenv("DATABASE_URL")
	if connStr == "" {
		connStr = "postgres://shop:shop@postgres:5432/shop?sslmode=disable"
	}
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	log.Println("Starting worker...")
	for {
		res, err := db.Exec("UPDATE orders SET processed = TRUE WHERE processed = FALSE")
		if err != nil {
			log.Printf("Error processing orders: %v", err)
		} else {
			count, _ := res.RowsAffected()
			if count > 0 {
				log.Printf("Processed %d orders", count)
			}
		}
		time.Sleep(2 * time.Second)
	}
}