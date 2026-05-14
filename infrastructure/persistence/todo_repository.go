// Package persistence はドメインのリポジトリインターフェースの実装を提供する。
// このパッケージは「インフラ層」に属し、DB（PostgreSQL）への接続詳細を担う。
//
// ドメイン層はこのパッケージを知らない（依存方向は infra → domain）。
package persistence

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	// ↓ モジュール名は適宜書き換える
	"github.com/kkwinter73/todo_cleanarc/domain/todo"
)

// TodoRepository は todo.Repository インターフェースのPostgreSQL実装。
//
// コンパイル時にインターフェースを満たしていることを保証するため、
// 下記の空変数代入でチェックする（Goのイディオム）。
var _ todo.Repository = (*TodoRepository)(nil)

// TodoRepository はTodoの永続化を担う。
type TodoRepository struct {
	pool *pgxpool.Pool
}

// NewTodoRepository はリポジトリのコンストラクタ。
// 接続プールを外から注入する（テスト容易性のため）。
func NewTodoRepository(pool *pgxpool.Pool) *TodoRepository {
	return &TodoRepository{pool: pool}
}

// ============================================================
// Save: 新規挿入 or 更新（UPSERT）
// ============================================================

// Save はTodoを保存する。
// 既存ならupdate、新規ならinsertだが、呼び出し側はそれを知らなくてよい。
// PostgreSQLのON CONFLICTを使ってUPSERTで実現する。
func (r *TodoRepository) Save(ctx context.Context, t *todo.Todo) error {
	const query = `
		INSERT INTO todos (id, title, due_date, priority, status, completed_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (id) DO UPDATE SET
			title        = EXCLUDED.title,
			due_date     = EXCLUDED.due_date,
			priority     = EXCLUDED.priority,
			status       = EXCLUDED.status,
			completed_at = EXCLUDED.completed_at
	`

	_, err := r.pool.Exec(ctx, query,
		t.ID().String(),
		t.Title(),
		t.DueDate(), // *time.Time はそのまま渡せる（nilならNULL）
		int(t.Priority()),
		string(t.Status()),
		t.CompletedAt(),
	)
	if err != nil {
		return fmt.Errorf("save todo: %w", err)
	}
	return nil
}

// ============================================================
// FindByID: IDで1件取得
// ============================================================

// ErrTodoNotFound はTodoが見つからなかったことを表すエラー。
// インフラ層のエラーなので、このパッケージで定義する。
// ドメインに置かないのは、これが「永続化の都合のエラー」だから。
var ErrTodoNotFound = errors.New("todo not found")

// FindByID はIDで1件取得する。
// 見つからない場合は ErrTodoNotFound を返す。
func (r *TodoRepository) FindByID(ctx context.Context, id todo.ID) (*todo.Todo, error) {
	const query = `
		SELECT id, title, due_date, priority, status, completed_at
		FROM todos
		WHERE id = $1
	`

	// スキャン先の変数を用意
	var (
		idStr       string
		title       string
		dueDate     *time.Time // pgxは*time.TimeにNULLをそのまま渡せる
		priority    int
		status      string
		completedAt *time.Time
	)

	err := r.pool.QueryRow(ctx, query, id.String()).Scan(
		&idStr,
		&title,
		&dueDate,
		&priority,
		&status,
		&completedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrTodoNotFound
		}
		return nil, fmt.Errorf("find todo by id: %w", err)
	}

	// ドメインオブジェクトに復元
	// Reconstruct はバリデーションをスキップする（DBの値は検証済みの前提）
	return todo.Reconstruct(
		todo.ID(idStr),
		title,
		dueDate,
		todo.Priority(priority),
		todo.Status(status),
		completedAt,
	), nil
}

// ============================================================
// Delete: IDで削除
// ============================================================

// Delete はIDで削除する。
// 存在しないIDを渡してもエラーにしない（冪等）。
func (r *TodoRepository) Delete(ctx context.Context, id todo.ID) error {
	const query = `DELETE FROM todos WHERE id = $1`

	_, err := r.pool.Exec(ctx, query, id.String())
	if err != nil {
		return fmt.Errorf("delete todo: %w", err)
	}
	return nil
}
