package storage

import(
	"os"
	"context"
	"fmt"
	"log"
	"github.com/jackc/pgx/v5"
)

type userRepository interface {
	GetUserCity(ctx context.Context, userID int64) (string, error)
	UpdateUserCity(ctx context.Context, userID int64, city string) error
	GetTimeZone(ctx context.Context, userID int64) (string, error)
	UpdateTimeZone(ctx context.Context, userID int64, city string) error
  CreateUser(ctx context.Context, userID int64, username string) error
}

func Start() *pgx.Conn {
	conn, err := pgx.Connect(context.Background(), os.Getenv("DATABASE_URL"))
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
