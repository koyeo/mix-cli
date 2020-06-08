package mysql

import "fmt"

func (p *Handler) printSQL(sql string) {
	fmt.Println(sql)
}
