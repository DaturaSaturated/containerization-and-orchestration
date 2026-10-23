package main

import (
	"fmt"
	"net/http"
	"strconv"
	"time"
)

var allocations [][]byte

func health(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok\n"))
}

func eat(w http.ResponseWriter, r *http.Request) {
	mb, err := strconv.Atoi(r.URL.Query().Get("mb"))
	if err != nil || mb < 1 {
		http.Error(w, "mb must be a positive integer", http.StatusBadRequest)
		return
	}

	data := make([]byte, mb*1024*1024)

	// Записываем в каждый блок, чтобы память действительно была затронута.
	for i := range data {
		data[i] = 1
	}

	allocations = append(allocations, data)

	fmt.Fprintf(w, "allocated %d MB\n", mb)
}

func burn(w http.ResponseWriter, r *http.Request) {
	for {
		// Намеренно занимаем CPU.
	}
}

func main() {
	http.HandleFunc("/health", health)
	http.HandleFunc("/eat", eat)
	http.HandleFunc("/burn", burn)

	server := &http.Server{
		Addr:              ":8080",
		ReadHeaderTimeout: 5 * time.Second,
	}

	fmt.Println("api listening on :8080")

	if err := server.ListenAndServe(); err != nil {
		panic(err)
	}
}