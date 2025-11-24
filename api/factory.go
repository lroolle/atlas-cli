package api

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

type InstallationType string

const (
	InstallationTypeCloud  InstallationType = "cloud"
	InstallationTypeServer InstallationType = "server"
)

type Config struct {
	Server   string
	Username string
	Token    string
}

func GetBitbucketClient() (*BitbucketClient, error) {
	cfg, err := loadConfig("bitbucket")
	if err != nil {
		return nil, err
	}
	return NewBitbucketClient(cfg.Server, cfg.Username, cfg.Token), nil
}

func GetJiraClient() (*JiraClient, error) {
	cfg, err := loadConfig("jira")
	if err != nil {
		return nil, err
	}

	client := NewJiraClient(cfg.Server, cfg.Username, cfg.Token)

	installationType := viper.GetString("jira.installation")
	if installationType == "" {
		if strings.Contains(cfg.Server, ".atlassian.net") {
			installationType = string(InstallationTypeCloud)
		} else {
			installationType = string(InstallationTypeServer)
		}
	}
	client.InstallationType = InstallationType(installationType)

	return client, nil
}

func GetConfluenceClient() (*ConfluenceClient, error) {
	cfg, err := loadConfig("confluence")
	if err != nil {
		return nil, err
	}
	return NewConfluenceClient(cfg.Server, cfg.Username, cfg.Token), nil
}

func loadConfig(service string) (Config, error) {
	server := viper.GetString(service + ".server")
	username := viper.GetString(service + ".username")
	if username == "" {
		username = viper.GetString("username")
	}
	token := viper.GetString(service + ".token")

	if server == "" || username == "" || token == "" {
		return Config{}, fmt.Errorf("%s configuration incomplete: server, username, and token required", service)
	}

	return Config{
		Server:   server,
		Username: username,
		Token:    token,
	}, nil
}