package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/sessions"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
)

type UserSession struct {
	Valid  bool
	UserId string
}

func NewUserSession() *UserSession {
	return &UserSession{
		Valid:  false,
		UserId: "",
	}
}

type ExternalAuthServices struct {
	AuthentificationClient      *http.Client
	AuthentificationServiceHost string
	AuthorisationClient         *http.Client
	AuthorisationServiceHost    string
}

func NewExternalAuthServices() *ExternalAuthServices {
	authentificationServiceHost, envVarSet := os.LookupEnv("AUTHENTIFICATION_SERVICE_HOST")
	if !envVarSet {
		log.Fatalln("AUTHENTIFICATION_SERVICE_HOST enviroment variable is not set.")
	}
	authorisationServiceHost, envVarSet := os.LookupEnv("AUTHORISATION_SERVICE_HOST")
	if !envVarSet {
		log.Fatalln("AUTHORISATION_SERVICE_HOST enviroment variable is not set.")
	}

	return &ExternalAuthServices{
		AuthentificationClient:      http.DefaultClient,
		AuthentificationServiceHost: authentificationServiceHost,
		AuthorisationClient:         http.DefaultClient,
		AuthorisationServiceHost:    authorisationServiceHost,
	}
}

func (eas *ExternalAuthServices) isUserAuthentificated(request *http.Request) (bool, string, error) {
	request.Host = GetHostFromRequest(request)
	request.RequestURI = ""
	request.URL.Host = eas.AuthentificationServiceHost
	request.URL.Path = "/oauth2/auth"
	request.URL.Scheme = "http"
	response, err := eas.AuthentificationClient.Do(request)

	if err != nil {
		return false, "", err
	}
	if response.StatusCode == 202 {
		userId := response.Header.Get("X-Auth-Request-User")
		return true, userId, nil
	}
	return false, "", nil
}

func (eas *ExternalAuthServices) isUserAuthorised(request *http.Request) (bool, error) {
	request.Host = GetHostFromRequest(request)
	request.RequestURI = ""
	request.URL.Host = eas.AuthorisationServiceHost
	request.URL.Path = "/webauthn/auth"
	request.URL.Scheme = "http"
	response, err := eas.AuthentificationClient.Do(request)
	if err != nil {
		return false, err
	}
	log.Println(response.StatusCode)
	if response.StatusCode == 200 {
		return true, nil
	}
	return false, nil
}

func AuthHandler(c echo.Context) error {
	sess, err := session.Get("session", c)
	if err != nil {
		log.Println("Error obtaining the session")
		log.Println(err)
	}

	userSession, ok := sess.Values["session"].(UserSession)
	if !ok {
		log.Println("User session is missing,")
		return c.String(http.StatusUnauthorized, "Unable to auth")
	}

	if userSession.Valid {
		c.Response().Header().Set("X-Auth-Request-User", userSession.UserId)
		return c.String(http.StatusAccepted, fmt.Sprintf("Hello, %s", userSession.UserId))
	} else {
		return c.String(http.StatusUnauthorized, "Unable to auth")
	}
}

func (eas *ExternalAuthServices) RegisterHandler(c echo.Context) error {
	host := GetHostFromRequest(c.Request())

	authentificated, userId, err := eas.isUserAuthentificated(c.Request())
	if err != nil {
		log.Printf("There was an error trying to authentificate the user: \n%s", err)
		return err
	}
	if !authentificated {
		return c.Redirect(http.StatusTemporaryRedirect, fmt.Sprintf("%s://%s/oauth2/sign_in", c.Scheme(), host))
	}

	authorised, err := eas.isUserAuthorised(c.Request())
	if err != nil {
		log.Printf("There was an error trying to authorise the device: \n%s", err)
		return err
	}
	if !authorised {
		return c.Redirect(http.StatusTemporaryRedirect, fmt.Sprintf(
			"%s://%s/webauthn/login?redirect_url=%s://%s&default_username=%s",
			c.Scheme(),
			host,
			c.Scheme(),
			host,
			userId))
	}

	if authentificated && authorised {
		sess, err := session.Get("session", c)
		if err != nil {
			log.Printf("There was an error trying obtain the user session: \n%s", err)
			return c.NoContent(http.StatusInternalServerError)
		}

		sess.Values["session"] = UserSession{
			Valid:  true,
			UserId: userId,
		}

		sess.Save(c.Request(), c.Response())

		c.Response().Header().Set("X-Auth-Request-User", userId)

		return c.String(http.StatusAccepted, fmt.Sprintf("Hello, %s", userId))
	} else {
		return c.NoContent(http.StatusInternalServerError)
	}
}

func GetHostFromRequest(request *http.Request) string {
	var host string

	if len(request.Header.Get("X-Forwarded-Host")) > 0 {
		host = request.Header.Get("X-Forwarded-Host")
	} else {
		host = request.Host
	}

	return host
}

func main() {
	e := echo.New()

	eas := NewExternalAuthServices()

	// Debug mode
	e.Debug = true

	e.Use(session.Middleware(sessions.NewCookieStore([]byte("secret"))))

	e.GET("/auth", AuthHandler)

	e.GET("/register", eas.RegisterHandler)

	// Start server
	e.Logger.Fatal(e.Start(":8080"))
}
