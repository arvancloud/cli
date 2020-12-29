package paas

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/arvancloud/cli/pkg/api"
	"github.com/openshift/oc/pkg/version"
	"io/ioutil"
	"k8s.io/client-go/rest"
	"net/http"
	"net/url"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/arvancloud/cli/pkg/oc"
	"github.com/spf13/cobra"

	"github.com/arvancloud/cli/pkg/config"
	"github.com/arvancloud/cli/pkg/utl"
)

const (
	kubeConfigFileName = "paasconfig"
	paasUrlInfix       = "/paas/v1/regions/"
	paasUrlPostfix     = "/o/"
	whoAmIPath         = "apis/user.openshift.io/v1/users/~"
	projectListPath    = "apis/project.openshift.io/v1/projects"
)

// NewCmdPaas return new cobra cli for paas
func NewCmdPaas() *cobra.Command {

	paasCommand := oc.InitiatedCommand()

	paasCommand.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		err := preparePaasAuthentication(cmd)
		utl.CheckErr(err)
		update, err := api.CheckUpdate()
		if err != nil {
			return
		}
		if update != nil {
			w := new(tabwriter.Writer)
			w.Init(os.Stdout, 4, 4, 0, '\t', 0)
			defer w.Flush()
			currentVersionInfo, _, err := version.ExtractVersion()
			utl.CheckErr(err)
			fmt.Fprint(w, strings.Repeat("*", 50))
			fmt.Fprint(w, fmt.Sprintf("\n\tUpdate available %s -> %s\t", currentVersionInfo.GitVersion, update.Version))
			fmt.Fprint(w, "\n\tRun 'arvan update' to update\n")
			fmt.Fprint(w, strings.Repeat("*", 50))
			fmt.Fprint(w, "\n")
		}
	}

	return paasCommand
}

func preparePaasAuthentication(cmd *cobra.Command) error {

	arvanConfig := config.GetConfigInfo()

	if len(arvanConfig.GetApiKey()) == 0 {
		return errors.New("no authorization credentials provided. \nTry \"arvan login\"")
	}

	// #TODO do not use InsecureSkipVerify
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	username, httpStatusCode, err := whoAmI()
	if err != nil {
		if httpStatusCode == 401 {
			return fmt.Errorf("%v\n%s", err, `Try "arvan login".`)
		}
		if httpStatusCode >= 500 {
			return fmt.Errorf("%v\n%s", err, `Please try again later`)
		}
		return err
	}

	projects, err := projectList()
	if err != nil {
		return err
	}
	if len(projects) == 0 && cmd.Name() != "new-project" {
		return errors.New("no project found. \n To get started create new project using \"arvan paas new-project NAME\".")
	}

	kubeConfigPath := paasConfigPath()
	err = setConfigFlag(cmd, kubeConfigPath)
	if err != nil {
		return err
	}
	err = syncKubeConfig(kubeConfigPath, username, projects)
	if err != nil {
		return err
	}
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

func setConfigFlag(cmd *cobra.Command, kubeConfigPath string) error {
	if len(cmd.Flags().Lookup("kubeconfig").Value.String()) == 0 {
		return cmd.Flags().Lookup("kubeconfig").Value.Set(kubeConfigPath)
	}
	return nil
}

// #TODO Implement whoAmI
func whoAmI() (string, int, error) {
	httpReq, err := http.NewRequest("GET", getArvanPaasServerBase()+whoAmIPath, nil)
	if err != nil {
		return "", 0, err
	}
	httpReq.Header.Add("accept", "application/json")
	httpReq.Header.Add("authorization", getArvanAuthorization())
	httpReq.Header.Add("User-Agent", rest.DefaultKubernetesUserAgent())
	httpResp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return "", 0, err
	}

	if httpResp.StatusCode != 200 {
		return "", httpResp.StatusCode, fmt.Errorf(httpResp.Status)
	}

	// read body
	defer httpResp.Body.Close()
	body, err := ioutil.ReadAll(httpResp.Body)
	if err != nil {
		return "", httpResp.StatusCode, err
	}

	// parse response
	var objmap map[string]*json.RawMessage
	err = json.Unmarshal(body, &objmap)
	if err != nil {
		return "", httpResp.StatusCode, err
	}

	if objmap["kind"] != nil {
		var kind string
		err = json.Unmarshal(*objmap["kind"], &kind)
		if err != nil || kind != "User" {
			return "", httpResp.StatusCode, err
		}
		if kind != "User" {
			return "", httpResp.StatusCode, errors.New("User kind not supported")
		}
		var metadata map[string]*json.RawMessage
		err = json.Unmarshal(*objmap["metadata"], &metadata)
		if err != nil {
			return "", httpResp.StatusCode, err
		}
		if metadata["name"] != nil {
			var userName string
			err = json.Unmarshal(*metadata["name"], &userName)
			if err != nil {
				return "", httpResp.StatusCode, err
			}
			return userName, httpResp.StatusCode, nil
		}
	}

	return "", httpResp.StatusCode, errors.New("invalid authentication credentials.")
}

func projectList() ([]string, error) {
	httpReq, err := http.NewRequest("GET", getArvanPaasServerBase()+projectListPath, nil)
	if err != nil {
		return nil, err
	}
	httpReq.Header.Add("accept", "application/json")
	httpReq.Header.Add("authorization", getArvanAuthorization())
	httpReq.Header.Add("User-Agent", rest.DefaultKubernetesUserAgent())
	httpResp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, err
	}

	// read body
	defer httpResp.Body.Close()
	body, err := ioutil.ReadAll(httpResp.Body)
	if err != nil {
		return nil, err
	}
	// parse response
	var objmap map[string]*json.RawMessage

	err = json.Unmarshal(body, &objmap)
	if err != nil {
		return nil, err
	}
	if objmap["items"] != nil {
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
			return result, nil
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
