package main

import (
    "context"
    "log"
    "os"
    "entgo.io/ent/dialect"
    "entgo.io/ent/dialect/sql"
    _ "github.com/lib/pq"
     "go-rag/internal/ent"
)

func initEntClient() *ent.Client {
    dbURL := os.Getenv("DB_URL")
    drv, err := sql.Open(dialect.Postgres, dbURL)
    if err != nil {
        log.Fatalf("failed opening connection to postgres: %v", err)
    }
    client := ent.NewClient(ent.Driver(drv))
    if err := client.Schema.Create(context.Background()); err != nil {
        log.Fatalf("failed creating schema resources: %v", err)
    }
    return client
}
