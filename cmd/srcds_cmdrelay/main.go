package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"path"
	"sync"
	"syscall"
	"time"

	"github.com/galexrt/srcds_controller/pkg/config"
	"github.com/galexrt/srcds_controller/pkg/userconfig"
	"github.com/gin-gonic/gin"
	"github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	yaml "gopkg.in/yaml.v2"
)

var (
	cfgFile       string
	globalCfgFile string
	cfgMutex      = &sync.Mutex{}
	home          string
	logger        *zap.Logger
	rootCmd       = &cobra.Command{
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
			if len(viper.GetStringSlice("auth-key")) == 0 {
				return fmt.Errorf("no auth key given, refusing to start without authentication")
			}

			// Sync config every 3 minutes
			go func() {
				for {
					time.Sleep(3 * time.Minute)
					cfgMutex.Lock()
					initConfig()
					cfgMutex.Unlock()
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
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "config file (default is $HOME/.srcds_controller.yaml)")
	rootCmd.PersistentFlags().StringVar(&globalCfgFile, "global-config", "", "global config file (default is "+config.GlobalConfigPath+")")
	rootCmd.PersistentFlags().String("listen-address", "127.0.0.1:8181", "Listen address")
	rootCmd.PersistentFlags().StringSlice("auth-key", []string{}, "Auth key(s) used for authentication to the cmd relay")
	viper.BindPFlag("listen-address", rootCmd.PersistentFlags().Lookup("listen-address"))
	viper.BindPFlag("auth-key", rootCmd.PersistentFlags().Lookup("auth-key"))
}

func main() {
	syscall.Umask(7)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
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
		c.String(http.StatusForbidden, "auth key not correct")
		logger.Warn("auth key wrong")
		return
	}

	var screen string
	if c.PostForm("screen") != "" {
		screen = c.PostForm("screen")
	} else if c.Query("screen") != "" {
		screen = c.Query("screen")
	} else {
		c.String(http.StatusBadRequest, "no screen name given")
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
		c.String(http.StatusBadRequest, "no command given.")
		return
	}

	cfgMutex.Lock()
	serverCfg, ok := userconfig.Cfg.Servers[screen]
	cfgMutex.Unlock()
	if !ok {
		c.String(http.StatusInternalServerError, fmt.Sprintf("no server config found by given screen name %s", screen))
		return
	}

	httpc := http.Client{
		Transport: &http.Transport{
			DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
				return net.Dial("unix", path.Join(serverCfg.Server.Path, ".srcds_runner.sock"))
			},
		},
	}

	resp, err := httpc.PostForm("http://unixlocalhost/", url.Values{
		"command": {command},
	})
	if err != nil {
		c.String(http.StatusInternalServerError, "failed to run command. error during post")
		logger.Error("failed to run command. error during post", zap.String("command", command), zap.String("screen", screen), zap.Error(err))
		return
	}

	if resp.StatusCode == http.StatusOK {
		c.String(http.StatusOK, "success running command")
		logger.Info("success running command", zap.String("command", command), zap.Int("respcode", resp.StatusCode), zap.String("screen", screen))
		return
	}

	body, err := ioutil.ReadAll(resp.Body)

	c.String(http.StatusInternalServerError, "failed to run command, got bad response code")
	logger.Warn(
		"failed to run command, got bad response code",
		zap.String("command", command),
		zap.Int("respcode", resp.StatusCode),
		zap.String("screen", screen),
		zap.String("body", string(body)),
		zap.Error(err))
	return
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile == "" {
		// Get current work
		home, err := homedir.Dir()
		if err != nil {
			log.Fatal(err)
		}
		cfgFile = path.Join(home, ".srcds_controller.yaml")
	}

	// Load global config
	globalCfg := &config.GlobalConfig{}
	if _, err := os.Stat(globalCfgFile); err == nil {
		out, err := ioutil.ReadFile(globalCfgFile)
		if err != nil {
			log.Fatal(err)
		}
		if err = yaml.Unmarshal(out, globalCfg); err != nil {
			log.Fatal(err)
		}
	}

	userCfg := &userconfig.UserConfig{}
	cfgs := &userconfig.Config{
		Servers: map[string]*config.Config{},
	}

	if _, err := os.Stat(cfgFile); err == nil {
		out, err := ioutil.ReadFile(cfgFile)
		if err != nil {
			log.Fatal(err)
		}
		if err = yaml.Unmarshal(out, userCfg); err != nil {
			log.Fatal(err)
		}
		if err = userCfg.Load(globalCfg, cfgs); err != nil {
			log.Fatal(err)
		}
	} else {
		log.Fatal("no config found in home dir nor specified by flag")
	}

	userconfig.Cfg = cfgs
}
