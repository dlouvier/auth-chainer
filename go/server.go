package main

import (
	"fmt"
	"log"
	"net/http"
	"net/url"

	"github.com/labstack/echo/v4"
)

type ExternalAuthServices struct {
	AuthentificationClient     *http.Client
	AuthentificationServiceUrl string
	AuthorisationClient        *http.Client
	AuthorisationServiceUrl    string
	UserAuthentificated        bool
	UserAuthorised             bool
	UserId                     string
}

func NewExternalAuthServices() *ExternalAuthServices {
	return &ExternalAuthServices{
		AuthentificationClient:     http.DefaultClient,
		AuthentificationServiceUrl: "https://www.google.es",
		AuthorisationClient:        http.DefaultClient,
		AuthorisationServiceUrl:    "https://www.google.es",
		UserAuthentificated:        false,
		UserAuthorised:             false,
	}
}

func (eas *ExternalAuthServices) CheckAuthentification(request *http.Request) {
	url, _ := url.Parse(eas.AuthentificationServiceUrl)
	request.Host = request.Header.Get("X-Forwarded-Host")
	request.RequestURI = ""
	request.URL.Host = url.Host
	request.URL.Path = url.Path
	request.URL.Scheme = url.Scheme
	response, err := eas.AuthentificationClient.Do(request)
	if err != nil {
		log.Printf("client: error making http request: %s\n", err)
	}
	if (response.StatusCode >= 200) && (response.StatusCode <= 202) {
		eas.UserId = response.Header.Get("X-Auth-Request-User")
		eas.UserAuthentificated = true
	}
}

func (eas *ExternalAuthServices) CheckAuthorisation(request *http.Request) {
	url, _ := url.Parse(eas.AuthorisationServiceUrl)
	request.Host = request.Header.Get("X-Forwarded-Host")
	request.RequestURI = ""
	request.URL.Host = url.Host
	request.URL.Path = url.Path
	request.URL.Scheme = url.Scheme
	response, err := eas.AuthentificationClient.Do(request)
	if err != nil {
		log.Printf("client: error making http request: %s\n", err)
	}
	if (response.StatusCode >= 200) && (response.StatusCode <= 202) {
		eas.UserAuthorised = true
	}
}

func (eas *ExternalAuthServices) AuthHandler(c echo.Context) error {
	eas.CheckAuthentification(c.Request())
	eas.CheckAuthorisation(c.Request())

	if eas.UserAuthentificated && eas.UserAuthorised {
		return c.String(http.StatusAccepted, fmt.Sprintf("Hello, %s", eas.UserId))
	} else {
		return c.String(http.StatusUnauthorized, "Unable to auth")
	}
}

func (eas *ExternalAuthServices) RegisterHandler(c echo.Context) error {
	eas.CheckAuthentification(c.Request())
	eas.CheckAuthorisation(c.Request())

	log.Println("DANIIIIIIIIIIIIIIIII")
	log.Println(c.Request())

	if !eas.UserAuthentificated {
		return c.Redirect(http.StatusTemporaryRedirect, fmt.Sprintf("http://%s/oauth2/sign_in", c.Request().Host))
	}

	if !eas.UserAuthorised {
		return c.Redirect(http.StatusTemporaryRedirect, fmt.Sprintf(
			"http://%s/webauthn/login?redirect_url=http://%s&default_username=%s",
			c.Request().Host,
			c.Request().Host,
			eas.UserId))
	}

	if eas.UserAuthentificated && eas.UserAuthorised {
		return c.String(http.StatusAccepted, "ALL DONE HERE")
	} else {
		return c.String(http.StatusUnauthorized, "Unable to auth")
	}
}

func main() {
	e := echo.New()

	eas := NewExternalAuthServices()

	// Debug mode
	e.Debug = true

	e.GET("/auth", eas.AuthHandler)

	e.GET("/register", eas.RegisterHandler)

	// Start server
	e.Logger.Fatal(e.Start(":8080"))
}
