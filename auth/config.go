
package auth

type Config struct {
  ClientID string `yaml:"client_id"`
  ClientSecret string `yaml:"client_secret"`
  RedirectURL string `yaml:"redirect_url"`
  AuthURL string `yaml:"auth_url"`
  TokenURL string `yaml:"token_url"`
  ProfileURL string `yaml:"profile_url"`
  LogoutURL string `yaml:"logout_url"`
  FrontendOrigin string
}
