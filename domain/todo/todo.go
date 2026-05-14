package todo

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

// 値オブジェクト
type ID string

// NewIDは新しいIDを生成する
func NewID() ID {
	return ID(uuid.NewString())
}

// StringはID文字列表現を返す
func (id ID) String() string {
	return string(id)
}

// StatusはTodoの状態。未完了or完了
type Status string

const (
	StatusPending   Status = "pending"
	StatusCompleted Status = "completed"
)

// 優先度の定義
type Priority int

const (
	PriorityLow Priority = iota + 1
	PriorityMedium
	PriorityHigh
)

// 優先度のバリデーション
func (p Priority) IsValid() bool {
	return p >= PriorityLow && p <= PriorityHigh
}

// Stringは優先度の文字列表現を返す
func (p Priority) String() string {
	switch p {
	case PriorityLow:
		return "low"
	case PriorityMedium:
		return "medium"
	case PriorityHigh:
		return "high"
	default:
		return "unknown"
	}
}

// ドメインエラー

var (
	ErrTitleEmpty      = errors.New("タイトルは必須です")
	ErrTitleTooLong    = errors.New("タイトルは200文字以内である必要があります")
	ErrInvalidPriority = errors.New("優先度の値が不正です")
	ErrAlreadyPending  = errors.New("既に未完了状態です")
)

const maxTitleLength = 200

// Todoエンティティ
// フィールドは全て隠蔽する

type Todo struct {
	id          ID
	title       string
	dueDate     *time.Time
	priority    Priority
	status      Status
	completedAt *time.Time
}

// 新しいTodoを作成する
// バリデーションを一元化する
func New(title string, dueDate *time.Time, priority Priority) (*Todo, error) {
	if strings.TrimSpace(title) == "" {
		return nil, ErrTitleEmpty
	}
	if len(title) > maxTitleLength {
		return nil, ErrTitleTooLong
	}
	if !priority.IsValid() {
		return nil, ErrInvalidPriority
	}
	return &Todo{
		id:       NewID(),
		title:    title,
		dueDate:  dueDate,
		priority: priority,
		status:   StatusPending, // 作成時は必ず未完了
	}, nil
}

// Reconstruct は永続化層からのドメインオブジェクト復元用。
// リポジトリ実装からのみ呼び出すことを意図しており、バリデーションは行わない
// （DBから取り出した値は既にバリデーション済みである前提）。
func Reconstruct(
	id ID,
	title string,
	dueDate *time.Time,
	priority Priority,
	status Status,
	completedAt *time.Time,
) *Todo {
	return &Todo{
		id:          id,
		title:       title,
		dueDate:     dueDate,
		priority:    priority,
		status:      status,
		completedAt: completedAt,
	}
}

// ============================================================
// 状態を変える振る舞い（業務ルールの本体）
// ============================================================

// Complete はTodoを完了状態にする。冪等。
// 完了済みのTodoに対して呼んでも完了日時は変わらない。
func (t *Todo) Complete(now time.Time) {
	if t.status == StatusCompleted {
		return // 冪等: 何もしない
	}
	t.status = StatusCompleted
	t.completedAt = &now
}

// Reopen は完了済みのTodoを未完了に戻す。
// 未完了のTodoに対してはエラーを返す。
func (t *Todo) Reopen() error {
	if t.status == StatusPending {
		return ErrAlreadyPending
	}
	t.status = StatusPending
	t.completedAt = nil
	return nil
}

// ChangeTitle はタイトルを変更する。バリデーションはNewと同じ。
func (t *Todo) ChangeTitle(newTitle string) error {
	if strings.TrimSpace(newTitle) == "" {
		return ErrTitleEmpty
	}
	if len(newTitle) > maxTitleLength {
		return ErrTitleTooLong
	}
	t.title = newTitle
	return nil
}

// ChangePriority は優先度を変更する。
func (t *Todo) ChangePriority(p Priority) error {
	if !p.IsValid() {
		return ErrInvalidPriority
	}
	t.priority = p
	return nil
}

// ChangeDueDate は期限を変更する。nilを渡せば「期限なし」にできる。
func (t *Todo) ChangeDueDate(due *time.Time) {
	t.dueDate = due
}

// ============================================================
// 問い合わせ系（状態は変えない）
// ============================================================

// IsOverdue は期限切れかを返す。
// 「期限あり、かつ未完了、かつ現在時刻が期限を過ぎている」とき true。
func (t *Todo) IsOverdue(now time.Time) bool {
	if t.dueDate == nil {
		return false
	}
	if t.status == StatusCompleted {
		return false
	}
	return now.After(*t.dueDate)
}

// ============================================================
// ゲッター（外部からは読み取り専用）
// ============================================================

func (t *Todo) ID() ID                  { return t.id }
func (t *Todo) Title() string           { return t.title }
func (t *Todo) DueDate() *time.Time     { return t.dueDate }
func (t *Todo) Priority() Priority      { return t.priority }
func (t *Todo) Status() Status          { return t.status }
func (t *Todo) CompletedAt() *time.Time { return t.completedAt }
