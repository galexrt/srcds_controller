/*
Copyright 2020 Alexander Trost <galexrt@googlemail.com>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/galexrt/srcds_controller/pkg/util"
	"github.com/gorilla/sessions"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
	"golang.org/x/oauth2"
)

// MapStringInterface simple map string interface type
type MapStringInterface map[string]interface{}

// UserInfo user info structure
type UserInfo struct {
	UserID   int64              `json:"userID"`
	Username string             `json:"username"`
	Picture  string             `json:"picture"`
	Profile  MapStringInterface `json:"profile"`
	Groups   MapStringInterface `json:"groups"`
}

var (
	oauthConfig *oauth2.Config
)

func setupOAuth2Config() {
	oauthConfig = &oauth2.Config{
		RedirectURL:  cfg.OAuth2Config.RedirectURL,
		ClientID:     cfg.OAuth2Config.ClientID,
		ClientSecret: cfg.OAuth2Config.ClientSecret,
		Scopes:       []string{"openid", "profile"},
		Endpoint: oauth2.Endpoint{
			AuthURL:  cfg.OAuth2Config.AuthURL,
			TokenURL: cfg.OAuth2Config.TokenURL,
		},
	}
}

func routesAuth(e *echo.Echo) {
	auth := e.Group("/api/auth/v1")
	auth.GET("/", func(c echo.Context) error {
		return c.Redirect(http.StatusTemporaryRedirect, "login")
	})
	auth.GET("/login", func(c echo.Context) error {
		return c.Render(http.StatusOK, "api/auth/v1/login.html", &Page{
			Title: "Login",
		})
	})
	auth.GET("/logout", func(c echo.Context) error {
		sess, err := session.Get("srcds_webber", c)
		if err != nil {
			return err
		}
		sess.Options = &sessions.Options{
			Path:     "/",
			MaxAge:   0,
			HttpOnly: true,
		}
		sess.Values = map[interface{}]interface{}{}
		sess.Save(c.Request(), c.Response())

		return c.Redirect(http.StatusTemporaryRedirect, "/api/auth/v1/login")
	})
	auth.POST("/openid", func(c echo.Context) error {
		sess, err := session.Get("srcds_webber", c)
		if err != nil {
			return err
		}
		sess.Options = &sessions.Options{
			Path:     "/",
			MaxAge:   86400 * 7,
			HttpOnly: true,
		}

		stateString, err := util.GenerateRandomString(64)
		if err != nil {
			return err
		}

		sess.Values["openid-state"] = stateString
		sess.Save(c.Request(), c.Response())

		url := oauthConfig.AuthCodeURL(stateString)
		return c.Redirect(http.StatusTemporaryRedirect, url)
	})
	auth.Match([]string{"GET", "POST"}, "/openid/callback", func(c echo.Context) error {
		sess, err := session.Get("srcds_webber", c)
		if err != nil {
			return err
		}
		sess.Options = &sessions.Options{
			Path:     "/",
			MaxAge:   86400 * 1,
			HttpOnly: true,
		}
		ourState, ok := sess.Values["openid-state"]

		fmt.Printf("Outstate: %+v - UserState: %+v\n", ourState.(string), c.FormValue("state"))

		if !ok {
			return c.String(http.StatusForbidden, "Wrong OpenID State")
		}

		content, err := getUserInfo(ourState.(string), c.FormValue("state"), c.FormValue("code"))
		if err != nil {
			fmt.Println(err.Error())
			return c.Redirect(http.StatusTemporaryRedirect, "/api/auth/v1/login?msg=err")
		}

		userInfo := UserInfo{}
		if err := json.Unmarshal(content, &userInfo); err != nil {
			return err
		}

		out, err := json.Marshal(userInfo)
		if err != nil {
			return err
		}
		sess.Values["userinfo"] = out
		sess.Save(c.Request(), c.Response())

		return c.Redirect(http.StatusTemporaryRedirect, "/")
	})
}

func checkForUserInfo(c echo.Context) (*UserInfo, error) {
	sess, err := session.Get("srcds_webber", c)
	if err != nil {
		return nil, err
	}
	sess.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   86400 * 7,
		HttpOnly: true,
	}
	data, ok := sess.Values["userinfo"]
	if !ok {
		return nil, nil
	}
	out, ok := data.([]byte)
	if !ok {
		return nil, fmt.Errorf("unable to cast byte userinfo in session")
	}

	userInfo := &UserInfo{}
	if err := json.Unmarshal(out, userInfo); err != nil {
		return nil, err
	}

	return userInfo, nil
}

func getUserInfo(ourState, userState string, code string) ([]byte, error) {
	if ourState != userState {
		return nil, fmt.Errorf("invalid oauth state")
	}
	token, err := oauthConfig.Exchange(oauth2.NoContext, code)
	if err != nil {
		return nil, fmt.Errorf("code exchange failed: %s", err.Error())
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	client := oauthConfig.Client(ctx, token)

	// Get user info
	resp, err := client.Get(cfg.OAuth2Config.UserAPI)
	if err != nil {
		return nil, fmt.Errorf("failed getting user info: %s", err.Error())
	}
	defer resp.Body.Close()

	contents, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed reading response body: %s", err.Error())
	}

	return contents, nil
}
