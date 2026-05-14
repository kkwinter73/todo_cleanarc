// 動作確認用のスクリプト。
// リポジトリの Save / FindByID / Delete が実際にPostgreSQLと繋がって動くかを
// 目視で確認する用途。後で本格的なアプリケーションエントリに置き換える。
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	// ↓ モジュール名は適宜書き換える
	"github.com/kkwinter73/todo_cleanarc/domain/todo"
	"github.com/kkwinter73/todo_cleanarc/infrastructure/persistence"
)

func main() {
	ctx := context.Background()

	// 接続文字列は環境変数から取る。
	// デフォルト値は docker-compose.yml の設定に合わせている。
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://user:password@localhost:5432/todo_app?sslmode=disable"
	}

	// 接続プールを作成
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		log.Fatalf("DB接続失敗: %v", err)
	}
	defer pool.Close()

	// 接続確認
	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("Ping失敗: %v", err)
	}
	fmt.Println("✓ DB接続OK")

	// リポジトリを生成
	repo := persistence.NewTodoRepository(pool)

	// ============================================================
	// シナリオ: Todoを作って → 保存 → 取得 → 完了 → 再保存 → 削除
	// ============================================================

	// 1. ドメインで Todo を作る
	due := time.Now().Add(7 * 24 * time.Hour)
	t, err := todo.New("牛乳を買う", &due, todo.PriorityMedium)
	if err != nil {
		log.Fatalf("Todo生成失敗: %v", err)
	}
	fmt.Printf("✓ Todo生成: ID=%s, Title=%s\n", t.ID(), t.Title())

	// 2. リポジトリで保存（INSERT）
	if err := repo.Save(ctx, t); err != nil {
		log.Fatalf("Save失敗: %v", err)
	}
	fmt.Println("✓ Save (INSERT) OK")

	// 3. IDで取得
	found, err := repo.FindByID(ctx, t.ID())
	if err != nil {
		log.Fatalf("FindByID失敗: %v", err)
	}
	fmt.Printf("✓ FindByID OK: Title=%s, Status=%s\n", found.Title(), found.Status())

	// 4. 完了状態にして再保存（UPDATE）
	found.Complete(time.Now())
	if err := repo.Save(ctx, found); err != nil {
		log.Fatalf("再Save失敗: %v", err)
	}
	fmt.Println("✓ Save (UPDATE) OK")

	// 5. 再取得して完了状態を確認
	found2, err := repo.FindByID(ctx, t.ID())
	if err != nil {
		log.Fatalf("再FindByID失敗: %v", err)
	}
	fmt.Printf("✓ 再取得: Status=%s, CompletedAt=%v\n", found2.Status(), found2.CompletedAt())

	// 6. 削除
	if err := repo.Delete(ctx, t.ID()); err != nil {
		log.Fatalf("Delete失敗: %v", err)
	}
	fmt.Println("✓ Delete OK")

	// 7. 削除後にFindByIDするとErrTodoNotFoundが返るはず
	_, err = repo.FindByID(ctx, t.ID())
	if err == persistence.ErrTodoNotFound {
		fmt.Println("✓ 削除後のFindByIDで ErrTodoNotFound が返った（期待通り）")
	} else {
		log.Fatalf("期待外のエラー: %v", err)
	}

	fmt.Println("\n全シナリオ成功 🎉")
}
