package database

import (
	"database/sql"
	"fmt"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

func ConnectDB(MaxRetryCount int) (*sql.DB, error) {
	for r := 1; r <= MaxRetryCount; r++ {
		db, err := sql.Open("mysql", fmt.Sprintf(
			"%s:%s@tcp(db:3306)/%s?parseTime=true",
			os.Getenv("MYSQL_USER"),
			os.Getenv("MYSQL_PASSWORD"),
			os.Getenv("MYSQL_DATABASE"),
		))
		if err != nil {
			return nil, fmt.Errorf("dbConnection: %w", err)
		}

		err = db.Ping()
		if err == nil {
			fmt.Println("DB connection successful")
			return db, nil
		}

		fmt.Println("DB Connection Error: " + err.Error())
		time.Sleep(5 * time.Second)
	}
	return nil, fmt.Errorf("DB connection error occured %d times", MaxRetryCount)
}
