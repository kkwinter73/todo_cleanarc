// presentation/http/todo_handler.go
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
// 依存はアプリケーション層の「インターフェース」(application.TodoUsecase)のみ。
// 具象の *application.TodoService や、ましてやインフラ・ドメインの実装には依存しない。
type TodoHandler struct {
	service application.TodoUsecase
}

// NewTodoHandler はTodoHandlerのコンストラクタ。
// 引数はインターフェース型なので、本物のTodoServiceでもテスト用モックでも注入できる。
func NewTodoHandler(service application.TodoUsecase) *TodoHandler {
	return &TodoHandler{service: service}
}

// 以下のメソッド本体は元のコードのまま。
// service の型がインターフェースに変わっただけで、呼び出し側のコードは変更不要。

func (h *TodoHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateTodoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeBadRequest(w, "リクエストの形式が不正です: "+err.Error())
		return
	}

	due, err := parseTime(req.DueDate)
	if err != nil {
		writeBadRequest(w, "due_date は RFC3339 形式で指定してください")
		return
	}

	out, err := h.service.CreateTodo(r.Context(), application.CreateTodoInput{
		Title:    req.Title,
		DueDate:  due,
		Priority: todo.Priority(req.Priority),
	})
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, CreatedResponse{ID: out.ID.String()})
}

func (h *TodoHandler) Find(w http.ResponseWriter, r *http.Request) {
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

	w.WriteHeader(http.StatusNoContent)
}
