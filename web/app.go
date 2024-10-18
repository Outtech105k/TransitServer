package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"outtech105.com/transit_server/database"
	"outtech105.com/transit_server/handler"
)

func main() {
	// DB接続
	db, err := database.ConnectDB(10)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	// エンドポイントとサーバ起動
	engine := setupRouter(db)
	srv := createServer(engine)

	// Graceful Shutdownの処理
	gracefulShutdown(srv)
}

// ルーターの設定
func setupRouter(db *sqlx.DB) *gin.Engine {
	engine := gin.Default()

	root := engine.Group("/api/v2/traffic")
	root.GET("/station", handler.GetStationsByKeyword(db))
	root.GET("/station/:id", handler.GetStationByID(db))
	root.POST("/search", handler.SearchTransitHandler(db))

	return engine
}

// サーバの作成
func createServer(handler http.Handler) *http.Server {
	return &http.Server{
		Addr:    ":80",
		Handler: handler,
	}
}

// Graceful Shutdownの実装
func gracefulShutdown(srv *http.Server) {
	// サーバを別のGoルーチンで起動
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Panicf("listen: %s\n", err)
		}
	}()

	// OSのシグナルを待機
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit
	log.Println("Shutting down server...")

	// タイムアウトを設定してGraceful Shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Panic("Server forced to shutdown:", err)
	}

	log.Println("Server exiting")
}
