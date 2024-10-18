package database

import (
	"fmt"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
)

// MySQL接続処理(maxRetryCountだけ試行)
func ConnectDB(maxRetryCount int) (*sqlx.DB, error) {
	for r := 1; r <= maxRetryCount; r++ {
		db, err := sqlx.Open("mysql", fmt.Sprintf(
			"%s:%s@tcp(db:3306)/%s?parseTime=true",
			os.Getenv("MYSQL_USER"),
			os.Getenv("MYSQL_PASSWORD"),
			os.Getenv("MYSQL_DATABASE"),
		))
		if err != nil {
			return nil, fmt.Errorf("dbConnection: %w", err)
		}

		// DB接続が確立されたら離脱
		err = db.Ping()
		if err == nil {
			fmt.Println("DB connection successful")
			return db, nil
		}

		// 接続失敗時は5秒待機後リトライ
		fmt.Println("DB Connection Error: " + err.Error())
		time.Sleep(5 * time.Second)
	}
	return nil, fmt.Errorf("DB connection error occured %d times", maxRetryCount)
}
