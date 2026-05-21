// プレゼンテーション層(HTTPハンドラ)のテスト。
package httpx_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/kkwinter73/todo_cleanarc/application"
	"github.com/kkwinter73/todo_cleanarc/domain/todo"
	httpx "github.com/kkwinter73/todo_cleanarc/presentation/http"
)

// ============================================================
// テストヘルパー: InMemoryモック + fakeClock + setupServer
// ============================================================

type inMemoryTodoRepository struct {
	mu    sync.Mutex
	store map[todo.ID]*todo.Todo
}

var _ todo.Repository = (*inMemoryTodoRepository)(nil)

func newInMemoryTodoRepository() *inMemoryTodoRepository {
	return &inMemoryTodoRepository{store: make(map[todo.ID]*todo.Todo)}
}

func (r *inMemoryTodoRepository) Save(ctx context.Context, t *todo.Todo) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.store[t.ID()] = t
	return nil
}

// FindByID: 見つからないときは「ドメインの」 ErrTodoNotFound を返す。
// インフラ実装(PostgreSQL版もInMemory版も)は、必ず同じエラーを返すべき。
// これによりプレゼン層がインフラ実装を知らなくてもエラーを判定できる。
func (r *inMemoryTodoRepository) FindByID(ctx context.Context, id todo.ID) (*todo.Todo, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	t, ok := r.store[id]
	if !ok {
		return nil, todo.ErrTodoNotFound
	}
	return t, nil
}

func (r *inMemoryTodoRepository) Delete(ctx context.Context, id todo.ID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.store, id)
	return nil
}

type fakeClock struct{ fixed time.Time }

func (c fakeClock) Now() time.Time { return c.fixed }

func setupTest(t *testing.T) (http.Handler, *inMemoryTodoRepository) {
	t.Helper()
	repo := newInMemoryTodoRepository()
	clock := fakeClock{fixed: time.Date(2026, 5, 19, 10, 0, 0, 0, time.UTC)}
	service := application.NewTodoService(repo, clock)
	handler := httpx.NewTodoHandler(service)
	router := httpx.NewRouter(handler)
	return router, repo
}

// ============================================================
// POST /todos のテスト
// ============================================================

func TestTodoHandler_Create_Success(t *testing.T) {
	router, repo := setupTest(t)

	body := []byte(`{
		"title": "牛乳を買う",
		"due_date": "2026-12-31T23:59:00Z",
		"priority": 2
	}`)
	req := httptest.NewRequest(http.MethodPost, "/todos", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("status = %d, want %d (body: %s)", rec.Code, http.StatusCreated, rec.Body.String())
	}

	var resp httpx.CreatedResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("レスポンスJSONパース失敗: %v", err)
	}
	if resp.ID == "" {
		t.Error("response.id が空")
	}

	if _, err := repo.FindByID(context.Background(), todo.ID(resp.ID)); err != nil {
		t.Errorf("保存されていない: %v", err)
	}
}

func TestTodoHandler_Create_InvalidJSON(t *testing.T) {
	router, _ := setupTest(t)

	body := []byte(`{this is not valid json`)
	req := httptest.NewRequest(http.MethodPost, "/todos", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestTodoHandler_Create_DomainValidationError(t *testing.T) {
	router, _ := setupTest(t)

	body := []byte(`{"title": "", "priority": 2}`)
	req := httptest.NewRequest(http.MethodPost, "/todos", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d (body: %s)", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
}

func TestTodoHandler_Create_InvalidDateFormat(t *testing.T) {
	router, _ := setupTest(t)

	body := []byte(`{"title":"タスク","due_date":"2026/12/31","priority":2}`)
	req := httptest.NewRequest(http.MethodPost, "/todos", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d (body: %s)", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
}

// ============================================================
// GET /todos/{id} のテスト
// ============================================================

func TestTodoHandler_Find_Success(t *testing.T) {
	router, repo := setupTest(t)

	td, _ := todo.New("タスク", nil, todo.PriorityMedium)
	repo.Save(context.Background(), td)

	req := httptest.NewRequest(http.MethodGet, "/todos/"+td.ID().String(), nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var resp httpx.TodoResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("JSONパース失敗: %v", err)
	}
	if resp.ID != td.ID().String() {
		t.Errorf("id = %q, want %q", resp.ID, td.ID().String())
	}
	if resp.Title != "タスク" {
		t.Errorf("title = %q, want %q", resp.Title, "タスク")
	}
	if resp.Status != "pending" {
		t.Errorf("status = %q, want %q", resp.Status, "pending")
	}
}

func TestTodoHandler_Find_NotFound(t *testing.T) {
	router, _ := setupTest(t)

	req := httptest.NewRequest(http.MethodGet, "/todos/"+todo.NewID().String(), nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

// ============================================================
// POST /todos/{id}/complete のテスト
// ============================================================

func TestTodoHandler_Complete_Success(t *testing.T) {
	router, repo := setupTest(t)

	td, _ := todo.New("タスク", nil, todo.PriorityMedium)
	repo.Save(context.Background(), td)

	req := httptest.NewRequest(http.MethodPost, "/todos/"+td.ID().String()+"/complete", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var resp httpx.TodoResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("JSONパース失敗: %v", err)
	}
	if resp.Status != "completed" {
		t.Errorf("status = %q, want %q", resp.Status, "completed")
	}
	if resp.CompletedAt == nil {
		t.Error("completed_at が nil")
	}
}

func TestTodoHandler_Complete_NotFound(t *testing.T) {
	router, _ := setupTest(t)

	req := httptest.NewRequest(http.MethodPost, "/todos/"+todo.NewID().String()+"/complete", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

// ============================================================
// DELETE /todos/{id} のテスト
// ============================================================

func TestTodoHandler_Delete_Success(t *testing.T) {
	router, repo := setupTest(t)

	td, _ := todo.New("タスク", nil, todo.PriorityMedium)
	repo.Save(context.Background(), td)

	req := httptest.NewRequest(http.MethodDelete, "/todos/"+td.ID().String(), nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}

	if _, err := repo.FindByID(context.Background(), td.ID()); err == nil {
		t.Error("削除されていない")
	}
}

func TestTodoHandler_Delete_NotFoundIdempotent(t *testing.T) {
	router, _ := setupTest(t)

	req := httptest.NewRequest(http.MethodDelete, "/todos/"+todo.NewID().String(), nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
}

// ============================================================
// ヘルスチェック
// ============================================================

func TestHealthCheck(t *testing.T) {
	router, _ := setupTest(t)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if rec.Body.String() != "OK" {
		t.Errorf("body = %q, want %q", rec.Body.String(), "OK")
	}
}
