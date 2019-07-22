package paas

import (
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"crypto/tls"
	"net/url"
	"strings"
	"fmt"

	oc "github.com/openshift/origin/pkg/oc/cli"
	"github.com/spf13/cobra"

	"git.arvan.me/arvan/cli/pkg/config"
)

const (
	kubeConfigFileName = "paasconfig"
	apiKeyHeaderKey    = "Apikey"
	paasUrlInfix       = "/paas/v1/regions/"
	paasUrlPostfix     = "/o/"
	whoAmIPath         = "apis/user.openshift.io/v1/users/~"
	projectListPath    = "apis/project.openshift.io/v1/projects"
)

type whoAmIMetadata struct {
	name string
}

// NewCmdPaas return new cobra cli for paas
func NewCmdPaas(in io.Reader, out, errout io.Writer) *cobra.Command {

	paasCommand := oc.InitiatedCommand("paas", "arvan paas")

	paasCommand.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		err := preparePaasAuthentication(cmd)
		if err != nil {
			fmt.Println(err)
			panic(`Login required. Try "arvan login"`)
		}
	}

	return paasCommand
}

func preparePaasAuthentication(cmd *cobra.Command) error {

	arvanConfig := config.GetConfigInfo()

	if len(arvanConfig.GetApiKey()) == 0 {
		return errors.New("no authorization credentials provided")
	}

	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	username, err := whoAmI()
	if err != nil {
		return err
	}

	projects, err := projectList()

	kubeConfigPath := paasConfigPath()
	setConfigFlag(cmd, kubeConfigPath)

	syncKubeConfig(kubeConfigPath, username, projects)

	return nil
}

func paasConfigPath() string {
	arvanConfig := config.GetConfigInfo()
	homeDir := arvanConfig.GetHomeDir()
	if strings.HasSuffix(homeDir, "/") {
		return homeDir + kubeConfigFileName
	} else {
		return homeDir + "/" + kubeConfigFileName
	}
}

func setConfigFlag(cmd *cobra.Command, kubeConfigPath string) {
	if len(cmd.Flags().Lookup("config").Value.String()) == 0 {
		cmd.Flags().Lookup("config").Value.Set(kubeConfigPath)
	}
}

// #TODO Implement whoAmI
func whoAmI() (string, error) {
	httpReq, err := http.NewRequest("GET", getArvanPaasServerBase()+whoAmIPath, nil)
	httpReq.Header.Add("accept", "application/json")
	httpReq.Header.Add("authorization", getArvanAuthorization())
	httpResp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return "", err
	}

	// read body
	defer httpResp.Body.Close()
	body, err := ioutil.ReadAll(httpResp.Body)

	// parse response
	var objmap map[string]*json.RawMessage
	err = json.Unmarshal(body, &objmap)
	if err != nil {
		return "", err
	}

	if objmap["kind"] != nil {
		var kind string
		err = json.Unmarshal(*objmap["kind"], &kind)
		if err != nil || kind != "User" {
			return "", err
		}
		if kind != "User" {
			return "", errors.New("User kind not supported")
		}
		var v map[string]*string
		err = json.Unmarshal(*objmap["metadata"], &v)
		if err != nil {
			return "", err
		}
		if v["name"] != nil && len(*v["name"]) > 0 {
			return *v["name"], nil
		}
	}

	return "", errors.New("No user.")
}

func projectList() ([]string, error) {
	httpReq, err := http.NewRequest("GET", getArvanPaasServerBase()+projectListPath, nil)
	httpReq.Header.Add("accept", "application/json")
	httpReq.Header.Add("authorization", getArvanAuthorization())
	httpResp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, err
	}

	// read body
	defer httpResp.Body.Close()
	body, err := ioutil.ReadAll(httpResp.Body)

	// parse response
	var objmap map[string]*json.RawMessage

	err = json.Unmarshal(body, &objmap)
	if err != nil {
		return nil, err
	}
	if (objmap["items"] != nil) {
		var projects []*json.RawMessage
		err = json.Unmarshal(*objmap["items"], &projects)
		if err != nil {
			return nil, err
		}
		if len(projects) > 0 {
			var result []string
			var project map[string]*json.RawMessage
			var projectMetadata map[string]*json.RawMessage
			var projectName string
			for i := 0; i < len(projects); i++ {
				err = json.Unmarshal(*projects[i], &project)
				if err != nil || project["metadata"] == nil {
					return nil, errors.New("Invalid projects response")
				}
				err = json.Unmarshal(*project["metadata"], &projectMetadata)
				if err != nil || projectMetadata["name"] == nil {
					return nil, errors.New("Invalid projects response")
				}
				err = json.Unmarshal(*projectMetadata["name"], &projectName)
				if err != nil || projectMetadata["name"] == nil {
					return nil, errors.New("Invalid projects response")
				}
				result = append(result, projectName)
			}
			return result,nil
		} else {
			return nil, nil
		}
	}
	return nil, errors.New("Invalid projects response")
}

func getArvanAuthorization() string {
	arvanConfig := config.GetConfigInfo()
	//#TODO fix this!
	//return apiKeyHeaderKey + " " + arvanConfig.GetApiKey()
	return arvanConfig.GetApiKey()
}

func getArvanPaasServerBase() string {
	arvanConfig := config.GetConfigInfo()
	arvanServer := arvanConfig.GetServer()
	region := arvanConfig.GetRegion()
	return arvanServer + paasUrlInfix + region + paasUrlPostfix
}

func syncKubeConfig(path, username string, projects []string) error {
	arvanConfig := config.GetConfigInfo()
	arvanHostnamePort, err := getArvanServerDomainPort()
	if err != nil {
		return err
	}

	kubeConfig := populateKubeConfig(getArvanPaasServerBase(), arvanHostnamePort, username, arvanConfig.GetApiKey(), projects, path)

	err = writeKubeConfig(kubeConfig, path)
	if err != nil {
		return err
	}
	return nil
}

func getArvanServerDomainPort() (string, error) {
	arvanConfig := config.GetConfigInfo()
	arvanServer := arvanConfig.GetServer()
	u, err := url.Parse(arvanServer)
	if err != nil {
		return "", err
	}
	port := u.Port()
	if len(port) == 0 {
		port = "80"
		if strings.HasPrefix(arvanServer, "https") {
			port = "443"
		}
	}

	hostnameEscaped := strings.Replace(u.Hostname(), ".", "-", -1)

	result := hostnameEscaped + ":" + port
	return result, nil
}
