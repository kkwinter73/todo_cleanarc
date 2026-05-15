// main.go はアプリケーションの組み立て係(Composition Root)。
//
// この場所だけが、全ての層の具体型を知っている特別な場所。
// インフラ・アプリケーション・(将来的に)プレゼンテーション層の実体をここで生成し、
// 依存を「外側から内側へ」注入していく。
//
// 動作確認用のシナリオ:
//
//	Create → Find → Complete → Find → Delete
//
// 本物のCLIやHTTPサーバーを実装するまでの繋ぎとして使う。
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kkwinter73/todo_cleanarc/application"
	"github.com/kkwinter73/todo_cleanarc/domain/todo"
	"github.com/kkwinter73/todo_cleanarc/infrastructure/persistence"
)

func main() {
	ctx := context.Background()

	// ============================================================
	// 1. インフラ層の組み立て: DB接続プール + リポジトリ
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
	fmt.Println("✓ DB接続OK")

	// インフラ層の実体を生成。
	// repo の型は *persistence.TodoRepository だが、
	// アプリ層に渡すときは todo.Repository インターフェースとして扱われる。
	repo := persistence.NewTodoRepository(pool)

	// ============================================================
	// 2. アプリケーション層の組み立て: TodoService
	// ============================================================
	// 依存を注入する: リポジトリと時計
	// アプリ層は repo の具体型(PostgreSQL実装)を知らない。
	// インターフェース越しに呼び出すだけ。
	service := application.NewTodoService(repo, application.SystemClock{})

	// ============================================================
	// 3. 動作確認シナリオ(プレゼン層の代わりに main から直接呼ぶ)
	// ============================================================

	// Create
	due := time.Now().Add(7 * 24 * time.Hour)
	out, err := service.CreateTodo(ctx, application.CreateTodoInput{
		Title:    "牛乳を買う",
		DueDate:  &due,
		Priority: todo.PriorityMedium,
	})
	if err != nil {
		log.Fatalf("CreateTodo失敗: %v", err)
	}
	fmt.Printf("✓ CreateTodo OK: ID=%s\n", out.ID)

	// Find (確認のため、リポジトリ直接呼び出し)
	t1, err := repo.FindByID(ctx, out.ID)
	if err != nil {
		log.Fatalf("FindByID失敗: %v", err)
	}
	fmt.Printf("✓ FindByID OK: Title=%s, Status=%s\n", t1.Title(), t1.Status())

	// Complete
	if err := service.CompleteTodo(ctx, out.ID); err != nil {
		log.Fatalf("CompleteTodo失敗: %v", err)
	}
	fmt.Println("✓ CompleteTodo OK")

	// 再Find
	t2, err := repo.FindByID(ctx, out.ID)
	if err != nil {
		log.Fatalf("再FindByID失敗: %v", err)
	}
	fmt.Printf("✓ 再取得: Status=%s, CompletedAt=%v\n", t2.Status(), t2.CompletedAt())

	// Delete
	if err := service.DeleteTodo(ctx, out.ID); err != nil {
		log.Fatalf("DeleteTodo失敗: %v", err)
	}
	fmt.Println("✓ DeleteTodo OK")

	// 削除確認
	_, err = repo.FindByID(ctx, out.ID)
	if err == persistence.ErrTodoNotFound {
		fmt.Println("✓ 削除後のFindByIDで ErrTodoNotFound が返った(期待通り)")
	} else {
		log.Fatalf("期待外のエラー: %v", err)
	}

	fmt.Println("\n全シナリオ成功 🎉")
}
