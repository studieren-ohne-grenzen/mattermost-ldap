package main

import (
	"database/sql"
	"log"

	"github.com/jasonlvhit/gocron"

	"github.com/RangelReale/osin"
	mauth "github.com/zonradkuse/oauth-authenticator"
)

func main() {
	var cli cliParameters
	err := handleCLIParameters(&cli)

	if err != nil {
		showDefaults()
		log.Fatal(err)
	}

	config := parseConfig(*cli.ConfigPath)

	log.Println("Initializing SQL connection")
	url := config.Mysql.User + ":" + config.Mysql.Password + "@tcp(" + config.Mysql.Host + ":" + config.Mysql.Port + ")/" + config.Mysql.OauthDB + "?parseTime=true"

	db, err := sql.Open("mysql", url)
	if err != nil {
		log.Fatal(err)
	}

	cfg := osin.NewServerConfig()
	cfg.AllowGetAccessRequest = true
	cfg.AllowClientSecretInParams = true

	var transformer Transformer
	// If ever necessary - refactor them into config.Ldap, however they are quite standard keep them for now
	transformer.CNAttrName = "cn"
	transformer.MailAttrName = "mail"
	transformer.UIDAttrName = "uid"
	transformer.UsernamePrefix = config.Mattermost.UsernamePrefix

	ldapAuthenticator := NewAuthenticatorWithSync(config.Ldap.BindDn, config.Ldap.BindPassword, config.Ldap.QueryDn, config.Ldap.GroupMemberQuery, config.Ldap.GroupBaseDN, transformer)
	if err := ldapAuthenticator.Connect(config.Ldap.BindURL); err != nil {
		log.Fatal(err)
	}

	if err := ldapAuthenticator.ConnectMattermost(config.Mattermost.URL, config.Mattermost.Username, config.Mattermost.Password); err != nil {
		log.Fatal(err)
	}

	oauthServer := mauth.NewServer(db, config.Mysql.OauthSchemaPrefix, cfg, &ldapAuthenticator)
	oauthServer.RouteInfo = config.Oauth.RouteInfo
	oauthServer.RouteLogin = config.Oauth.RouteLogin
	oauthServer.RouteStatic = config.Oauth.RouteStatic
	oauthServer.RouteToken = config.Oauth.RouteToken
	oauthServer.StaticPath = config.Oauth.StaticPath
	oauthServer.TemplatePath = config.Oauth.TemplatePath

	if *cli.StartServer {
		gocron.Every(5).Minutes().Do(ldapAuthenticator.syncAllOAuthUsers)
		gocron.Start()

		oauthServer.ListenAndServe(config.General.ListenAddr)

	}

	if *cli.AddClient {
		oauthServer.CreateClient(*cli.ClientID, *cli.ClientSecret, *cli.RedirectURI)
	}

	if *cli.RevokeClient {
		oauthServer.RemoveClient(*cli.ClientID)
	}
}
