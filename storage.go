package main

import (
	"encoding/json"
	"errors"
	"os"
)

const defaultFilePath = "todos.json"

func Save(tl *TodoList, filePath string) error {
	data, err := json.MarshalIndent(tl, "", " ")
	if err != nil {
		return err
	}

	return os.WriteFile(filePath, data, 0644)
}

func Load(filePath string) (*TodoList, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return NewTodoList(), nil
		}
		return nil, err
	}

	var tl TodoList
	if err := json.Unmarshal(data, &tl); err != nil {
		return nil, err
	}

	return &tl, nil
}
