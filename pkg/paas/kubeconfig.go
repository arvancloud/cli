package paas

import (
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

type KubeConfig struct {
	ApiVersion     string            `yaml:"apiVersion"`
	Clusters       []KubeCluster     `yaml:"clusters,omitempty"`
	Contexts       []KubeContext     `yaml:"contexts,omitempty"`
	CurrentContext string            `yaml:"current-context"`
	Kind           string            `yaml:"kind"`
	Preferences    map[string]string `yaml:"preferences"`
	Users          []User            `yaml:"users,omitempty"`
}

type KubeCluster struct {
	Cluster ClusterInfo `yaml:"cluster"`
	Name    string      `yaml:"name"`
}

type ClusterInfo struct {
	InsecureSkipTlsVerify *bool  `yaml:"insecure-skip-tls-verify,omitempty"`
	Server                string `yaml:"server"`
}

type KubeContext struct {
	Context ContextInfo `yaml:"context"`
	Name    string      `yaml:"name"`
}

type ContextInfo struct {
	Cluster   string  `yaml:"cluster"`
	Namespace *string `yaml:"namespace,omitempty"`
	User      string  `yaml:"user"`
}

type User struct {
	Name string   `yaml:"name"`
	User UserInfo `yaml:"user"`
}

type UserInfo struct {
	Token string `yaml:"token"`
}

func loadCurrentKubeConfig(path string) *KubeConfig {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil
	}
	kubeConfigData := KubeConfig{}
	err = yaml.Unmarshal(data, &kubeConfigData)
	if err != nil {
		return nil
	}

	return &kubeConfigData
}

func populateKubeConfig(arvanPaasServer, arvanHostnamePort, username, token string, projects []string, path string) KubeConfig {
	kubeConfigData := KubeConfig{}
	kubeConfigData.ApiVersion = "v1"
	kubeConfigData.Kind = "Config"
	insecureSkipTlsVerify := true
	kubeCluster := KubeCluster{
		Name: arvanHostnamePort,
		Cluster: ClusterInfo{
			Server:                arvanPaasServer,
			InsecureSkipTlsVerify: &insecureSkipTlsVerify,
		},
	}
	kubeConfigData.Clusters = append(kubeConfigData.Clusters, kubeCluster)

	fullUserName := username + "/" + arvanHostnamePort

	if len(projects) > 0 {
		for i := 0; i < len(projects); i++ {
			kubeContext := KubeContext{
				Name: projects[i] + "/" + arvanHostnamePort + "/" + username,
				Context: ContextInfo{
					Cluster:   arvanHostnamePort,
					User:      fullUserName,
					Namespace: &projects[i],
				},
			}
			kubeConfigData.Contexts = append(kubeConfigData.Contexts, kubeContext)
		}
		kubeConfigData.CurrentContext = kubeConfigData.Contexts[0].Name

		// check for current kube config if it has different current context
		currentKubeConfigPtr := loadCurrentKubeConfig(path)
		if currentKubeConfigPtr != nil {
			currentKubeConfig := *currentKubeConfigPtr
			if currentContextExistsAndValid(currentKubeConfig.CurrentContext, kubeConfigData.Contexts) {
				kubeConfigData.CurrentContext = currentKubeConfig.CurrentContext
			}
		}
	} else {
		kubeContext := KubeContext{
			Name: "/" + arvanHostnamePort + "/" + username,
			Context: ContextInfo{
				Cluster: arvanHostnamePort,
				User:    fullUserName,
			},
		}
		kubeConfigData.Contexts = append(kubeConfigData.Contexts, kubeContext)
		kubeConfigData.CurrentContext = "/" + arvanHostnamePort + "/" + username
	}

	user := User{
		Name: fullUserName,
		User: UserInfo{
			Token: token,
		},
	}

	kubeConfigData.Users = append(kubeConfigData.Users, user)

	return kubeConfigData
}

func currentContextExistsAndValid(currentContext string, contexts []KubeContext) bool {
	for _, context := range contexts {
		if currentContext == context.Name {
			return true
		}
	}
	return false
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

func writeKubeConfig(kubeConfig KubeConfig, path string) error {
	kcBytes, err := yaml.Marshal(kubeConfig)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(path, kcBytes, 0644)
	if err != nil {
		return err
	}
	return nil
}
