package db

import (
    "os"

    "entgo.io/ent/dialect"
    "entgo.io/ent/dialect/sql"
    _ "github.com/lib/pq"
    "github.com/sirupsen/logrus"

    "go-rag/ent/ent"
)

func NewClient() *ent.Client {
    logrus.Debug("initializing database connection")
    
    dbURL := os.Getenv("DATABASE_URL")
    if dbURL == "" {
        logrus.Fatal("DATABASE_URL environment variable is not set")
    }
    
    logrus.WithField("database_url", "***").Debug("database URL loaded from environment")

    logrus.Debug("opening database connection")
    drv, err := sql.Open(dialect.Postgres, dbURL)
    if err != nil {
        logrus.WithError(err).Fatal("failed opening connection to postgres")
    }

    logrus.Debug("creating ent client")
    client := ent.NewClient(ent.Driver(drv))
    
    logrus.Info("database client created successfully")
    return client
}
