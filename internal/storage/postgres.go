package storage

import(
	"os"
	"context"
	"fmt"
	"log"
	"github.com/jackc/pgx/v5/pgxpool"
)

func Start() *pgxpool.Pool {
	conn, err := pgxpool.New(context.Background(), os.Getenv("DATABASE_URL"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to connect to database: %v\n", err)
		os.Exit(1)
	}

	err = conn.Ping(context.Background())
	 if err != nil {
		 log.Fatal("Error ping db")
	 }
	 return conn
 }
