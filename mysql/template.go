package mysql

import (
	"github.com/flosch/pongo2"
	"github.com/koyeo/snippet/storage"
	"os"
	"path/filepath"
)

func (p *Handler) getTemplate(file string) (tpl *pongo2.Template, err error) {

	root, err := os.Getwd()
	if err != nil {
		return
	}

	path := filepath.Join(root, "/plugins/mysql/template/", file)

	content, err := storage.ReadFile(path)
	if err != nil {
		return
	}

	tpl, err = pongo2.FromString(content)

	if err != nil {
		return
	}

	return
}
