package main

import (
	"os"
	"strings"

	"github.com/fsnotify/fsnotify"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

var (
	lg = logrus.New()
)

func setupConfig() *viper.Viper {
	cfg := viper.New()
	cfg.AddConfigPath(".")
	cfg.AddConfigPath("$HOME/ca-injector")
	cfg.AddConfigPath("/etc/ca-injector")

	cfg.SetConfigName("ca-injector")
	cfg.SetEnvPrefix("CERT-INJECTOR")

	cfg.AutomaticEnv()
	cfg.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	if err := cfg.ReadInConfig(); err != nil {
		lg.WithError(err).Error("could not read initial config")
	}

	cfg.OnConfigChange(func(_ fsnotify.Event) {
		if err := cfg.ReadInConfig(); err != nil {
			lg.WithError(err).Warn("could not reload config")
		}
	})
	if os.Getenv("KUBERNETES_SERVICE_PORT") != "" {
		lg.SetFormatter(&logrus.JSONFormatter{})
	}

	go cfg.WatchConfig()

	return cfg
}
