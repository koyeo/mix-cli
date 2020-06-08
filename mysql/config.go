package mysql

import (
	"fmt"
	"strings"
)

type Config struct {
	Version     string
	Connections map[string]string
	Diff        map[string]string
	Migrations  []MigrationConfig
}

type MigrationConfig struct {
	Database   string
	Charset    string
	Collate    string
	EntityPath string
	Users      []UserConfig
}

type UserConfig struct {
	Hosts      string
	Username   string
	Privileges string
}

func (p *Handler) getMigrationConfig(database string) (migrationConfig MigrationConfig, err error) {

	exists := false
	for _, v := range p.config.Migrations {
		if v.Database == database {
			migrationConfig = v
			exists = true
			break
		}
	}

	migrationConfig.Database = strings.TrimSpace(migrationConfig.Database)
	migrationConfig.Charset = strings.TrimSpace(migrationConfig.Charset)
	migrationConfig.Collate = strings.TrimSpace(migrationConfig.Collate)
	migrationConfig.EntityPath = strings.TrimSpace(migrationConfig.EntityPath)

	for k := range migrationConfig.Users {
		migrationConfig.Users[k].Hosts = strings.TrimSpace(migrationConfig.Users[k].Hosts)
		migrationConfig.Users[k].Username = strings.TrimSpace(migrationConfig.Users[k].Username)
		migrationConfig.Users[k].Privileges = strings.TrimSpace(migrationConfig.Users[k].Privileges)
	}

	if !exists {
		err = fmt.Errorf("not match migration database config")
		return
	}

	return
}
