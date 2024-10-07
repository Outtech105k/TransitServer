package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/gin-gonic/gin"
	"outtech105.com/transit_server/database"
	"outtech105.com/transit_server/handler"
)

func main() {
	db, err := database.ConnectDB(10)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	engine := gin.Default()

	engine.POST("/search", handler.SearchTransitHandler(db))

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
