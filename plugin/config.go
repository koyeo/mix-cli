package plugin

import (
	"github.com/koyeo/snippet/logger"
	"github.com/spf13/viper"
	"os"
)

var conf *viper.Viper

func InitConfig() {

	path, err := os.Getwd()
	if err != nil {
		logger.Error("Init config error", err)
		os.Exit(1)
	}
	conf = viper.New()
	conf.SetConfigName("mix.yml")
	conf.AddConfigPath(path)
	_ = conf.ReadInConfig()
}

func NewConfig(name string) *viper.Viper {
	return conf.Sub(name)
}
