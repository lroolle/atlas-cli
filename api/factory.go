package api

import (
	"fmt"
	"strings"
	"sync"

	"github.com/spf13/viper"
)

var (
	bitbucketClient  *BitbucketClient
	jiraClient       *JiraClient
	confluenceClient *ConfluenceClient
	clientMutex      sync.Mutex
)

type InstallationType string

const (
	InstallationTypeCloud InstallationType = "cloud"
	InstallationTypeServer InstallationType = "server"
)

func GetBitbucketClient() (*BitbucketClient, error) {
	clientMutex.Lock()
	defer clientMutex.Unlock()

	if bitbucketClient != nil {
		return bitbucketClient, nil
	}

	server := viper.GetString("bitbucket.server")
	username := viper.GetString("bitbucket.username")
	if username == "" {
		username = viper.GetString("username")
	}
	token := viper.GetString("bitbucket.token")

	if server == "" || username == "" || token == "" {
		return nil, fmt.Errorf("bitbucket configuration incomplete: server, username, and token required")
	}

	bitbucketClient = NewBitbucketClient(server, username, token)
	return bitbucketClient, nil
}

func GetJiraClient() (*JiraClient, error) {
	clientMutex.Lock()
	defer clientMutex.Unlock()

	if jiraClient != nil {
		return jiraClient, nil
	}

	server := viper.GetString("jira.server")
	username := viper.GetString("jira.username")
	if username == "" {
		username = viper.GetString("username")
	}
	token := viper.GetString("jira.token")

	if server == "" || username == "" || token == "" {
		return nil, fmt.Errorf("jira configuration incomplete: server, username, and token required")
	}

	jiraClient = NewJiraClient(server, username, token)
	
	// Detect installation type
	installationType := viper.GetString("jira.installation")
	if installationType == "" {
		if strings.Contains(server, ".atlassian.net") {
			installationType = string(InstallationTypeCloud)
		} else {
			installationType = string(InstallationTypeServer)
		}
		viper.Set("jira.installation", installationType)
	}

	return jiraClient, nil
}

func GetConfluenceClient() (*ConfluenceClient, error) {
	clientMutex.Lock()
	defer clientMutex.Unlock()

	if confluenceClient != nil {
		return confluenceClient, nil
	}

	server := viper.GetString("confluence.server")
	username := viper.GetString("confluence.username")
	if username == "" {
		username = viper.GetString("username")
	}
	token := viper.GetString("confluence.token")

	if server == "" || username == "" || token == "" {
		return nil, fmt.Errorf("confluence configuration incomplete: server, username, and token required")
	}

	confluenceClient = NewConfluenceClient(server, username, token)
	return confluenceClient, nil
}

func ResetClients() {
	clientMutex.Lock()
	defer clientMutex.Unlock()
	
	bitbucketClient = nil
	jiraClient = nil
	confluenceClient = nil
}