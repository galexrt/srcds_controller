package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"path"
	"syscall"
	"time"

	"github.com/galexrt/srcds_controller/pkg/config"
	"github.com/galexrt/srcds_controller/pkg/userconfig"
	"github.com/gin-gonic/gin"
	"github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	yaml "gopkg.in/yaml.v2"
)

var (
	home    string
	logger  *zap.Logger
	rootCmd = &cobra.Command{
		Use:   "srcds_cmdrelay",
		Short: "",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			var err error
			logger, err = zap.NewDevelopment()
			if err != nil {
				return err
			}

			initConfig()
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			// Sync config every 3 minutes
			go func() {
				for {
					time.Sleep(3 * time.Minute)
					userconfig.Cfg.Lock()
					initConfig()
					userconfig.Cfg.Unlock()
				}
			}()

			r := gin.Default()
			r.GET("/", handler)
			r.POST("/", handler)
			logger.Info(fmt.Sprintf("listening on %s", viper.GetString("listen-address")))
			if err := r.Run(viper.GetString("listen-address")); err != nil {
				return err
			}
			return nil
		},
	}
)

func init() {
	rootCmd.PersistentFlags().String("listen-address", "127.0.0.1:8181", "Listen address")
	rootCmd.PersistentFlags().String("config", "", "Config file path")
	rootCmd.PersistentFlags().StringSlice("auth-key", []string{}, "Auth key(s) used for authentication to the cmd relay")
	viper.BindPFlag("listen-address", rootCmd.PersistentFlags().Lookup("listen-address"))
	viper.BindPFlag("config", rootCmd.PersistentFlags().Lookup("config"))
	viper.BindPFlag("auth-key", rootCmd.PersistentFlags().Lookup("auth-key"))
}

func handler(c *gin.Context) {
	var authKey string
	if c.PostForm("auth-key") != "" {
		authKey = c.PostForm("auth-key")
	} else if c.Query("auth-key") != "" {
		authKey = c.Query("auth-key")
	} else {
		c.String(http.StatusBadRequest, "No auth key given.")
		logger.Warn("no auth key given")
		return
	}
	authed := false
	for _, aKey := range viper.GetStringSlice("auth-key") {
		if authKey == aKey {
			authed = true
			break
		}
	}
	if !authed {
		c.String(http.StatusForbidden, "auth key not correct.")
		logger.Warn("auth key wrong")
		return
	}

	var screen string
	if c.PostForm("screen") != "" {
		screen = c.PostForm("screen")
	} else if c.Query("screen") != "" {
		screen = c.Query("screen")
	} else {
		c.String(http.StatusBadRequest, "No screen name given.")
		logger.Warn("no screen name given")
		return
	}

	var command string
	if c.PostForm("command") != "" {
		command = c.PostForm("command")
	} else if c.Query("command") != "" {
		command = c.Query("command")
	} else {
		logger.Warn("no command given")
		c.String(http.StatusBadRequest, "No command given.")
		return
	}

	userconfig.Cfg.Lock()
	serverCfg, ok := userconfig.Cfg.Servers[screen]
	userconfig.Cfg.Unlock()
	if !ok {
		c.String(http.StatusInternalServerError, fmt.Sprintf("no server config found by given screen name %s", screen))
		return
	}

	httpc := http.Client{
		Transport: &http.Transport{
			DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
				return net.Dial("unix", fmt.Sprintf("unix://%s", path.Join(serverCfg.Server.Path, ".srcds_runner.sock")))
			},
		},
	}
	resp, err := httpc.PostForm("http://unixlocalhost/", url.Values{
		"command": {command},
	})
	if err != nil {
		c.String(http.StatusInternalServerError, "failed to run command. error during post")
		logger.Warn("failed to run command. error during post", zap.String("command", command), zap.String("screen", screen))
		return
	}

	if resp.StatusCode == http.StatusOK {
		c.String(http.StatusOK, "success running command")
		logger.Info("success running command", zap.String("command", command), zap.Int("respcode", resp.StatusCode), zap.String("screen", screen))
		return
	}

	c.String(http.StatusInternalServerError, "failed to run command, got bad response code")
	logger.Warn("failed to run command, got bad response code", zap.String("command", command), zap.Int("respcode", resp.StatusCode), zap.String("screen", screen))
	return
}

func main() {
	syscall.Umask(7)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	cfgFile := viper.GetString("config")
	if cfgFile == "" {
		// Get current work
		home, err := homedir.Dir()
		if err != nil {
			logger.Fatal("failed to get home dir", zapcore.Field{Key: "error", Interface: err})
		}
		cfgFile = path.Join(home, ".srcds_controller.yaml")
	}
	userCfg := &userconfig.UserConfig{}
	cfgs := &userconfig.Config{
		Servers: map[string]*config.Config{},
	}

	if _, err := os.Stat(cfgFile); err == nil {
		out, err := ioutil.ReadFile(cfgFile)
		if err != nil {
			logger.Fatal(fmt.Sprintf("failed to read userconfig dir from %s", cfgFile), zapcore.Field{Key: "error", Interface: err})
		}
		if err = yaml.Unmarshal(out, userCfg); err != nil {
			logger.Fatal("failed to unmarshal userconfig", zapcore.Field{Key: "error", Interface: err})
		}
		if err = userCfg.Load(cfgs); err != nil {
			logger.Fatal("failed to load configs from userconfig", zap.Error(err))
		}
	} else {
		logger.Fatal("no config found in home dir nor specified by flag")
	}

	userconfig.Cfg = cfgs
}
