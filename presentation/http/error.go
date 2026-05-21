package httpx

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/kkwinter73/todo_cleanarc/domain/todo"
)

// writeJSON は任意の値をJSONで返す共通関数。
func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

// writeError は エラーをHTTPレスポンスに変換して返す。
//
// この関数がプレゼン層の「翻訳」の本質:
//   - ドメインのバリデーションエラー → 400 Bad Request
//   - ドメインの「見つからない」エラー → 404 Not Found
//   - それ以外 → 500 Internal Server Error
//
// アプリ層やドメイン層はHTTPステータスコードを知らない。
// 「業務上のエラー」を「HTTPの言葉」に翻訳するのがプレゼン層の責務。
//
// 重要: ここで参照しているのは全て domain/todo のエラー。
// インフラ層の具体的な実装は知らない(プレゼン層はインフラ層に依存しない)。
func writeError(w http.ResponseWriter, err error) {
	status := mapErrorToStatus(err)
	writeJSON(w, status, ErrorResponse{Error: err.Error()})
}

// mapErrorToStatus はエラーの種類からHTTPステータスコードを決定する。
func mapErrorToStatus(err error) int {
	switch {
	// ドメインのバリデーションエラー → 400 Bad Request
	case errors.Is(err, todo.ErrTitleEmpty),
		errors.Is(err, todo.ErrTitleTooLong),
		errors.Is(err, todo.ErrInvalidPriority),
		errors.Is(err, todo.ErrAlreadyPending):
		return http.StatusBadRequest

	// ドメインの「見つからない」エラー → 404 Not Found
	// (インフラ実装が PostgreSQL でも InMemory でも、同じこのエラーを返す)
	case errors.Is(err, todo.ErrTodoNotFound):
		return http.StatusNotFound

	// それ以外 → 500 Internal Server Error
	default:
		return http.StatusInternalServerError
	}
}

// writeBadRequest はリクエストパース時のエラーを 400 で返す簡易ヘルパー。
func writeBadRequest(w http.ResponseWriter, message string) {
	writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: message})
}
