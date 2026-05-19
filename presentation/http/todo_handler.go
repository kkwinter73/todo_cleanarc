package httpx

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/kkwinter73/todo_cleanarc/application"
	"github.com/kkwinter73/todo_cleanarc/domain/todo"
)

// TodoHandler はTodoに関するHTTPハンドラ。
//
// 依存はアプリケーション層のみ。
// インフラ層やドメイン層の具体型は(エラー判定を除き)直接呼ばない。
type TodoHandler struct {
	service *application.TodoService
}

// NewTodoHandler はTodoHandlerのコンストラクタ。
func NewTodoHandler(service *application.TodoService) *TodoHandler {
	return &TodoHandler{service: service}
}

// ============================================================
// POST /todos: 新規作成
// ============================================================

// 段取り:
//  1. リクエストボディ(JSON)をパース
//  2. ドメインの型に変換
//  3. アプリ層のCreateTodoを呼ぶ
//  4. 結果をJSONで返す(201 Created)
func (h *TodoHandler) Create(w http.ResponseWriter, r *http.Request) {
	// 1. リクエストパース
	var req CreateTodoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeBadRequest(w, "リクエストの形式が不正です: "+err.Error())
		return
	}

	// 2. 日時の文字列を *time.Time に変換
	due, err := parseTime(req.DueDate)
	if err != nil {
		writeBadRequest(w, "due_date は RFC3339 形式で指定してください")
		return
	}

	// 3. アプリ層呼び出し
	out, err := h.service.CreateTodo(r.Context(), application.CreateTodoInput{
		Title:    req.Title,
		DueDate:  due,
		Priority: todo.Priority(req.Priority),
	})
	if err != nil {
		writeError(w, err)
		return
	}

	// 4. レスポンス生成(201 Created)
	writeJSON(w, http.StatusCreated, CreatedResponse{ID: out.ID.String()})
}

// ============================================================
// GET /todos/{id}: 1件取得
// ============================================================

func (h *TodoHandler) Find(w http.ResponseWriter, r *http.Request) {
	// chi.URLParam で URLパスから {id} を取り出す
	idStr := chi.URLParam(r, "id")
	if idStr == "" {
		writeBadRequest(w, "id は必須です")
		return
	}

	out, err := h.service.FindTodo(r.Context(), todo.ID(idStr))
	if err != nil {
		writeError(w, err)
		return
	}

	// Output → レスポンスDTOに変換
	resp := TodoResponse{
		ID:          out.ID.String(),
		Title:       out.Title,
		DueDate:     formatTime(out.DueDate),
		Priority:    int(out.Priority),
		Status:      string(out.Status),
		CompletedAt: formatTime(out.CompletedAt),
	}
	writeJSON(w, http.StatusOK, resp)
}

// ============================================================
// POST /todos/{id}/complete: 完了
// ============================================================

func (h *TodoHandler) Complete(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	if idStr == "" {
		writeBadRequest(w, "id は必須です")
		return
	}

	if err := h.service.CompleteTodo(r.Context(), todo.ID(idStr)); err != nil {
		writeError(w, err)
		return
	}

	// 完了後の状態を取得して返す(UX的に有用)
	out, err := h.service.FindTodo(r.Context(), todo.ID(idStr))
	if err != nil {
		writeError(w, err)
		return
	}
	resp := TodoResponse{
		ID:          out.ID.String(),
		Title:       out.Title,
		DueDate:     formatTime(out.DueDate),
		Priority:    int(out.Priority),
		Status:      string(out.Status),
		CompletedAt: formatTime(out.CompletedAt),
	}
	writeJSON(w, http.StatusOK, resp)
}

// ============================================================
// DELETE /todos/{id}: 削除
// ============================================================

func (h *TodoHandler) Delete(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	if idStr == "" {
		writeBadRequest(w, "id は必須です")
		return
	}

	if err := h.service.DeleteTodo(r.Context(), todo.ID(idStr)); err != nil {
		writeError(w, err)
		return
	}

	// 204 No Content: 成功・本体なし
	w.WriteHeader(http.StatusNoContent)
}
