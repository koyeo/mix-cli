package plugin

import "github.com/urfave/cli"

type Plugin interface {
	Name() string
	Commands() cli.Commands
}
