// Package httpx はHTTPプレゼンテーション層を提供する。
//
// パッケージ名を "httpx" にしている理由:
//   - 標準ライブラリの "net/http" と名前衝突しないようにするため
//   - "http" という名前は予約語的に避けるのが慣習
//
// この層の役割は「HTTPの世界とアプリ層の世界の翻訳」だけ。
// 業務ルール、SQL、ユースケースの段取りは書かない。
package httpx

import "time"

// ============================================================
// リクエストDTO
// ============================================================

// CreateTodoRequest は POST /todos のリクエストボディ。
//
// JSON タグでマッピングを明示する。
// DueDate を string にしているのは、JSONには日時型がなく文字列(RFC3339形式)で送られてくるため。
// アプリ層に渡す前に *time.Time に変換する。
type CreateTodoRequest struct {
	Title    string  `json:"title"`
	DueDate  *string `json:"due_date,omitempty"` // RFC3339形式の文字列
	Priority int     `json:"priority"`           // 1=low, 2=medium, 3=high
}

// ============================================================
// レスポンスDTO
// ============================================================

// TodoResponse はTodoの情報を返す共通レスポンス。
//
// 注: アプリ層の FindTodoOutput と似ているが別の型として定義している。
// 理由:
//   - JSONタグでフィールド名を明示できる(snake_case)
//   - 日時は文字列(RFC3339)に変換して返す
//   - HTTPの形式が変わってもアプリ層は影響を受けない
type TodoResponse struct {
	ID          string  `json:"id"`
	Title       string  `json:"title"`
	DueDate     *string `json:"due_date,omitempty"` // RFC3339形式
	Priority    int     `json:"priority"`
	Status      string  `json:"status"`
	CompletedAt *string `json:"completed_at,omitempty"` // RFC3339形式
}

// CreatedResponse は POST /todos のレスポンス。
// 作成されたIDだけ返す簡易な形式。
type CreatedResponse struct {
	ID string `json:"id"`
}

// ErrorResponse はエラー時のレスポンスボディ。
// 全エンドポイント共通の形式。
type ErrorResponse struct {
	Error string `json:"error"`
}

// ============================================================
// 時刻の変換ヘルパー
// ============================================================

// formatTime は *time.Time を *string(RFC3339形式) に変換する。
// nil なら nil を返す(JSON で省略される)。
func formatTime(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := t.Format(time.RFC3339)
	return &s
}

// parseTime は *string(RFC3339形式) を *time.Time に変換する。
// nil なら nil を返す。
// パースエラーなら error を返す。
func parseTime(s *string) (*time.Time, error) {
	if s == nil || *s == "" {
		return nil, nil
	}
	t, err := time.Parse(time.RFC3339, *s)
	if err != nil {
		return nil, err
	}
	return &t, nil
}
