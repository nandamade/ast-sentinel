package main

import (
	"database/sql"
	"fmt"
	"os"
	"sync"
)

type Account struct {
	mu sync.Mutex
}

func (a *Account) ProcessTransaction(db *sql.DB, userID string) {
	// SEC-001: SQL Injection via Sprintf
	query := fmt.Sprintf("SELECT * FROM users WHERE id = '%s'", userID)
	db.Query(query)

	// CONC-001: Mutex Deadlock
	a.mu.Lock()
	fmt.Println("Transaction step 1")
	a.mu.Lock() // Double lock!
	a.mu.Unlock()

	// ERR-001: Explicit Unchecked Error
	_ = os.Remove("/tmp/cache.txt")
}
