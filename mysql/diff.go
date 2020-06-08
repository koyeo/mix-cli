package mysql

import (
	"fmt"
	"github.com/go-xorm/xorm"
	_ "github.com/jinzhu/gorm/dialects/mysql"
	"github.com/koyeo/snippet/logger"
	"github.com/koyeo/tablewriter"
	"github.com/ttacon/chalk"
	"github.com/urfave/cli"
	"os"
	"regexp"
	"strings"
	"xorm.io/core"
)

func (p *Handler) DiffCommand(ctx *cli.Context) (err error) {

	p.loadConfig()

	args, err := p.validateDiffArgs(ctx)
	if err != nil {
		return
	}
	databases := strings.Split(args, ":")

	db1 := databases[0]
	db2 := databases[1]

	engine1 := p.getEngine(fmt.Sprintf("%s.%s", databases, db1))
	defer engine1.Close()

	engine2 := p.getEngine(fmt.Sprintf("%s.%s", databases, db2))
	defer engine2.Close()

	db1Metas, err := engine1.DBMetas()
	if err != nil {
		logger.Error("Get db metas error", err)
		return
	}
	db2Metas, err := engine2.DBMetas()
	if err != nil {
		logger.Error("Get db metas error", err)
		return
	}

	db1Count, db2Count, diffTables, sameTables := p.compareTables(db1Metas, db2Metas)

	if db1Count != 0 || db2Count != 0 {
		printFirstLevelTitle("Diff Schemas:")
		printTables(db1, db2, db1Count, db2Count, diffTables)
	}

	printFirstLevelTitle("Diff fields:")
	for _, v := range sameTables {
		tb1Count, tb2Count, diffFields := p.compareTable(v, db1Metas, db2Metas)
		if len(diffFields) > 0 {
			printFields(fmt.Sprintf("【 %s 】 fields: ", v), db1, db2, tb1Count, tb2Count, diffFields)
		}
		tb1Count, tb2Count, diffIndexes := p.compareIndexes(v, db1Metas, db2Metas)
		if len(diffIndexes) > 0 {
			printIndexes(fmt.Sprintf("【 %s 】 indexes: ", v), db1, db2, tb1Count, tb2Count, diffIndexes)
		}
	}

	return
}

func (p *Handler) validateDiffArgs(ctx *cli.Context) (databases string, err error) {

	databases = ctx.Args().First()

	reg := regexp.MustCompile(`.+:.+`)

	if !reg.MatchString(databases) {
		err = fmt.Errorf("illgeal args, expected database1:database2 which defined in config file")
	}

	return
}

func (p *Handler) getEngine(path string) (engine *xorm.Engine) {
	//engine, err := xorm.NewEngine("mysql", fmt.Sprintf("%s?charset=utf8&parseTime=True&loc=Local", p.config.Get(path)))
	//if err != nil {
	//	core.Error(err)
	//	return
	//}
	return
}

func (p *Handler) compareTables(db1Metas, db2Metas []*core.Table) (db1Count, db2Count int, diffTables [][]string, sameTables []string) {

	diffTables = make([][]string, 0)

	var db1Tables, db2Tables []string
	for _, v := range db1Metas {
		db1Tables = append(db1Tables, v.Name)
	}

	for _, v := range db2Metas {
		db2Tables = append(db2Tables, v.Name)
	}

	for _, v := range db1Tables {
		if p.inArray(v, db2Tables) {
			if !p.inArray(v, sameTables) {
				sameTables = append(sameTables, v)
			}
			continue
		}
		db1Count++
		diffTables = append(diffTables, []string{v, "-"})
	}

	for _, v := range db2Tables {
		if p.inArray(v, db1Tables) {
			if !p.inArray(v, sameTables) {
				sameTables = append(sameTables, v)
			}
			continue
		}
		db2Count++
		diffTables = append(diffTables, []string{"-", v})
	}

	return
}

func (p *Handler) compareTable(name string, db1Metas, db2Metas []*core.Table) (tb1Count, tb2Count int, diffFields [][]string) {

	diffFields = make([][]string, 0)

	tb1Fields := p.getFields(name, db1Metas)
	tb2Fields := p.getFields(name, db2Metas)

	tb1Count = p.getDiffFields(&diffFields, tb1Fields, tb2Fields)
	tb2Count = p.getDiffFields(&diffFields, tb2Fields, tb1Fields)

	return
}

func (p *Handler) getDiffFields(diffFields *[][]string, tb1Fields, tb2Fields [][]string) (count int) {

	for _, v1 := range tb1Fields {
		exists := false
		for _, v2 := range tb2Fields {
			if v1[0] == v2[0] {
				if v1[1] == v2[1] {
					exists = true
					continue

				}
				if !p.inArrays(v1[0], *diffFields) {
					*diffFields = append(*diffFields, []string{v1[0], v1[1], v2[0], v2[1]})
					count++
				}
			}
		}
		if !exists && !p.inArrays(v1[0], *diffFields) {
			count++
			*diffFields = append(*diffFields, []string{v1[0], v1[1], "-", "-"})
		}

	}

	return
}

func (p *Handler) getFields(name string, metas []*core.Table) [][]string {

	fields := make([][]string, 0)
	for _, v1 := range metas {
		if v1.Name == name {
			for _, v2 := range v1.Columns() {
				length := "0"
				if v2.SQLType.DefaultLength != 0 {
					length = fmt.Sprintf("%d", v2.SQLType.DefaultLength)
				}
				if v2.SQLType.DefaultLength2 != 0 {
					length = fmt.Sprintf("%d,%d", v2.SQLType.DefaultLength, v2.SQLType.DefaultLength)
				}
				field := fmt.Sprintf("%s(%s)", v2.SQLType.Name, length)
				fields = append(fields, []string{v2.Name, field})
			}
		}

	}
	return fields
}

func (p *Handler) compareIndexes(name string, db1Metas, db2Metas []*core.Table) (tb1Count, tb2Count int, diffIndexes [][]string) {

	diffIndexes = make([][]string, 0)

	for _, v1 := range db1Metas {
		for _, v2 := range db2Metas {
			if v1.Name == name && v2.Name == name {
				if !p.sameArray(v1.PrimaryKeys, v2.PrimaryKeys) {
					index1 := strings.Join(v1.PrimaryKeys, ",")
					index2 := strings.Join(v2.PrimaryKeys, ",")
					diffIndexes = append(diffIndexes, []string{
						"primary",
						"Primary",
						p.formatIndexCols(index1),
						"primary",
						"Primary",
						p.formatIndexCols(index2),
					})
				}
			}
		}
	}

	tb1Indexes := p.getIndexes(name, db1Metas)
	tb2Indexes := p.getIndexes(name, db2Metas)

	p.getDiffIndexes(&diffIndexes, tb1Indexes, tb2Indexes)
	p.getDiffIndexes(&diffIndexes, tb2Indexes, tb1Indexes)

	return
}

func (p *Handler) getDiffIndexes(diffIndexes *[][]string, indexes1, indexes2 [][]string) {

	for _, v1 := range indexes1 {

		if !p.inArrays(v1[0], indexes2) {
			if !p.inArrays(v1[0], *diffIndexes) {
				if len(v1) == 3 {
					v1 = append(v1, "-", "-", "-")
				}
				*diffIndexes = append(*diffIndexes, v1)
			}
		} else {
			for _, v2 := range indexes2 {
				if v1[0] == v2[0] {
					if v1[1] != v2[1] || v1[2] != v2[2] {
						if !p.inArrays(v1[0], *diffIndexes) {
							if len(v1) == 3 {
								v1 = append(v1, v2...)
							}
							*diffIndexes = append(*diffIndexes, v1)
						}
					}
				}
			}
		}
	}

}

func (p *Handler) getIndexes(name string, metas []*core.Table) [][]string {

	indexes := make([][]string, 0)
	for _, v1 := range metas {
		if v1.Name == name {
			for _, v2 := range v1.Indexes {
				indexes = append(indexes, []string{
					v2.Name,
					p.formatIndexType(v2.Type),
					p.formatIndexCols(strings.Join(v2.Cols, ",")),
				})
			}
		}

	}
	return indexes
}

func (p *Handler) inArray(value string, items []string) bool {
	for _, v := range items {
		if v == value {
			return true
		}
	}
	return false
}

func (p *Handler) inArrays(value string, items [][]string) bool {
	for _, v := range items {
		if v[0] == value {
			return true
		}
	}
	return false
}

func (p *Handler) sameArray(a1, a2 []string) bool {
	for _, v1 := range a1 {
		for _, v2 := range a2 {
			if v1 != v2 {
				return false
			}
		}
	}
	return true
}

func (p *Handler) formatIndexCols(cols string) string {
	if cols == "" {
		cols = "-"
	}
	return cols
}

func (p *Handler) formatIndexType(indexType int) string {
	switch indexType {
	case core.IndexType:
		return "Index"
	case core.UniqueType:
		return "Unique"
	}
	return "-"
}

func printFirstLevelTitle(title string) {
	fmt.Println(chalk.Magenta.Color(chalk.Bold.TextStyle(title)))
}

//func printFieldsTitle(title string) {
//	fmt.Printf("\n")
//	fmt.Println(chalk.Cyan.Color(chalk.Underline.TextStyle(title)))
//	fmt.Printf("\n")
//}
//
//func printIndexesTitle(title string) {
//	fmt.Printf("\n")
//	fmt.Println(chalk.Magenta.NewStyle().WithTextStyle(chalk.Underline).Style(title))
//	fmt.Printf("\n")
//}

func printTables(db1, db2 string, db1Count, db2Count int, data [][]string) {

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{
		fmt.Sprintf("%s", db1),
		fmt.Sprintf("%s", db2)},
	)
	table.SetFooter([]string{
		fmt.Sprintf("Diff tables: %d", db1Count),
		fmt.Sprintf("Diff tables: %d", db2Count)},
	)
	table.SetBorder(true)

	table.SetHeaderColor(
		tablewriter.Colors{tablewriter.FgHiYellowColor, tablewriter.Bold, tablewriter.BgBlackColor},
		tablewriter.Colors{tablewriter.FgHiGreenColor, tablewriter.Bold, tablewriter.BgBlackColor},
	)

	table.SetColumnColor(
		tablewriter.Colors{tablewriter.FgHiYellowColor, tablewriter.Bold, tablewriter.BgBlackColor},
		tablewriter.Colors{tablewriter.FgHiGreenColor, tablewriter.Bold, tablewriter.BgBlackColor},
	)

	table.SetFooterColor(
		tablewriter.Colors{tablewriter.FgHiYellowColor, tablewriter.Bold, tablewriter.BgBlackColor},
		tablewriter.Colors{tablewriter.FgHiGreenColor, tablewriter.Bold, tablewriter.BgBlackColor},
	)

	table.AppendBulk(data)
	table.Render()
}

func printFields(title, db1, db2 string, tb1Count, tb2Count int, data [][]string) {

	table := tablewriter.NewWriter(os.Stdout)
	table.SetCaption(true, chalk.Cyan.Color(chalk.TextStyle{}.TextStyle(title)))
	table.SetHeader([]string{
		fmt.Sprintf("%s", db1),
		"",
		fmt.Sprintf("%s", db2),
		"",
	})
	table.SetSubHeader([]string{
		"Fields",
		"typ",
		"Fields",
		"typ",
	})
	table.SetSubHeaderAlignment(tablewriter.ALIGN_LEFT)
	table.SetSubHeaderLine(true)
	table.SetBorder(true)
	table.SetHeaderColor(
		tablewriter.Colors{tablewriter.FgHiYellowColor, tablewriter.Bold, tablewriter.BgBlackColor},
		tablewriter.Colors{tablewriter.FgHiYellowColor, tablewriter.Bold, tablewriter.BgBlackColor},
		tablewriter.Colors{tablewriter.FgHiGreenColor, tablewriter.Bold, tablewriter.BgBlackColor},
		tablewriter.Colors{tablewriter.FgHiGreenColor, tablewriter.Bold, tablewriter.BgBlackColor},
	)
	table.SetSubHeaderColor(
		tablewriter.Colors{tablewriter.FgHiYellowColor, tablewriter.Bold, tablewriter.BgBlackColor},
		tablewriter.Colors{tablewriter.FgHiYellowColor, tablewriter.Bold, tablewriter.BgBlackColor},
		tablewriter.Colors{tablewriter.FgHiGreenColor, tablewriter.Bold, tablewriter.BgBlackColor},
		tablewriter.Colors{tablewriter.FgHiGreenColor, tablewriter.Bold, tablewriter.BgBlackColor},
	)

	table.SetColumnColor(
		tablewriter.Colors{tablewriter.FgHiYellowColor, tablewriter.Bold, tablewriter.BgBlackColor},
		tablewriter.Colors{tablewriter.FgHiYellowColor, tablewriter.Bold, tablewriter.BgBlackColor},
		tablewriter.Colors{tablewriter.FgHiGreenColor, tablewriter.Bold, tablewriter.BgBlackColor},
		tablewriter.Colors{tablewriter.FgHiGreenColor, tablewriter.Bold, tablewriter.BgBlackColor},
	)

	table.AppendBulk(data)
	table.Render()
	fmt.Printf("\n")
}

func printIndexes(title, db1, db2 string, tb1Count, tb2Count int, data [][]string) {

	table := tablewriter.NewWriter(os.Stdout)

	table.SetCaption(true, chalk.Magenta.Color(chalk.TextStyle{}.TextStyle(title)))
	table.SetHeader([]string{
		fmt.Sprintf("%s", db1),
		"",
		"",
		fmt.Sprintf("%s", db2),
		"",
		"",
	})
	table.SetSubHeaderAlignment(tablewriter.ALIGN_LEFT)
	table.SetSubHeaderLine(true)
	table.SetBorder(true)

	table.SetSubHeader([]string{
		"Slug",
		"typ",
		"fields",
		"Slug",
		"typ",
		"fields",
	})
	table.SetBorder(true)

	table.SetHeaderColor(
		tablewriter.Colors{tablewriter.FgHiYellowColor, tablewriter.Bold, tablewriter.BgBlackColor},
		tablewriter.Colors{tablewriter.FgHiYellowColor, tablewriter.Bold, tablewriter.BgBlackColor},
		tablewriter.Colors{tablewriter.FgHiYellowColor, tablewriter.Bold, tablewriter.BgBlackColor},
		tablewriter.Colors{tablewriter.FgHiGreenColor, tablewriter.Bold, tablewriter.BgBlackColor},
		tablewriter.Colors{tablewriter.FgHiGreenColor, tablewriter.Bold, tablewriter.BgBlackColor},
		tablewriter.Colors{tablewriter.FgHiGreenColor, tablewriter.Bold, tablewriter.BgBlackColor},
	)

	table.SetSubHeaderColor(
		tablewriter.Colors{tablewriter.FgHiYellowColor, tablewriter.Bold, tablewriter.BgBlackColor},
		tablewriter.Colors{tablewriter.FgHiYellowColor, tablewriter.Bold, tablewriter.BgBlackColor},
		tablewriter.Colors{tablewriter.FgHiYellowColor, tablewriter.Bold, tablewriter.BgBlackColor},
		tablewriter.Colors{tablewriter.FgHiGreenColor, tablewriter.Bold, tablewriter.BgBlackColor},
		tablewriter.Colors{tablewriter.FgHiGreenColor, tablewriter.Bold, tablewriter.BgBlackColor},
		tablewriter.Colors{tablewriter.FgHiGreenColor, tablewriter.Bold, tablewriter.BgBlackColor},
	)

	table.SetColumnColor(
		tablewriter.Colors{tablewriter.FgHiYellowColor, tablewriter.Bold, tablewriter.BgBlackColor},
		tablewriter.Colors{tablewriter.FgHiYellowColor, tablewriter.Bold, tablewriter.BgBlackColor},
		tablewriter.Colors{tablewriter.FgHiYellowColor, tablewriter.Bold, tablewriter.BgBlackColor},
		tablewriter.Colors{tablewriter.FgHiGreenColor, tablewriter.Bold, tablewriter.BgBlackColor},
		tablewriter.Colors{tablewriter.FgHiGreenColor, tablewriter.Bold, tablewriter.BgBlackColor},
		tablewriter.Colors{tablewriter.FgHiGreenColor, tablewriter.Bold, tablewriter.BgBlackColor},
	)

	table.AppendBulk(data)
	table.Render()
	fmt.Printf("\n")
}
