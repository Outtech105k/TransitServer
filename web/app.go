package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
	"outtech105.com/gin_template/handler"
)

func main() {
	db, err := connectDB(10)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	engine := gin.Default()

	dbHandler := handler.NewDBHandler(db)

	engine.GET("/", handler.RootHandler)
	engine.POST("/search", dbHandler.SearchTransitHandle)

	srv := &http.Server{
		Addr:    ":80",
		Handler: engine,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	log.Println("Server exiting")
}

func connectDB(MaxRetryCount int) (*sql.DB, error) {
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
