package git

import (
	"fmt"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"net/http"
	"net/url"
)

type AuthMethod interface {
	toGitAuthMethod() transport.AuthMethod
}

func UrlTokenAuthMethod(token string) AuthMethod {
	return &urlTokenAuth{token: token}
}

type urlTokenAuth struct {
	token string
}

func (a *urlTokenAuth) toGitAuthMethod() transport.AuthMethod {
	return a
}

func (a *urlTokenAuth) SetAuth(r *http.Request) {
	if a == nil {
		return
	}
	r.URL.User = url.User(a.token)
}

func (a *urlTokenAuth) Name() string {
	return "url-token-auth"
}

func (a *urlTokenAuth) String() string {
	return fmt.Sprintf("%s - masked-token", a.Name())
}
