
package config

type Config struct {
  Listen string `yaml:"listen"`
  MountPath string `yaml:"mount_path"`
  SelfUrl string `yaml:"self_url"`
  SessionSecret string `yaml:"session_secret"`
  CsrfSecret string `yaml:"csrf_secret"`
  DataSource string `yaml:"datasource"`
  FrontendOrigin string `yaml:"frontend_origin"`
  ApiVersion string `yaml:"api_version"`
  ApiKey string `yaml:"api_key"`
  Auth AuthConfig `yaml:"auth"`
  Blocks BlocksConfig `yaml:"blocks"`
}

type AuthConfig struct {
  ClientID string `yaml:"client_id"`
  ClientSecret string `yaml:"client_secret"`
  RedirectURL string `yaml:"redirect_url"`
  AuthURL string `yaml:"auth_url"`
  TokenURL string `yaml:"token_url"`
  ProfileURL string `yaml:"profile_url"`
  LogoutURL string `yaml:"logout_url"`
  FrontendOrigin string
}

type BlocksConfig struct {
  Path string `yaml:"store_path"`
  TaskToolsCmd string `yaml:"task_tools_cmd"`
  TaskHelperCmd string `yaml:"task_helper_cmd"`
  SkipDelete bool `yaml:"skip_delete"`
}
