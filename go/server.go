package main

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/gorilla/sessions"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
)

const (
	OAUTH2_PROXY_HOST = "http://oauth2-proxy.default.svc.cluster.local:4180/oauth2/auth"
	WEBAUTHN_HOST     = "http://webauthn.default.svc.cluster.local:8080/webauthn/auth"
)

func ExternalAuthSucessful(serviceUrl string, request *http.Request) (bool, string) {
	log.Printf("Validating requests against: %s", serviceUrl)
	url, _ := url.Parse(serviceUrl)
	request.Header.Set("X-Forwarded-Host", request.Header.Get("X-Forwarded-Host"))
	request.Host = request.Header.Get("X-Forwarded-Host")
	request.RequestURI = ""
	request.URL.Host = url.Host
	request.URL.Path = url.Path
	request.URL.Scheme = url.Scheme

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		log.Printf("client: error making http request: %s\n", err)
	}

	if (response.StatusCode >= 200) && (response.StatusCode <= 202) {
		log.Println("Validation successful")
		return true, "dlouvier@protonmail.com"
	} else if response.StatusCode == 401 {
		log.Println("User not authorised successful")
		return false, ""
	} else {
		log.Println("There was an error.")
		return false, ""
	}
}

func AuthorisationHandle(c echo.Context) error {
	register := strings.Contains(c.Request().URL.RequestURI(), "register") // true
	log.Println("What is in register")
	log.Println(register)
	log.Printf("Yooo what it is hereeee: %s", c.Request().URL.RequestURI())
	sess, _ := session.Get("session", c)
	auth_oauth2_proxy, _ := sess.Values["auth_oauth2_proxy"].(bool)
	auth_user, _ := sess.Values["auth_user"].(string)
	auth_webauthn, _ := sess.Values["auth_webauthn"].(bool)
	log.Printf("Current session")
	log.Println(sess)
	if register {
		log.Println("I am in register")
		if !auth_oauth2_proxy {
			log.Println("Missing Oauth - redirecting...")
			return c.Redirect(http.StatusTemporaryRedirect, fmt.Sprintf("https://%s/oauth2/sign_in", c.Request().Host))
		}
		if !auth_webauthn {
			log.Println("Missing webauth - redirecting")
			return c.Redirect(http.StatusTemporaryRedirect, fmt.Sprintf("https://%s/webauthn/login?redirect_url=https://%s&default_username=%s", c.Request().Host, c.Request().Host, auth_user))
		}
	}

	if !auth_oauth2_proxy {
		success, user := ExternalAuthSucessful(OAUTH2_PROXY_HOST, c.Request())
		if success {
			log.Println("Sucessfully found the user and setting cookie for ouath2")
			sess.Values["auth_oauth2_proxy"] = true
			sess.Values["auth_user"] = user
			sess.Save(c.Request(), c.Response())
			auth_webauthn = true
		} else {
			log.Println("Setting oauth_proxy not possible, continue")
		}
	}

	if !auth_webauthn {
		success, _ := ExternalAuthSucessful(WEBAUTHN_HOST, c.Request())
		if success {
			log.Println("Sucessfully found the user and setting cookie for webauthn")
			sess.Values["auth_webauthn"] = true
			sess.Save(c.Request(), c.Response())
			auth_webauthn = true
		} else {
			log.Println("Setting webauthn not possible, continue")
		}

	}

	log.Println("telllll me alll")
	log.Printf("--- oauth2_proxy: %t  - webauthn: %t", auth_oauth2_proxy, auth_webauthn)

	if auth_oauth2_proxy && auth_webauthn {
		return c.NoContent(http.StatusOK)
	} else {
		return c.NoContent(http.StatusUnauthorized)
	}
}

func main() {
	e := echo.New()

	// Debug mode
	e.Debug = true

	e.Use(session.Middleware(sessions.NewCookieStore([]byte("secret"))))

	e.GET("/auth", AuthorisationHandle)

	e.GET("/register", AuthorisationHandle)

	// Start server
	e.Logger.Fatal(e.Start(":1338"))
}
