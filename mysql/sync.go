package mysql

import (
	"github.com/koyeo/mix-cli/database/mysql"
	"github.com/urfave/cli"
)

func (p *Handler) SyncCommand(ctx *cli.Context) (err error) {

	_, err = mysql.dump(ctx.String("conn"), ctx.String("dist"))
	if err != nil {
		return
	}

	return
}
