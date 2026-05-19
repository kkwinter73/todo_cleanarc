package httpx

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// NewRouter はTodoのHTTPルーターを構築する。
//
// chi の Router は標準ライブラリの http.Handler インターフェースを満たすので、
// 受け取った側は通常の http.Server.ListenAndServe(...) に渡せる。
//
// ミドルウェア:
//   - Logger     : アクセスログを出す
//   - Recoverer  : ハンドラ内でpanicが起きてもサーバーを落とさず500を返す
func NewRouter(h *TodoHandler) http.Handler {
	r := chi.NewRouter()

	// ミドルウェア(全リクエストに適用される処理)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// /todos リソースに対するルーティング
	r.Route("/todos", func(r chi.Router) {
		r.Post("/", h.Create)                // POST   /todos
		r.Get("/{id}", h.Find)               // GET    /todos/{id}
		r.Post("/{id}/complete", h.Complete) // POST   /todos/{id}/complete
		r.Delete("/{id}", h.Delete)          // DELETE /todos/{id}
	})

	// ヘルスチェック(動作確認用)
	r.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	return r
}
