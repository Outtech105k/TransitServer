package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/gin-gonic/gin"
)

func main() {
	// Ginのエンジンを生成
	engine := gin.Default()

	// ルートハンドラの定義
	engine.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "hello world",
		})
	})

	// サーバーの設定
	srv := &http.Server{
		Addr:    ":80",  // ポート設定
		Handler: engine, // Ginのハンドラを使用
	}

	// サーバーを非同期で開始
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	// Graceful Shutdownを設定
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt) // SIGINT (Ctrl+C)などのシグナルを受け取る
	<-quit                            // シグナルを受け取るまで待機
	log.Println("Shutting down server...")

	// コンテキストを使って最大5秒間の猶予を持たせてサーバーをシャットダウン
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	log.Println("Server exiting")
}
