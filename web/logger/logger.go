package logger

import (
	"log/slog"
	"os"
)

var Logger *slog.Logger

func InitLogger() {
	Logger = slog.New(slog.NewJSONHandler(os.Stdout, nil)) // 例: JSON形式のログハンドラーを使用
}
