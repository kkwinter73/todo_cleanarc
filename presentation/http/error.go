package httpx

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/kkwinter73/todo_cleanarc/domain/todo"
	"github.com/kkwinter73/todo_cleanarc/infrastructure/persistence"
)

// writeJSON は任意の値をJSONで返す共通関数。
func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	// エンコードエラーは実用上ほぼ起きないので簡略化
	_ = json.NewEncoder(w).Encode(body)
}

// writeError は エラーをHTTPレスポンスに変換して返す。
//
// この関数がプレゼン層の「翻訳」の本質:
//   - ドメインのバリデーションエラー → 400 Bad Request
//   - リポジトリの「見つからない」エラー → 404 Not Found
//   - それ以外 → 500 Internal Server Error
//
// アプリ層やドメイン層はHTTPステータスコードを知らない。
// 「業務上のエラー」を「HTTPの言葉」に翻訳するのがプレゼン層の責務。
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

	// リポジトリの「見つからない」エラー → 404 Not Found
	case errors.Is(err, persistence.ErrTodoNotFound):
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
