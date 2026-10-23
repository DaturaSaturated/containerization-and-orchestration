package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	_ "github.com/lib/pq"
)

var db *sql.DB

func main() {
	connStr := os.Getenv("DATABASE_URL")
	if connStr == "" {
		connStr = "postgres://shop:shop@postgres:5432/shop?sslmode=disable"
	}
	var err error
	db, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS orders (
		id SERIAL PRIMARY KEY, data TEXT, processed BOOLEAN DEFAULT FALSE
	)`)
	if err != nil {
		log.Printf("Warning: could not create table: %v", err)
	}

	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/order", orderHandler)
	http.HandleFunc("/orders", ordersHandler)

	log.Println("Starting api server on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	if os.Getenv("HEALTH_FAIL") == "true" {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, "ok")
}

func orderHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var body map[string]string
	json.NewDecoder(r.Body).Decode(&body)
	_, err := db.Exec("INSERT INTO orders (data) VALUES ($1)", body["item"])
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func ordersHandler(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT id, data, processed FROM orders")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var orders []map[string]interface{}
	for rows.Next() {
		var id int
		var data string
		var processed bool
		rows.Scan(&id, &data, &processed)
		orders = append(orders, map[string]interface{}{"id": id, "data": data, "processed": processed})
	}
	json.NewEncoder(w).Encode(orders)
}