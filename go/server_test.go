package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/sessions"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

func mockupServer(port int, responseCode int, responseHeaders map[string]string) *httptest.Server {
	mock := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/oauth2") || strings.HasPrefix(r.URL.Path, "/webauthn") {
			for key, value := range responseHeaders {
				w.Header().Add(key, value)
			}
			w.WriteHeader(responseCode)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))
	AuthenticationServer, _ := net.Listen("tcp", fmt.Sprintf(":%d", port))
	mock.Listener = AuthenticationServer
	mock.Start()
	return mock
}

func TestConstructorNewExternalAuthServices(t *testing.T) {
	t.Setenv("AUTHENTIFICATION_SERVICE_HOST", "oauth2.default.svc.cluster.local")
	t.Setenv("AUTHORISATION_SERVICE_HOST", "webauthn.default.svc.cluster.local")

	eas := NewExternalAuthServices()

	assert.Equal(t, eas.AuthentificationServiceHost, "oauth2.default.svc.cluster.local")
	assert.Equal(t, eas.AuthorisationServiceHost, "webauthn.default.svc.cluster.local")
}

func TestAuthHandlerUserIsValid(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/auth", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// https://github.com/labstack/echo/issues/1447
	c.Set("_session_store", sessions.NewCookieStore([]byte("secret")))

	sess, err := session.Get("session", c)
	if err != nil {
		log.Println("Error obtaining the session")
		log.Println(err)

	}

	sess.Values["session"] = UserSession{
		Valid:  true,
		UserId: "t3stus3r",
	}

	sess.Save(c.Request(), c.Response())

	if assert.NoError(t, AuthHandler(c)) {
		assert.Equal(t, http.StatusAccepted, rec.Code)
		assert.Equal(t, "Hello, t3stus3r", string(rec.Body.String()))
	}
}

func TestAuthHandlerUserIsNotValid(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/auth", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// https://github.com/labstack/echo/issues/1447
	c.Set("_session_store", sessions.NewCookieStore([]byte("secret")))

	sess, err := session.Get("session", c)
	if err != nil {
		log.Println("Error obtaining the session")
		log.Println(err)
	}

	sess.Values["session"] = UserSession{
		Valid:  false,
		UserId: "",
	}

	sess.Save(c.Request(), c.Response())

	if assert.NoError(t, AuthHandler(c)) {
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	}
}

func TestAuthHandlerSessionDoesNotExist(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/auth", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// https://github.com/labstack/echo/issues/1447
	c.Set("_session_store", sessions.NewCookieStore([]byte("secret")))

	if assert.NoError(t, AuthHandler(c)) {
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	}
}

func TestRegisterHandlerUserIsAuthentificatedAndAuthorised(t *testing.T) {
	AuthentificationServer := mockupServer(9090, http.StatusAccepted, map[string]string{
		"X-Auth-Request-User": "T3stUs3r",
	})
	defer AuthentificationServer.Close()

	AuthorisationServer := mockupServer(9091, http.StatusOK, nil)
	defer AuthorisationServer.Close()

	eas := ExternalAuthServices{
		AuthentificationClient:      AuthentificationServer.Client(),
		AuthentificationServiceHost: "127.0.0.1:9090",
		AuthorisationClient:         AuthorisationServer.Client(),
		AuthorisationServiceHost:    "127.0.0.1:9091",
	}

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/register", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	// https://github.com/labstack/echo/issues/1447
	c.Set("_session_store", sessions.NewCookieStore([]byte("secret")))

	if assert.NoError(t, eas.RegisterHandler(c)) {
		assert.Equal(t, http.StatusAccepted, rec.Code)
		assert.Equal(t, "Hello, T3stUs3r", string(rec.Body.String()))
		assert.Equal(t, "T3stUs3r", rec.Header().Get("X-Auth-Request-User"))
	}
}

func TestRegisterHandlerUserIsNotAuthentificated(t *testing.T) {
	AuthentificationServer := mockupServer(9090, http.StatusUnauthorized, nil)
	defer AuthentificationServer.Close()

	AuthorisationServer := mockupServer(9091, http.StatusUnauthorized, nil)
	defer AuthorisationServer.Close()

	eas := ExternalAuthServices{
		AuthentificationClient:      AuthentificationServer.Client(),
		AuthentificationServiceHost: "127.0.0.1:9090",
		AuthorisationClient:         AuthorisationServer.Client(),
		AuthorisationServiceHost:    "127.0.0.1:9091",
	}

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "http://localhost/register", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if assert.NoError(t, eas.RegisterHandler(c)) {
		assert.Equal(t, http.StatusTemporaryRedirect, rec.Code)
		expectedUrl := "http://localhost/oauth2/sign_in"
		actualUrl, _ := rec.Result().Location()
		assert.Equal(t, expectedUrl, actualUrl.String())
	}
}

func TestRegisterHandlerUserIsNotAuthorised(t *testing.T) {
	AuthentificationServer := mockupServer(9090, http.StatusAccepted, map[string]string{
		"X-Auth-Request-User": "T3stUs3r",
	})
	defer AuthentificationServer.Close()

	AuthorisationServer := mockupServer(9091, http.StatusUnauthorized, nil)
	defer AuthorisationServer.Close()

	eas := ExternalAuthServices{
		AuthentificationClient:      AuthentificationServer.Client(),
		AuthentificationServiceHost: "127.0.0.1:9090",
		AuthorisationClient:         AuthorisationServer.Client(),
		AuthorisationServiceHost:    "127.0.0.1:9091",
	}

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "http://localhost/register", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if assert.NoError(t, eas.RegisterHandler(c)) {
		assert.Equal(t, http.StatusTemporaryRedirect, rec.Code)
		expectedUrl := "http://localhost/webauthn/login?redirect_url=http://localhost&default_username=T3stUs3r"
		actualUrl, _ := rec.Result().Location()
		assert.Equal(t, expectedUrl, actualUrl.String())
	}
}
