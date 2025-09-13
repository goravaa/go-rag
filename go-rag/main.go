package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"go-rag/internal/ent"
	"go-rag/internal/ent/user" // Import the user package for predicates

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

func main() {
	// Load .env file
	err := godotenv.Load()
	if err != nil {
		// This is not a fatal error if the env var is set in the system
		log.Println("Warning: .env file not found, relying on system environment variables.")
	}

	client := initEntClient()
	defer client.Close()

	ctx := context.Background()

	// Try inserting a dummy user
	// If it already exists, this will produce an error, which is expected on subsequent runs.
	u, err := client.User.
		Create().
		SetEmail("test@example.com").
		SetPasswordHash("hashedpassword").
		Save(ctx)

	if err != nil {
		// We'll log the error but continue, as the user likely already exists.
		fmt.Println("\nCould not create user (it probably already exists):", err)
	} else {
		fmt.Println("Created user:", u)
	}

	// Fetch the user back
	fmt.Println("\nFetching user 'test@example.com'...")
	fetchedUser, err := client.User.
		Query().
		Where(user.EmailEQ("test@example.com")).
		Only(ctx)
	if err != nil {
		log.Fatalf("failed querying user: %v", err)
	}
	fmt.Println("Successfully fetched user:", fetchedUser)
}

// initEntClient connects to the database and returns an ent client.
func initEntClient() *ent.Client {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL environment variable is not set. Please create a .env file or set it in your environment.")
	}

	client, err := ent.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("failed opening connection to postgres: %v", err)
	}
	// NOTE: We DO NOT run client.Schema.Create() here because we are using Atlas for migrations.
	return client
}