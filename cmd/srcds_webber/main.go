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

//go:generate go get -u github.com/GeertJohan/go.rice/rice
//go:generate rice embed-go -i github.com/galexrt/srcds_controller/cmd/srcds_webber

package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/sessions"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"gopkg.in/yaml.v2"
)

// Config config options
type Config struct {
	ServerDirectories []string     `yaml:"serverDirectories"`
	OAuth2Config      OAuth2Config `yaml:"oauth2"`
}

// OAuth2Config OAuth2 config options
type OAuth2Config struct {
	ClientID     string `yaml:"clientID"`
	ClientSecret string `yaml:"clientSecret"`
	RedirectURL  string `yaml:"redirectURL"`
	UserAPI      string `yaml:"userAPI"`
	AuthURL      string `yaml:"authURL"`
	TokenURL     string `yaml:"tokenURL"`
}

var (
	cfgFilename string
	cfg         *Config
)

func init() {
	rand.Seed(time.Now().UTC().UnixNano())

	flag.StringVar(&cfgFilename, "config", ".srcds_webber.yaml", "srcds_webber config file")
}

func main() {
	flag.Parse()
	out, err := ioutil.ReadFile(cfgFilename)
	if err != nil {
		fmt.Println(err)
		os.Exit(2)
	}
	if err = yaml.Unmarshal(out, cfg); err != nil {
		fmt.Println(err)
		os.Exit(2)
	}

	e := echo.New()
	e.HideBanner = true
	e.Renderer = getTemplateRenderer()

	e.Use(middleware.Logger())
	e.Use(session.Middleware(sessions.NewCookieStore([]byte("secret"))))

	routesAuth(e)

	e.GET("/", func(c echo.Context) error {
		sess, err := session.Get("srcds_webber", c)
		if err != nil {
			return err
		}
		sess.Options = &sessions.Options{
			Path:     "/",
			MaxAge:   86400 * 7,
			HttpOnly: true,
		}
		userInfo, err := checkForUserInfo(c)
		if err != nil {
			return err
		}
		if userInfo != nil {
			return c.HTML(http.StatusOK, fmt.Sprintf("UserInfo: %+v\n", userInfo))
		}
		return c.Redirect(http.StatusTemporaryRedirect, "/api/auth/v1/login")
	})

	e.Logger.Fatal(e.Start(":8081"))
}
