package storage

import(
	"os"
	"context"
	"fmt"
	"log"
	"github.com/jackc/pgx/v5"
)

func Start() {
	conn, err := pgx.Connect(context.Background(), os.Getenv("DATABASE_URL"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to connect to database: %v\n", err)
		os.Exit(1)
	}

	err = conn.Ping(context.Background())
	 if err != nil {
		 log.Fatal("Error ping db")
	 }
 }
