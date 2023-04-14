package main

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

func mockupServer(port int, responseCode int, responseHeaders map[string]string) *httptest.Server {
	mock := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for key, value := range responseHeaders {
			w.Header().Add(key, value)
		}
		w.WriteHeader(responseCode)
	}))
	AuthenticationServer, _ := net.Listen("tcp", fmt.Sprintf(":%d", port))
	mock.Listener = AuthenticationServer
	mock.Start()
	return mock
}
func TestAuthHandlerUserIsAuthentificatedAndAuthorised(t *testing.T) {
	AuthentificationServer := mockupServer(9090, http.StatusUnauthorized, nil)
	defer AuthentificationServer.Close()

	AuthorisationServer := mockupServer(9091, http.StatusUnauthorized, nil)
	defer AuthorisationServer.Close()

	eas := ExternalAuthServices{
		AuthentificationClient:     AuthentificationServer.Client(),
		AuthentificationServiceUrl: "http://localhost:9090",
		AuthorisationClient:        AuthorisationServer.Client(),
		AuthorisationServiceUrl:    "http://localhost:9091",
		UserAuthentificated:        true,
		UserAuthorised:             true,
		UserId:                     "T3stUs3r",
	}

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/auth", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if assert.NoError(t, eas.AuthHandler(c)) {
		assert.Equal(t, http.StatusAccepted, rec.Code)
		assert.Equal(t, fmt.Sprintf("Hello, %s", eas.UserId), string(rec.Body.String()))
	}
}
func TestAuthHandlerUserIsAuthentificatedButNotAuthorised(t *testing.T) {
	AuthentificationServer := mockupServer(9090, http.StatusUnauthorized, nil)
	defer AuthentificationServer.Close()

	AuthorisationServer := mockupServer(9091, http.StatusUnauthorized, nil)
	defer AuthorisationServer.Close()

	eas := ExternalAuthServices{
		AuthentificationClient:     AuthentificationServer.Client(),
		AuthentificationServiceUrl: "http://localhost:9090",
		AuthorisationClient:        AuthorisationServer.Client(),
		AuthorisationServiceUrl:    "http://localhost:9091",
		UserAuthentificated:        true,
		UserAuthorised:             false,
		UserId:                     "T3stUs3r",
	}

	// Setup
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/auth", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Assertions
	if assert.NoError(t, eas.AuthHandler(c)) {
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	}
}
func TestAuthHandlerUserNotAuthentificatedButIsAuthorised(t *testing.T) {
	AuthentificationServer := mockupServer(9090, http.StatusUnauthorized, nil)
	defer AuthentificationServer.Close()

	AuthorisationServer := mockupServer(9091, http.StatusUnauthorized, nil)
	defer AuthorisationServer.Close()

	eas := ExternalAuthServices{
		AuthentificationClient:     AuthentificationServer.Client(),
		AuthentificationServiceUrl: "http://localhost:9090",
		AuthorisationClient:        AuthorisationServer.Client(),
		AuthorisationServiceUrl:    "http://localhost:9091",
		UserAuthentificated:        false,
		UserAuthorised:             true,
	}

	// Setup
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/auth", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Assertions
	if assert.NoError(t, eas.AuthHandler(c)) {
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	}
}

func TestRegisterHandlerUserIsAuthentificatedAndAuthorised(t *testing.T) {
	AuthentificationServer := mockupServer(9090, http.StatusUnauthorized, nil)
	defer AuthentificationServer.Close()

	AuthorisationServer := mockupServer(9091, http.StatusUnauthorized, nil)
	defer AuthorisationServer.Close()

	eas := ExternalAuthServices{
		AuthentificationClient:     AuthentificationServer.Client(),
		AuthentificationServiceUrl: "http://localhost:9090",
		AuthorisationClient:        AuthorisationServer.Client(),
		AuthorisationServiceUrl:    "http://localhost:9091",
		UserAuthentificated:        true,
		UserAuthorised:             true,
		UserId:                     "T3stUs3r",
	}

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/auth", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if assert.NoError(t, eas.AuthHandler(c)) {
		assert.Equal(t, http.StatusAccepted, rec.Code)
		assert.Equal(t, fmt.Sprintf("Hello, %s", eas.UserId), string(rec.Body.String()))
	}
}

func TestRegisterHandlerUserIsNotAuthentificated(t *testing.T) {
	AuthentificationServer := mockupServer(9090, http.StatusUnauthorized, nil)
	defer AuthentificationServer.Close()

	AuthorisationServer := mockupServer(9091, http.StatusUnauthorized, nil)
	defer AuthorisationServer.Close()

	eas := ExternalAuthServices{
		AuthentificationClient:     AuthentificationServer.Client(),
		AuthentificationServiceUrl: "http://localhost:9090",
		AuthorisationClient:        AuthorisationServer.Client(),
		AuthorisationServiceUrl:    "http://localhost:9091",
	}

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "https://localhost/register", nil)
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
	AuthentificationServer := mockupServer(9090, http.StatusAccepted, nil)
	defer AuthentificationServer.Close()

	AuthorisationServer := mockupServer(9091, http.StatusUnauthorized, nil)
	defer AuthorisationServer.Close()

	eas := ExternalAuthServices{
		AuthentificationClient:     AuthentificationServer.Client(),
		AuthentificationServiceUrl: "http://localhost:9090",
		AuthorisationClient:        AuthorisationServer.Client(),
		AuthorisationServiceUrl:    "http://localhost:9091",
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
