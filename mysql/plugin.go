package mysql

import (
	"fmt"
	"github.com/koyeo/mix-cli/cmd/plugin"
	mysql2 "github.com/koyeo/mix-cli/database/mysql"
	"github.com/koyeo/snippet/logger"
	"github.com/urfave/cli"
	"os"
)

const (
	NAME       = "mysql"
	DATABASE   = "database"
	CONNECTION = "connection"
	USERS      = "users"
	USERNAME   = "username"
	PASSWORD   = "password"
	PRIVILEGES = "privileges"
	HOST       = "host"
	CHARSET    = "charset"
	COLLATE    = "collate"
	DAEMON     = "daemon"
)

const version = "1.0"

type Handler struct {
	config Config

	tables   map[string]*mysql2.Schema
	rootPath string
}

func NewHandler() *Handler {

	handler := new(Handler)

	var err error
	handler.rootPath, err = os.Getwd()
	if err != nil {
		logger.Error("New Handler error", err)
		os.Exit(1)
	}

	return handler
}

func (p *Handler) Name() string {
	return NAME
}

func (p *Handler) loadConfig() {
	err := plugin.NewConfig(NAME).UnmarshalExact(&p.config)
	if err != nil {
		logger.Error("New Handler error", err)
		os.Exit(1)
	}
}

func (p *Handler) Commands() cli.Commands {

	migrationDatabaseFlag := cli.StringFlag{
		Name:     fmt.Sprintf("%s", "database"),
		Required: true,
		Usage:    "Migration database name",
	}

	commands := []cli.Command{
		{
			Name:  "mysql:version",
			Usage: "Show mysql plugin version",
			Action: func(ctx *cli.Context) (err error) {
				fmt.Println(version)
				return
			},
		},
		{
			Name:   "mysql:make",
			Usage:  "MakeService table and view code",
			Action: p.MakeCommand,
			Flags: []cli.Flag{
				migrationDatabaseFlag,
			},
		},
		{
			Name:   "mysql:diff",
			Usage:  "Diff two databases",
			Action: p.DiffCommand,
			Flags: []cli.Flag{
				migrationDatabaseFlag,
				cli.StringFlag{
					Required: true,
					Name:     "connections",
					Usage:    "Select connections, example: conn1:conn2",
				},
				cli.StringFlag{
					Required: true,
					Name:     "databases",
					Usage:    "Select connection1, example: db1:db2",
				},
			},
		},
		{
			Name:   "mysql:sql",
			Usage:  "MakeService migration sql",
			Action: p.SqlCommand,
			Flags: []cli.Flag{
				migrationDatabaseFlag,
			},
		},
		{
			Name:        "mysql:sync",
			Usage:       "MakeService table and view info",
			Description: "Generate mysql table entities.",
			Action:      p.SyncCommand,
			Flags: []cli.Flag{
				cli.StringFlag{
					Required: true,
					Name:     "conn",
					Usage:    "Database connection, e.g: 'user:password@tcp(localhost:3306)/db_name'",
				},
				cli.StringFlag{
					Required: true,
					Name:     "dist",
					Usage:    "MakeService file storage place, e.g: ./test",
				},
			},
		},
		{
			Name:   "mysql:monitor",
			Usage:  "Show mysql monitor",
			Action: p.MonitorCommand,
			Flags: []cli.Flag{
				cli.StringFlag{
					Required: true,
					Name:     "connection",
					Usage:    "Select connection",
				},
				cli.BoolFlag{
					Name:  fmt.Sprintf("%s", DAEMON),
					Usage: "Run as daemon",
				},
			},
		},
	}

	return commands
}
