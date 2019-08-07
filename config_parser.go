package main

import (
	gcfg "gopkg.in/gcfg.v1"

	"log"
)

// MysqlConfig describes all possible MySQL configuration fields
type MysqlConfig struct {
	OauthDB           string
	OauthSchemaPrefix string
	Host              string
	Port              string
	User              string
	Password          string
}

// LdapConfig describes all possible LDAP configuration fields
type LdapConfig struct {
	BindDn           string
	BindPassword     string
	BindURL          string
	QueryDn          string
	GroupMemberQuery string
	GroupBaseDN      string

	AttrSelectors []string
}

// OauthConfig describes all possible Oauth configuration fields
type OauthConfig struct {
	StaticPath   string
	TemplatePath string
	RouteStatic  string
	RouteLogin   string
	RouteToken   string
	RouteInfo    string
}

// MattermostConfig describes all possible Mattermost configuration fields
type MattermostConfig struct {
	URL            string
	Username       string
	Password       string
	UsernamePrefix string
}

// GeneralConfig describes all general configuration properties
type GeneralConfig struct {
	ListenAddr string
}

type config struct {
	Ldap       LdapConfig
	Mysql      MysqlConfig
	Oauth      OauthConfig
	Mattermost MattermostConfig
	General    GeneralConfig
}

func parseConfig(path string) (cfg config) {
	err := gcfg.ReadFileInto(&cfg, path)

	if err != nil {
		log.Fatal(err)
	}

	return
}
