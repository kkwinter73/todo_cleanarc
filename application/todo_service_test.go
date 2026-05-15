package application // このファイルはテスト用ヘルパーを定義する。
//
// _test.go ファイルなので、通常のビルドには含まれない。
// テスト実行時のみ参照されるため、本番コードに混入する心配がない。
//
// ここで定義しているもの:
//   - inMemoryTodoRepository: todo.Repository を満たす、メモリ上のmapで動く実装
//   - fakeClock             : Clock を満たす、固定時刻を返す実装
//
// これらは明日書くテストコード(TestTodoService_xxx)から使う。

// package application_test

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/kkwinter73/todo_cleanarc/domain/todo"
)

// ============================================================
// inMemoryTodoRepository: todo.Repository のテスト用InMemory実装
// ============================================================

// inMemoryTodoRepository は todo.Repository インターフェースを満たす
// メモリ上の実装。テスト時にPostgreSQLの代わりに使う。
//
// 型名は小文字始まり = パッケージ外から参照不可。
// テスト専用なので意図的にこうしている。
type inMemoryTodoRepository struct {
	mu    sync.Mutex             // 並行テスト時の競合防止
	store map[todo.ID]*todo.Todo // Todoの保管庫
}

// コンパイル時にインターフェース実装を検証するイディオム。
// もし Save/FindByID/Delete のシグネチャを間違えたらここでビルドが落ちる。
var _ todo.Repository = (*inMemoryTodoRepository)(nil)

// errTodoNotFound はテスト用のエラー。
// 本物のインフラ層では persistence.ErrTodoNotFound が定義されているが、
// テストヘルパーではここで完結させる。
var errTodoNotFound = errors.New("todo not found (in-memory)")

func newInMemoryTodoRepository() *inMemoryTodoRepository {
	return &inMemoryTodoRepository{
		store: make(map[todo.ID]*todo.Todo),
	}
}

// Save: mapに保存する。新規・更新は気にしない(キーに上書き)。
// PostgreSQL版はUPSERTで複雑だったが、メモリならこれだけ。
func (r *inMemoryTodoRepository) Save(ctx context.Context, t *todo.Todo) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.store[t.ID()] = t
	return nil
}

// FindByID: mapから取り出す。なければエラー。
func (r *inMemoryTodoRepository) FindByID(ctx context.Context, id todo.ID) (*todo.Todo, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	t, ok := r.store[id]
	if !ok {
		return nil, errTodoNotFound
	}
	return t, nil
}

// Delete: mapから消す。存在しないIDでもエラーにしない(冪等)。
// 本物のインフラ層と挙動を合わせている。
func (r *inMemoryTodoRepository) Delete(ctx context.Context, id todo.ID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.store, id)
	return nil
}

// ============================================================
// fakeClock: Clock のテスト用実装(固定時刻を返す)
// ============================================================

// fakeClock は application.Clock を満たす、固定時刻を返す実装。
// テストで時刻を制御するために使う。
//
// なぜ必要か:
//   - 本番のSystemClockは time.Now() を返すので、テストの度に値が変わる
//   - 「Complete を呼んだら CompletedAt が ○○ になる」というテストでは
//     時刻を固定値にしておく必要がある
type fakeClock struct {
	fixed time.Time
}

func (c fakeClock) Now() time.Time { return c.fixed }

// newFakeClock は指定された時刻を返す fakeClock を作る。
func newFakeClock(t time.Time) fakeClock {
	return fakeClock{fixed: t}
}

// ============================================================
// 注: テストコード本体(TestTodoService_xxx)は明日書く
// ============================================================
//
// 明日書くテストの例:
//
//   func TestTodoService_CompleteTodo(t *testing.T) {
//       repo := newInMemoryTodoRepository()
//       clock := newFakeClock(time.Date(2026, 5, 13, 10, 0, 0, 0, time.UTC))
//       service := application.NewTodoService(repo, clock)
//
//       // 事前にTodoを保存
//       td, _ := todo.New("買い物", nil, todo.PriorityMedium)
//       repo.Save(context.Background(), td)
//
//       // ユースケース実行
//       err := service.CompleteTodo(context.Background(), td.ID())
//       if err != nil { t.Fatal(err) }
//
//       // 検証: 取り直したらcompletedになっているはず
//       got, _ := repo.FindByID(context.Background(), td.ID())
//       if got.Status() != todo.StatusCompleted {
//           t.Errorf("Status = %v, want %v", got.Status(), todo.StatusCompleted)
//       }
//   }
