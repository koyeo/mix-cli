package mysql

import (
	"github.com/fatih/structs"
	"github.com/flosch/pongo2"
	mysql "github.com/koyeo/mix-cli/database/mysql"
	"github.com/koyeo/snippet/storage"
	"github.com/urfave/cli"
	"strings"
)

type tableContext struct {
	TableName     string
	TableEngine   string
	TableCharset  string
	AutoIncrement int
	Fields        []string
}

type indexContext struct {
	Type   string
	Name   string
	Fields []string
}

type primaryKeyContext struct {
	Keys []string
}

func (p *Handler) SqlCommand(ctx *cli.Context) (err error) {

	p.loadConfig()

	db := ctx.String(DATABASE)

	migrationConfig, err := p.getMigrationConfig(db)
	if err != nil {
		return
	}

	if migrationConfig.Charset == "" {
		migrationConfig.Charset = mysql.DefaultDatabaseCharset
	}

	if migrationConfig.Collate == "" {
		migrationConfig.Collate = mysql.DefaultDatabaseCollate
	}

	// 1. Database SQL
	tpl, err := p.getTemplate(createDatabaseTpl)
	if err != nil {
		return
	}

	sql, err := tpl.Execute(pongo2.Context{
		DATABASE: migrationConfig.Database,
		CHARSET:  migrationConfig.Charset,
		COLLATE:  migrationConfig.Collate,
	})
	if err != nil {
		return
	}

	p.printSQL(sql)

	// 2. User SQL
	tpl, err = p.getTemplate(createUserTpl)
	if err != nil {
		return
	}

	data := make([]map[string]interface{}, 0)

	for _, user := range migrationConfig.Users {
		hosts := strings.Split(user.Hosts, ",")
		for _, host := range hosts {
			data = append(data, map[string]interface{}{
				USERNAME:   user.Username,
				PASSWORD:   "123456",
				HOST:       host,
				PRIVILEGES: user.Privileges,
			})
		}
	}

	sql, err = tpl.Execute(pongo2.Context{
		DATABASE: migrationConfig.Database,
		USERS:    data,
	})
	if err != nil {
		return
	}

	p.printSQL(sql)

	// 3. EntityName SQL
	tables, err := mysql.ReadTableFiles(storage.Abs(migrationConfig.EntityPath))
	if err != nil {
		return
	}

	pongo2.SetAutoescape(false)

	tableTpl, err := p.getTemplate(createTableTpl)
	if err != nil {
		return
	}

	fieldTpl, err := p.getTemplate(createFieldTpl)
	if err != nil {
		return
	}

	primaryTpl, err := p.getTemplate(createPrimaryTpl)
	if err != nil {
		return
	}

	indexTpl, err := p.getTemplate(createIndexTpl)
	if err != nil {
		return
	}

	var statements []string

	for _, table := range tables.List {

		data := tableContext{
			TableName:    table.Name,
			TableEngine:  table.Engine,
			TableCharset: table.Charset,
		}

		var sql string

		for _, v := range table.Field {

			sql, err = fieldTpl.Execute(pongo2.Context(structs.Map(v)))
			if err != nil {
				return err
			}
			data.Fields = append(data.Fields, sql)
		}

		sql, err = primaryTpl.Execute(pongo2.Context(structs.Map(primaryKeyContext{
			Keys: strings.Split(table.Primary, ","),
		})))
		if err != nil {
			return err
		}
		data.Fields = append(data.Fields, sql)

		for _, v := range table.Index {
			sql, err = indexTpl.Execute(pongo2.Context(structs.Map(indexContext{
				Type:   v.Type,
				Name:   v.Name,
				Fields: strings.Split(v.Fields, ","),
			})))
			data.Fields = append(data.Fields, sql)
		}

		for k := range data.Fields {
			if k != len(data.Fields)-1 {
				data.Fields[k] += ","
			}
		}

		sql, err = tableTpl.Execute(pongo2.Context(structs.Map(data)))
		if err != nil {
			return
		}

		statements = append(statements, sql)
	}

	for _, sql := range statements {
		p.printSQL(sql)
	}

	return
}
