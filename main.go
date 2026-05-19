// main.go はアプリケーションの組み立て係(Composition Root)。
//
// 全層の実体をここで生成し、依存を「外側から内側へ」注入していく:
//  1. インフラ層: DB接続プール + リポジトリ
//  2. アプリ層 : TodoService
//  3. プレゼン層: HTTPハンドラ + ルーター
//  4. HTTPサーバー起動
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kkwinter73/todo_cleanarc/application"
	"github.com/kkwinter73/todo_cleanarc/infrastructure/persistence"
	httpx "github.com/kkwinter73/todo_cleanarc/presentation/http"
)

func main() {
	ctx := context.Background()

	// ============================================================
	// 1. インフラ層: DB接続プール + リポジトリ
	// ============================================================
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://user:password@localhost:5432/todo_app?sslmode=disable"
	}

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		log.Fatalf("DB接続失敗: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("Ping失敗: %v", err)
	}
	log.Println("✓ DB接続OK")

	repo := persistence.NewTodoRepository(pool)

	// ============================================================
	// 2. アプリ層: TodoService
	// ============================================================
	service := application.NewTodoService(repo, application.SystemClock{})

	// ============================================================
	// 3. プレゼン層: HTTPハンドラ + ルーター
	// ============================================================
	handler := httpx.NewTodoHandler(service)
	router := httpx.NewRouter(handler)

	// ============================================================
	// 4. HTTPサーバー起動(グレースフルシャットダウン対応)
	// ============================================================
	addr := os.Getenv("HTTP_ADDR")
	if addr == "" {
		addr = ":8080"
	}

	server := &http.Server{
		Addr:              addr,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second, // セキュリティ対策(Slowloris攻撃防止)
	}

	// シャットダウンシグナル待ち受け用
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	// サーバーを別goroutineで起動
	go func() {
		log.Printf("✓ HTTPサーバー起動: http://localhost%s", addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("ListenAndServe失敗: %v", err)
		}
	}()

	// Ctrl+C などで停止シグナルを受けたら、グレースフルにシャットダウン
	<-stop
	log.Println("シャットダウン中...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Shutdown失敗: %v", err)
	}
	log.Println("✓ シャットダウン完了")
}
