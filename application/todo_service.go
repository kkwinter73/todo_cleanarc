// Package application はユースケース（アプリケーションサービス）を提供する。
//
// この層の役割は「業務処理の段取り」を書くこと:
//   - ドメイン層からエンティティを取得する
//   - ドメインのメソッドを呼んで業務ルールを実行する
//   - リポジトリに保存する
//
// 業務ルールそのものはドメイン層に委ねる。
// 永続化の詳細(SQLなど)はインフラ層に委ねる。
//
// 依存方向: application -> domain（インフラ層は知らない）
package application

import (
	"context"
	"time"

	"github.com/kkwinter73/todo_cleanarc/domain/todo"
)

// ============================================================
// TodoUsecase: Todoユースケースの契約
// ============================================================

// TodoUsecase はTodoに関するユースケースの契約。
//
// アプリケーション層が外側(プレゼン層)に対して公開する「窓口」。
// プレゼン層はこのインターフェースに依存し、TodoService(具象)には直接依存しない。
//
// 設計方針:
//   - クリーンアーキテクチャ原典に沿い、ユースケースの抽象は
//     「呼ばれる側」であるアプリケーション層で定義する。
//   - インターフェースと実装(TodoService)が同じパッケージに置かれるため、
//     メソッド追加時の同期が取りやすい。
type TodoUsecase interface {
	CreateTodo(ctx context.Context, in CreateTodoInput) (*CreateTodoOutput, error)
	FindTodo(ctx context.Context, id todo.ID) (*FindTodoOutput, error)
	CompleteTodo(ctx context.Context, id todo.ID) error
	DeleteTodo(ctx context.Context, id todo.ID) error
}

// コンパイル時に TodoService が TodoUsecase を満たすことを保証する。
//
// この行があると、将来 TodoUsecase にメソッドを追加した時に
// TodoService 側に未実装があれば、コンパイルエラーで即座に気づける。
var _ TodoUsecase = (*TodoService)(nil)

// ============================================================
// Clock: 時刻取得を抽象化する
// ============================================================

// Clock は現在時刻を取得するインターフェース。
//
// アプリケーション層から time.Now() を直接呼ばずに、
// このインターフェース経由で時刻を取得する。a
// これによりテスト時に時刻を固定でき、テスタビリティが上がる。
type Clock interface {
	Now() time.Time
}

// SystemClock は本番用の Clock 実装。time.Now() をそのまま返す。
type SystemClock struct{}

func (SystemClock) Now() time.Time { return time.Now() }

// ============================================================
// TodoService: Todoに関するユースケースを束ねる
// ============================================================

// TodoService はTodoに関するユースケース(アプリケーションサービス)を提供する。
//
// 依存は「ドメインの抽象」のみ:
//   - todo.Repository (インターフェース) ← 実体はインフラ層から注入される
//   - Clock          (インターフェース) ← 実体はSystemClockやテスト用Fakeから注入される
//
// インフラ層の具体型(persistence.TodoRepository など)は知らない。
type TodoService struct {
	repo  todo.Repository
	clock Clock
}

// NewTodoService はTodoServiceのコンストラクタ。
// 依存を外から注入する(Dependency Injection)。
func NewTodoService(repo todo.Repository, clock Clock) *TodoService {
	return &TodoService{
		repo:  repo,
		clock: clock,
	}
}

// ============================================================
// CreateTodo: Todoを新規作成する
// ============================================================

// CreateTodoInput はCreateTodoユースケースの入力。
// アプリケーション層の入出力は専用の型(DTO)で表現する。
// これによりプレゼン層の入力形式(HTTPリクエストなど)とアプリ層を分離できる。
type CreateTodoInput struct {
	Title    string
	DueDate  *time.Time
	Priority todo.Priority
}

// CreateTodoOutput はCreateTodoユースケースの出力。
type CreateTodoOutput struct {
	ID todo.ID
}

// CreateTodo は新しいTodoを作成して保存する。
func (s *TodoService) CreateTodo(ctx context.Context, in CreateTodoInput) (*CreateTodoOutput, error) {
	t, err := todo.New(in.Title, in.DueDate, in.Priority)
	if err != nil {
		return nil, err
	}
	if err := s.repo.Save(ctx, t); err != nil {
		return nil, err
	}
	return &CreateTodoOutput{ID: t.ID()}, nil
}

// ============================================================
// FindTodo: IDでTodoを1件取得する
// ============================================================

// FindTodoOutput は FindTodo ユースケースの出力。
// ドメインの Todo をそのまま返さず、プレゼン層に渡しやすい形にする。
//
// 注: ドメインの Todo を直接返す設計もありうるが、
// 「アプリ層の出力はDTOで表現する」一貫性のためここでも専用型を使う。
type FindTodoOutput struct {
	ID          todo.ID
	Title       string
	DueDate     *time.Time
	Priority    todo.Priority
	Status      todo.Status
	CompletedAt *time.Time
}

// FindTodo はIDでTodoを1件取得する。
// 見つからない場合はリポジトリ実装が返すエラー(ErrTodoNotFound等)がそのまま返る。
func (s *TodoService) FindTodo(ctx context.Context, id todo.ID) (*FindTodoOutput, error) {
	t, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return &FindTodoOutput{
		ID:          t.ID(),
		Title:       t.Title(),
		DueDate:     t.DueDate(),
		Priority:    t.Priority(),
		Status:      t.Status(),
		CompletedAt: t.CompletedAt(),
	}, nil
}

// ============================================================
// CompleteTodo: Todoを完了状態にする
// ============================================================

// CompleteTodo は指定IDのTodoを完了状態にする。
func (s *TodoService) CompleteTodo(ctx context.Context, id todo.ID) error {
	t, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	t.Complete(s.clock.Now())
	return s.repo.Save(ctx, t)
}

// ============================================================
// DeleteTodo: Todoを削除する
// ============================================================

// DeleteTodo は指定IDのTodoを削除する。
func (s *TodoService) DeleteTodo(ctx context.Context, id todo.ID) error {
	return s.repo.Delete(ctx, id)
}
