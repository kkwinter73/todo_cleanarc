package main

import (
	"time"
)

type Todo struct {
	Title       string
	Explanation string
	Priority    priority
	IsDone      bool
	Created_At  time.Time
	Deadline    time.Time
}

type TodoList []*Todo

// 優先度を数値として表す
type priority struct {
	p int
}
