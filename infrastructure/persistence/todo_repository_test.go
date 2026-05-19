package persistence

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kkwinter73/todo_cleanarc/domain/todo"
)

// ============================================================
// テストセットアップ
// ============================================================

// テスト用DBの接続文字列。
// docker-compose.test.yml で立てた testuser/todo_app_test に接続する。
// CI環境などでは TEST_DATABASE_URL で上書き可能にしておく。
const defaultTestDSN = "postgres://testuser:testpassword@localhost:5433/todo_app_test?sslmode=disable"

// newTestPool はテスト用のDB接続プールを作る。
// 全テスト共通で使う想定。
func newTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper() // このメソッド内のエラーは呼び出し元の行番号で表示される

	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		dsn = defaultTestDSN
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("テストDB接続失敗: %v (docker-compose.test.yml は起動していますか?)", err)
	}
	if err := pool.Ping(ctx); err != nil {
		t.Fatalf("テストDB Ping失敗: %v", err)
	}
	return pool
}

// truncateTodos はテーブルを空にする。
// 各テストの最初に呼ぶことで、テスト間の独立性を保つ。
func truncateTodos(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	_, err := pool.Exec(context.Background(), "TRUNCATE TABLE todos")
	if err != nil {
		t.Fatalf("TRUNCATE失敗: %v", err)
	}
}

// ============================================================
// テスト本体
// ============================================================

// TestTodoRepository_Save_NewTodo: 新規Todoの保存(INSERT相当)
func TestTodoRepository_Save_NewTodo(t *testing.T) {
	pool := newTestPool(t)
	defer pool.Close()
	truncateTodos(t, pool)

	repo := NewTodoRepository(pool)
	ctx := context.Background()

	// Arrange: ドメインで新規Todoを作る
	due := time.Date(2026, 12, 31, 23, 59, 0, 0, time.UTC)
	td, err := todo.New("牛乳を買う", &due, todo.PriorityMedium)
	if err != nil {
		t.Fatalf("Todo生成失敗: %v", err)
	}

	// Act: 保存する
	if err := repo.Save(ctx, td); err != nil {
		t.Fatalf("Save失敗: %v", err)
	}

	// Assert: 取得して内容を検証
	got, err := repo.FindByID(ctx, td.ID())
	if err != nil {
		t.Fatalf("FindByID失敗: %v", err)
	}
	if got.Title() != "牛乳を買う" {
		t.Errorf("Title = %q, want %q", got.Title(), "牛乳を買う")
	}
	if got.Priority() != todo.PriorityMedium {
		t.Errorf("Priority = %v, want %v", got.Priority(), todo.PriorityMedium)
	}
	if got.Status() != todo.StatusPending {
		t.Errorf("Status = %v, want %v", got.Status(), todo.StatusPending)
	}
	if got.DueDate() == nil || !got.DueDate().Equal(due) {
		t.Errorf("DueDate = %v, want %v", got.DueDate(), due)
	}
}

// TestTodoRepository_Save_UpdateExisting: 既存Todoの更新(UPDATE相当)
// ON CONFLICT DO UPDATE が正しく動いているか確認。
func TestTodoRepository_Save_UpdateExisting(t *testing.T) {
	pool := newTestPool(t)
	defer pool.Close()
	truncateTodos(t, pool)

	repo := NewTodoRepository(pool)
	ctx := context.Background()

	// Arrange: 一度保存しておく
	td, _ := todo.New("元のタイトル", nil, todo.PriorityLow)
	if err := repo.Save(ctx, td); err != nil {
		t.Fatalf("初回Save失敗: %v", err)
	}

	// 同じTodoを完了状態にして、再度保存
	now := time.Date(2026, 5, 14, 10, 0, 0, 0, time.UTC)
	td.Complete(now)
	if err := repo.Save(ctx, td); err != nil {
		t.Fatalf("再Save失敗: %v", err)
	}

	// Assert: 上書きされていることを確認
	got, err := repo.FindByID(ctx, td.ID())
	if err != nil {
		t.Fatalf("FindByID失敗: %v", err)
	}
	if got.Status() != todo.StatusCompleted {
		t.Errorf("Status = %v, want %v", got.Status(), todo.StatusCompleted)
	}
	if got.CompletedAt() == nil || !got.CompletedAt().Equal(now) {
		t.Errorf("CompletedAt = %v, want %v", got.CompletedAt(), now)
	}
}

// TestTodoRepository_FindByID_NotFound: 存在しないIDのとき ErrTodoNotFound
func TestTodoRepository_FindByID_NotFound(t *testing.T) {
	pool := newTestPool(t)
	defer pool.Close()
	truncateTodos(t, pool)

	repo := NewTodoRepository(pool)
	ctx := context.Background()

	// 存在しないIDで取得
	_, err := repo.FindByID(ctx, todo.NewID())

	if !errors.Is(err, ErrTodoNotFound) {
		t.Errorf("err = %v, want ErrTodoNotFound", err)
	}
}

// TestTodoRepository_Delete: 削除できる
func TestTodoRepository_Delete(t *testing.T) {
	pool := newTestPool(t)
	defer pool.Close()
	truncateTodos(t, pool)

	repo := NewTodoRepository(pool)
	ctx := context.Background()

	// Arrange: 保存しておく
	td, _ := todo.New("削除対象", nil, todo.PriorityHigh)
	if err := repo.Save(ctx, td); err != nil {
		t.Fatalf("Save失敗: %v", err)
	}

	// Act: 削除
	if err := repo.Delete(ctx, td.ID()); err != nil {
		t.Fatalf("Delete失敗: %v", err)
	}

	// Assert: 取得するとErrTodoNotFound
	_, err := repo.FindByID(ctx, td.ID())
	if !errors.Is(err, ErrTodoNotFound) {
		t.Errorf("err = %v, want ErrTodoNotFound", err)
	}
}

// TestTodoRepository_Delete_NotFound: 存在しないIDの削除はエラーにならない(冪等)
func TestTodoRepository_Delete_NotFound(t *testing.T) {
	pool := newTestPool(t)
	defer pool.Close()
	truncateTodos(t, pool)

	repo := NewTodoRepository(pool)
	ctx := context.Background()

	// 存在しないIDを削除 → エラーにならない
	err := repo.Delete(ctx, todo.NewID())
	if err != nil {
		t.Errorf("err = %v, want nil (Deleteは冪等であるべき)", err)
	}
}

// TestTodoRepository_NullableFields: 期限なし(dueDate=nil)が正しく保存・復元される
// PostgreSQL の TIMESTAMPTZ で NULL を扱う部分が壊れていないか確認。
func TestTodoRepository_NullableFields(t *testing.T) {
	pool := newTestPool(t)
	defer pool.Close()
	truncateTodos(t, pool)

	repo := NewTodoRepository(pool)
	ctx := context.Background()

	// 期限なしのTodoを作る
	td, _ := todo.New("期限なしタスク", nil, todo.PriorityLow)
	if err := repo.Save(ctx, td); err != nil {
		t.Fatalf("Save失敗: %v", err)
	}

	got, err := repo.FindByID(ctx, td.ID())
	if err != nil {
		t.Fatalf("FindByID失敗: %v", err)
	}
	if got.DueDate() != nil {
		t.Errorf("DueDate = %v, want nil", got.DueDate())
	}
	if got.CompletedAt() != nil {
		t.Errorf("CompletedAt = %v, want nil", got.CompletedAt())
	}
}
