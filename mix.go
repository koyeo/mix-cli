package main

import (
	"github.com/koyeo/mix-cli/plugin"
	"github.com/koyeo/mix-cli/swagger"
	"github.com/urfave/cli"
	"log"
	"os"
)

func loadPlugins() []plugin.Plugin {

	plugins := make([]plugin.Plugin, 0)
	plugins = append(plugins, swagger.NewHandler())

	return plugins
}

func main() {

	app := cli.NewApp()
	app.Name = "Mix"
	app.Usage = "You know you always don't know."

	plugin.InitConfig()
	plugins := loadPlugins()

	for _, p := range plugins {
		for _, command := range p.Commands() {
			app.Commands = append(app.Commands, command)
		}
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
