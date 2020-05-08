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

// go:generate go get -u github.com/GeertJohan/go.rice/rice
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

	"github.com/casbin/casbin/v2"
	"github.com/gorilla/sessions"
	casbin_mw "github.com/labstack/echo-contrib/casbin"
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
	cfgFile := &Config{}
	if err = yaml.Unmarshal(out, cfgFile); err != nil {
		fmt.Println(err)
		os.Exit(2)
	}
	cfg = cfgFile

	// Setup config
	setupOAuth2Config()

	e := echo.New()
	e.HideBanner = true
	e.Renderer = getTemplateRenderer()

	e.Use(middleware.Logger())
	e.Use(session.Middleware(sessions.NewCookieStore([]byte("secret"))))

	// Casbin setup
	enforcer, err := casbin.NewEnforcer("casbin_auth_model.conf", "casbin_auth_policy.csv")
	if err != nil {
		fmt.Println(err)
		os.Exit(2)
	}
	e.Use(casbin_mw.Middleware(enforcer))

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
		if userInfo == nil {
			return c.Redirect(http.StatusTemporaryRedirect, "/api/auth/v1/login")
		}

		return c.Render(http.StatusOK, "index", &Page{
			Title:    "Home",
			UserInfo: userInfo,
		})
	})

	e.Logger.Fatal(e.Start(":8081"))
}
