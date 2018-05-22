package main

import (
	gcfg "gopkg.in/gcfg.v1"

	"log"
)

type MysqlConfig struct {
	OauthTable        string
	OauthSchemaPrefix string
	Host              string
	Port              string
	User              string
	Password          string
}

type LdapConfig struct {
	BindDn       string
	BindPassword string
	BindUrl      string
	QueryDn      string
}

type MattermostConfig struct {
	Url      string
	Username string
	Password string
}

type config struct {
	Ldap       LdapConfig
	Mysql      MysqlConfig
	Mattermost MattermostConfig
}

func parseConfig(path string) (cfg config) {
	err := gcfg.ReadFileInto(&cfg, path)

	if err != nil {
		log.Fatal(err)
	}

	return
}
