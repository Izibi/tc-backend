
package game

type Config struct {
  ApiVersion string `yaml:"api_version"`
  BlockStorePath string `yaml:"block_store_path"`
  TaskToolsBin string `yaml:"task_tools_bin"`
  TaskPath string `yaml:"task_path"`
}
