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
// Clock: 時刻取得を抽象化する
// ============================================================

// Clock は現在時刻を取得するインターフェース。
//
// アプリケーション層から time.Now() を直接呼ばずに、
// このインターフェース経由で時刻を取得する。
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
//
// 段取り:
//  1. ドメインのNewでTodoを生成(バリデーションはドメインが行う)
//  2. リポジトリで保存する
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
// CompleteTodo: Todoを完了状態にする
// ============================================================

// CompleteTodo は指定IDのTodoを完了状態にする。
//
// 段取り:
//  1. リポジトリからTodoを取得
//  2. ドメインのComplete()を呼ぶ(冪等・業務ルールはドメインが守る)
//  3. リポジトリに保存
//
// 時刻は Clock 経由で取得し、ドメインに引数として渡す。
// (ドメイン層のメソッドは time.Now() を呼ばず、時刻を引数で受け取る設計)
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
//
// 段取り:
//  1. リポジトリのDelete()を呼ぶ
//
// シンプルなのでドメイン呼び出しはない。
// もし将来「削除には所有者チェックが必要」などの業務ルールが増えたら、
// その時点で FindByID -> ドメインルール検証 -> Delete のフローに変える。
func (s *TodoService) DeleteTodo(ctx context.Context, id todo.ID) error {
	return s.repo.Delete(ctx, id)
}
