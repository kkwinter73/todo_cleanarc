package main

import "time"

// タスク追加
func (tl *TodoList) Add(title string, explanation string, p int, DeadLine time.Time) {
	todo := &Todo{
		Title:       title,
		Explanation: explanation,
		Priority: priority{
			p: 0,
		},
		IsDone:     false,
		Created_At: time.Now(),
		Deadline:   DeadLine,
	}

	// スライス（データベース）にアクセスして、要素を追加する
	*tl = append(*tl, todo)
}

// タスク一覧を見る
func (tl TodoList) List() TodoList {
	return tl
}

// タスク削除
func (tl *TodoList) Delete(taskid int) {
	*tl = append((*tl)[:taskid], (*tl)[taskid+1:]...)
}
