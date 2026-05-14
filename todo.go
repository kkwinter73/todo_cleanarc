package main

import (
	"errors"
	"fmt"
	"time"
)

// 構造体の定義（ TODO そのもの）
type Todo struct {
	ID        int       `json:"id"`
	Title     string    `json:"title"`
	Done      bool      `json:"done"`
	CreatedAt time.Time `json:"created_at"`
}

// 構造体の定義　（ 一覧表示したり、タスクを追加したりするようのもの）
type TodoList struct {
	Todos  []Todo `json:"Todos"`
	NextID int    `json:"next_id"`
}

// 初期化関数（メソッドではない）　レシーバないでしょ
func NewTodoList() *TodoList {
	return &TodoList{
		Todos:  []Todo{}, // 空スライス nil だとjson変換時に "null"になってしまう（型がなくなる）
		NextID: 1,
	}
}

// タスク追加用
func (tl *TodoList) Add(title string) Todo {
	todo := Todo{
		ID:        tl.NextID,
		Title:     title,
		Done:      false,
		CreatedAt: time.Now(),
	}

	tl.Todos = append(tl.Todos, todo)
	tl.NextID++
	return todo
}

// 一覧表示用
func (tl *TodoList) List() []Todo {
	return tl.Todos
}

// タスク完了反映用
func (tl *TodoList) Done(id int) error {
	for i := range tl.Todos {
		if tl.Todos[i].ID == id {
			if tl.Todos[i].Done {
				return fmt.Errorf("タスク %d は既に完了しています", id)
			}
			tl.Todos[i].Done = true
			return nil
		}
	}
	return errors.New("指定されたIDのタスクがありません")
}

// タスク削除用
func (tl *TodoList) Delete(id int) error {
	for i := range tl.Todos {
		if tl.Todos[i].ID == id {
			tl.Todos = append(tl.Todos[:i], tl.Todos[i+1:]...)
			return nil
		}
	}

	return errors.New("指定されたIDのタスクがありません")
}
