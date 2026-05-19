// アプリケーション層のテスト。
//
// テスト対象: TodoService の3ユースケース(CreateTodo / CompleteTodo / DeleteTodo)
//
// テスト戦略:
//   - 本物のDBは使わず、InMemoryモックでリポジトリを差し替える
//   - 時刻も fakeClock で固定値に差し替える
//   - DBなしで高速にユースケースの「段取り」を検証する
//
// _test.go ファイルなので本番ビルドに含まれない。
package application_test

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/kkwinter73/todo_cleanarc/application"
	"github.com/kkwinter73/todo_cleanarc/domain/todo"
)

// ============================================================
// テストヘルパー: InMemoryモック + fakeClock
// ============================================================

// inMemoryTodoRepository は todo.Repository を満たすメモリ実装。
// テスト用なので型名は小文字(パッケージ外から見えない)。
type inMemoryTodoRepository struct {
	mu    sync.Mutex
	store map[todo.ID]*todo.Todo
}

// コンパイル時に interface 実装を保証するイディオム。
var _ todo.Repository = (*inMemoryTodoRepository)(nil)

var errTodoNotFound = errors.New("todo not found (in-memory)")

func newInMemoryTodoRepository() *inMemoryTodoRepository {
	return &inMemoryTodoRepository{
		store: make(map[todo.ID]*todo.Todo),
	}
}

func (r *inMemoryTodoRepository) Save(ctx context.Context, t *todo.Todo) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.store[t.ID()] = t
	return nil
}

func (r *inMemoryTodoRepository) FindByID(ctx context.Context, id todo.ID) (*todo.Todo, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	t, ok := r.store[id]
	if !ok {
		return nil, errTodoNotFound
	}
	return t, nil
}

func (r *inMemoryTodoRepository) Delete(ctx context.Context, id todo.ID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.store, id)
	return nil
}

// count はテストで「リポジトリに何件保存されているか」を確認するためのヘルパー。
func (r *inMemoryTodoRepository) count() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.store)
}

// fakeClock は固定時刻を返す application.Clock 実装。
type fakeClock struct {
	fixed time.Time
}

func (c fakeClock) Now() time.Time { return c.fixed }

func newFakeClock(t time.Time) fakeClock {
	return fakeClock{fixed: t}
}

// ============================================================
// CreateTodo のテスト
// ============================================================

// 正常系: 正しい入力でTodoを作成 → IDが返り、リポジトリに保存される
func TestTodoService_CreateTodo_Success(t *testing.T) {
	repo := newInMemoryTodoRepository()
	clock := newFakeClock(time.Date(2026, 5, 16, 10, 0, 0, 0, time.UTC))
	service := application.NewTodoService(repo, clock)

	due := time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC)
	out, err := service.CreateTodo(context.Background(), application.CreateTodoInput{
		Title:    "牛乳を買う",
		DueDate:  &due,
		Priority: todo.PriorityMedium,
	})

	// エラーなし
	if err != nil {
		t.Fatalf("err = %v, want nil", err)
	}
	// IDが返ってきている
	if out == nil || out.ID == "" {
		t.Fatalf("output = %v, want non-empty ID", out)
	}
	// 副作用: リポジトリに1件保存されている
	if got := repo.count(); got != 1 {
		t.Errorf("repository count = %d, want 1", got)
	}
	// 副作用: 取得すると同じTodoが返る
	stored, err := repo.FindByID(context.Background(), out.ID)
	if err != nil {
		t.Fatalf("FindByID failed: %v", err)
	}
	if stored.Title() != "牛乳を買う" {
		t.Errorf("Title = %q, want %q", stored.Title(), "牛乳を買う")
	}
	if stored.Status() != todo.StatusPending {
		t.Errorf("Status = %v, want %v", stored.Status(), todo.StatusPending)
	}
}

// 異常系: ドメインのバリデーションエラーがそのまま伝わる
// ドメインエラーをプレゼン層やアプリ層で握りつぶさないことを確認する。
func TestTodoService_CreateTodo_ValidationErrors(t *testing.T) {
	// テーブル駆動テスト: 似たケースをまとめて書く
	tests := []struct {
		name    string
		title   string
		prio    todo.Priority
		wantErr error
	}{
		{"空タイトル", "", todo.PriorityMedium, todo.ErrTitleEmpty},
		{"空白のみ", "   ", todo.PriorityMedium, todo.ErrTitleEmpty},
		{"201文字(超過)", strings.Repeat("a", 201), todo.PriorityMedium, todo.ErrTitleTooLong},
		{"優先度ゼロ", "タスク", todo.Priority(0), todo.ErrInvalidPriority},
		{"優先度範囲外", "タスク", todo.Priority(99), todo.ErrInvalidPriority},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newInMemoryTodoRepository()
			service := application.NewTodoService(repo, newFakeClock(time.Now()))

			_, err := service.CreateTodo(context.Background(), application.CreateTodoInput{
				Title:    tt.title,
				DueDate:  nil,
				Priority: tt.prio,
			})

			// 期待されるドメインエラーが返るか
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("err = %v, want %v", err, tt.wantErr)
			}
			// 副作用: エラー時はリポジトリに保存されていない
			if got := repo.count(); got != 0 {
				t.Errorf("repository count = %d, want 0 (エラー時は保存されないはず)", got)
			}
		})
	}
}

// 境界値: タイトル200文字ちょうどは成功する
func TestTodoService_CreateTodo_TitleBoundary(t *testing.T) {
	repo := newInMemoryTodoRepository()
	service := application.NewTodoService(repo, newFakeClock(time.Now()))

	title := strings.Repeat("a", 200) // 上限ちょうど
	out, err := service.CreateTodo(context.Background(), application.CreateTodoInput{
		Title:    title,
		DueDate:  nil,
		Priority: todo.PriorityLow,
	})

	if err != nil {
		t.Fatalf("200文字は成功するはず: err = %v", err)
	}
	if out == nil {
		t.Fatalf("output = nil, want non-nil")
	}
}

// ============================================================
// CompleteTodo のテスト
// ============================================================

// 正常系: 未完了Todoを完了にする → ステータスが変わり、CompletedAtに渡したnowが入る
func TestTodoService_CompleteTodo_Success(t *testing.T) {
	repo := newInMemoryTodoRepository()
	fixedNow := time.Date(2026, 5, 16, 10, 0, 0, 0, time.UTC)
	service := application.NewTodoService(repo, newFakeClock(fixedNow))

	// Arrange: 事前にTodoを作っておく
	td, _ := todo.New("タスク", nil, todo.PriorityMedium)
	if err := repo.Save(context.Background(), td); err != nil {
		t.Fatalf("setup Save failed: %v", err)
	}

	// Act
	err := service.CompleteTodo(context.Background(), td.ID())

	// Assert
	if err != nil {
		t.Fatalf("err = %v, want nil", err)
	}

	// 副作用: ステータスが completed になっている
	got, _ := repo.FindByID(context.Background(), td.ID())
	if got.Status() != todo.StatusCompleted {
		t.Errorf("Status = %v, want %v", got.Status(), todo.StatusCompleted)
	}
	// 副作用: CompletedAt に fakeClock の固定時刻が入っている
	if got.CompletedAt() == nil {
		t.Fatal("CompletedAt = nil, want non-nil")
	}
	if !got.CompletedAt().Equal(fixedNow) {
		t.Errorf("CompletedAt = %v, want %v", got.CompletedAt(), fixedNow)
	}
}

// 異常系: 存在しないIDを完了しようとするとエラー(リポジトリのErrTodoNotFoundがそのまま伝わる)
func TestTodoService_CompleteTodo_NotFound(t *testing.T) {
	repo := newInMemoryTodoRepository()
	service := application.NewTodoService(repo, newFakeClock(time.Now()))

	err := service.CompleteTodo(context.Background(), todo.NewID())

	if err == nil {
		t.Fatal("err = nil, want non-nil (存在しないIDなのでエラーになるはず)")
	}
}

// 冪等性: 完了済みTodoに再度Completeを呼んでも壊れない、CompletedAtは初回のまま
// ドメイン側の Complete は冪等になっているはず(StatusCompletedなら何もしない)
func TestTodoService_CompleteTodo_Idempotent(t *testing.T) {
	repo := newInMemoryTodoRepository()
	firstNow := time.Date(2026, 5, 16, 10, 0, 0, 0, time.UTC)
	service := application.NewTodoService(repo, newFakeClock(firstNow))

	// 1回目の完了
	td, _ := todo.New("タスク", nil, todo.PriorityMedium)
	repo.Save(context.Background(), td)
	_ = service.CompleteTodo(context.Background(), td.ID())

	// 時刻を進めて、2回目の完了
	secondNow := firstNow.Add(1 * time.Hour)
	service2 := application.NewTodoService(repo, newFakeClock(secondNow))
	err := service2.CompleteTodo(context.Background(), td.ID())

	if err != nil {
		t.Fatalf("2回目もエラーにならないはず: err = %v", err)
	}

	// CompletedAt は初回のままで、2回目の時刻に上書きされていないこと
	got, _ := repo.FindByID(context.Background(), td.ID())
	if !got.CompletedAt().Equal(firstNow) {
		t.Errorf("CompletedAt = %v, want %v (初回の時刻のまま上書きされないはず)", got.CompletedAt(), firstNow)
	}
}

// ============================================================
// DeleteTodo のテスト
// ============================================================

// 正常系: 存在するTodoを削除 → リポジトリから消える
func TestTodoService_DeleteTodo_Success(t *testing.T) {
	repo := newInMemoryTodoRepository()
	service := application.NewTodoService(repo, newFakeClock(time.Now()))

	td, _ := todo.New("削除対象", nil, todo.PriorityHigh)
	repo.Save(context.Background(), td)

	err := service.DeleteTodo(context.Background(), td.ID())

	if err != nil {
		t.Fatalf("err = %v, want nil", err)
	}
	// 副作用: リポジトリから消えている
	if got := repo.count(); got != 0 {
		t.Errorf("repository count = %d, want 0", got)
	}
}

// 冪等性: 存在しないIDを削除してもエラーにならない
// (InMemoryもPostgreSQLも、存在しないIDのDELETEは何もしない仕様)
func TestTodoService_DeleteTodo_NotFound(t *testing.T) {
	repo := newInMemoryTodoRepository()
	service := application.NewTodoService(repo, newFakeClock(time.Now()))

	err := service.DeleteTodo(context.Background(), todo.NewID())

	if err != nil {
		t.Errorf("err = %v, want nil (削除は冪等であるべき)", err)
	}
}

// 副作用の範囲: 削除しても、他のTodoは影響を受けない
func TestTodoService_DeleteTodo_DoesNotAffectOthers(t *testing.T) {
	repo := newInMemoryTodoRepository()
	service := application.NewTodoService(repo, newFakeClock(time.Now()))

	// 2件保存
	td1, _ := todo.New("残るほう", nil, todo.PriorityLow)
	td2, _ := todo.New("消えるほう", nil, todo.PriorityHigh)
	repo.Save(context.Background(), td1)
	repo.Save(context.Background(), td2)

	// td2 だけ削除
	if err := service.DeleteTodo(context.Background(), td2.ID()); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// td1 はまだ残っている
	if _, err := repo.FindByID(context.Background(), td1.ID()); err != nil {
		t.Errorf("td1 が消えてしまった: %v", err)
	}
	// td2 は消えている
	if _, err := repo.FindByID(context.Background(), td2.ID()); err == nil {
		t.Error("td2 がまだ存在している")
	}
	// 件数も正しい
	if got := repo.count(); got != 1 {
		t.Errorf("repository count = %d, want 1", got)
	}
}
