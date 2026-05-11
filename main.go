package main

import (
	"fmt"
	"os"
	"strconv"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	todoList, err := Load(defaultFilePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "データの読み込みに失敗しました: %v\n", err)
		os.Exit(1)
	}

	switch os.Args[1] {
	case "add":
		err = runAdd(todoList, os.Args[2:])
	case "list":
		runList(todoList)
		return
	case "done":
		err = runDone(todoList, os.Args[2:])
	case "delete":
		err = runDelete(todoList, os.Args[2:])
	default:
		fmt.Fprintf(os.Stderr, "不明なコマンド: %s\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "エラー: %v\n", err)
		os.Exit(1)
	}

	if err := Save(todoList, defaultFilePath); err != nil {
		fmt.Fprintf(os.Stderr, "データの保存に失敗しました: %v\n", err)
		os.Exit(1)
	}
}

func runAdd(tl *TodoList, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("タスク名を指定してください")
	}
	todo := tl.Add(args[0])
	fmt.Printf("✅ タスクを追加しました: [%d] %s\n", todo.ID, todo.Title)
	return nil
}

func runList(tl *TodoList) {
	todos := tl.List()
	if len(todos) == 0 {
		fmt.Println("📋 タスクはありません")
		return
	}
	fmt.Println("📋 タスク一覧:")
	fmt.Println("─────────────────────────────────────")
	for _, t := range todos {
		status := "⬜"
		if t.Done {
			status = "✅"
		}
		fmt.Printf("  %s [%d] %s  (%s)\n",
			status, t.ID, t.Title,
			t.CreatedAt.Format("2006-01-02 15:04"),
		)
	}
	fmt.Println("─────────────────────────────────────")
	fmt.Printf("  合計: %d 件\n", len(todos))
}

func runDone(tl *TodoList, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("タスクIDを指定してください")
	}
	id, err := strconv.Atoi(args[0])
	if err != nil {
		return fmt.Errorf("IDは数値で指定してください: %s", args[0])
	}
	if err := tl.Done(id); err != nil {
		return err
	}
	fmt.Printf("✅ タスク %d を完了にしました\n", id)
	return nil
}

func runDelete(tl *TodoList, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("タスクIDを指定してください")
	}
	id, err := strconv.Atoi(args[0])
	if err != nil {
		return fmt.Errorf("IDは数値で指定してください: %s", args[0])
	}
	if err := tl.Delete(id); err != nil {
		return err
	}
	fmt.Printf("🗑️  タスク %d を削除しました\n", id)
	return nil
}

func printUsage() {
	fmt.Println(`
📝 Go TODO CLI

使い方:
  go run . <command> [arguments]

コマンド:
  add <title>    タスクを追加する
  list           タスク一覧を表示する
  done <id>      タスクを完了にする
  delete <id>    タスクを削除する`)
}
