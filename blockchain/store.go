
package blockchain

import (
  "path/filepath"
)

type Store struct {
  Path string `yaml:"store_path"`
  TaskToolsCmd string `yaml:"task_tools_cmd"`
}

func (store *Store) taskToolsPath(taskBlockHash string) string {
  return filepath.Join(store.Path, taskBlockHash, store.TaskToolsCmd)
}

func (store *Store) blockDir(hash string) string {
  return filepath.Join(store.Path, hash)
}
