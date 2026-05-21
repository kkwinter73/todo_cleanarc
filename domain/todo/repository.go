package todo

import (
	"context"
	"errors"
)

// ErrTodoNotFound は「指定IDのTodoが存在しない」ことを表すドメインエラー。
//
// インフラ層(PostgreSQL、InMemory等)はこのエラーを返す。
// プレゼン層は errors.Is でこれを判定し、HTTPの404に翻訳する。
//
// このエラーをドメイン層に置くことで、
//   - インフラ実装が変わっても、エラー型は変わらない
//   - プレゼン層がインフラ層を知らずに済む
//
// という設計上の利点が得られる。
var ErrTodoNotFound = errors.New("todo not found")

// Repository はTodoの永続化を抽象化するインターフェース。
//
// ドメイン層にインターフェースを置き、実装はインフラ層に置く。
// ドメイン層をDBなどの技術的詳細から独立させる。
type Repository interface {
	// Save は既存ならupdate、新規ならinsertを実装側が判断する。
	// 呼び出し側はそれを知らなくて良い。
	Save(ctx context.Context, t *Todo) error

	// FindByID はIDで1件取得する。
	// 見つからない場合は ErrTodoNotFound を返す。
	FindByID(ctx context.Context, id ID) (*Todo, error)

	// Delete はIDで削除する。
	// 存在しないIDの削除はエラーにしない(冪等)。
	Delete(ctx context.Context, id ID) error
}
