
package auth

import (
  "html/template"
)

type loginCompleteData struct {
  Message string
  Target string
}

type logoutCompleteData struct {
  Message string
  Target string
  LogoutUrl string
}

type loginErrorData struct {
  Error string
  Message string
}

func SetupTemplates(t *template.Template) {
  template.Must(t.New("loginComplete").Parse(`<!DOCTYPE html>
<head lang="en"><meta charset="utf-8"><title>login complete</title></head>
<body><script type="text/javascript">
  window.opener.postMessage({{.Message}}, {{.Target}});
  window.close();
</script></body>`))
  template.Must(t.New("logoutComplete").Parse(`<!DOCTYPE html>
<head lang="en"><meta charset="utf-8"><title>logout complete</title></head>
<body><script type="text/javascript">
  window.opener.postMessage({{.Message}}, {{.Target}});
  window.location.href = {{.LogoutUrl}};
</script></body>`))
  template.Must(t.New("loginError").Parse(`<!DOCTYPE html>
<head lang="en"><meta charset="utf-8"><title>Authentication Failed</title></head>
<body><h1>{{.Error}}</h1><p>{{.Message}}</p></body>`))
  template.Must(t.New("noSession").Parse(`<!DOCTYPE html>
<head lang="en"><meta charset="utf-8"><title>no session</title></head>
<body><p>No session found, please try again.</p></body>`))
}
