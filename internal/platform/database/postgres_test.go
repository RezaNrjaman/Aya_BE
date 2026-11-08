package database

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/joho/godotenv"
)

func TestLiveSupabaseConnection(t *testing.T) {
	_ = godotenv.Load("../../../.env")
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" || dbURL == "postgresql://postgres:postgres@localhost:5432/araya_hub?sslmode=disable" {
		t.Skip("DATABASE_URL not configured for live test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db, err := Connect(ctx, dbURL)
	if err != nil {
		t.Skipf("Supabase network unreachable (offline or sandboxed): %v", err)
		return
	}
	defer db.Close()

	if err := db.Ping(ctx); err != nil {
		t.Fatalf("Ping failed: %v", err)
	}
}

func TestVerifyTablesExist(t *testing.T) {
	_ = godotenv.Load("../../../.env")
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("DATABASE_URL not configured")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db, err := Connect(ctx, dbURL)
	if err != nil {
		t.Skipf("Supabase network unreachable (offline or sandboxed): %v", err)
		return
	}
	defer db.Close()

	var uCount, rCount, buCount, pCount int
	_ = db.Pool.QueryRow(ctx, "SELECT count(*) FROM users").Scan(&uCount)
	_ = db.Pool.QueryRow(ctx, "SELECT count(*) FROM roles").Scan(&rCount)
	_ = db.Pool.QueryRow(ctx, "SELECT count(*) FROM business_units").Scan(&buCount)
	_ = db.Pool.QueryRow(ctx, "SELECT count(*) FROM products").Scan(&pCount)

	t.Logf("Data Counts -> Users: %d, Roles: %d, Business Units: %d, Products: %d", uCount, rCount, buCount, pCount)
}
