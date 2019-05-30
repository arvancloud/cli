package paas

import (
	"errors"
	"fmt"
	"io"
	"os"

	oc "github.com/openshift/origin/pkg/oc/cli"
	"github.com/spf13/cobra"

	"git.arvan.me/arvan/cli/pkg/config"
)

const kubeConfigFileName = "/paasconfig"

// NewCmdPaas return new cobra cli for paas
func NewCmdPaas(in io.Reader, out, errout io.Writer) *cobra.Command {

	paasCommand := oc.InitiatedCommand("paas", "arvan paas")

	paasCommand.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		fmt.Println(cmd.Flags().Lookup("config").Value)
		preparePaasAuthentication(cmd)
	}

	return paasCommand
}

func preparePaasAuthentication(cmd *cobra.Command) error {

	kubeConfigPath := paasConfigPath()
	arvanConfig := config.GetConfigInfo()

	if len(arvanConfig.GetApiKey()) == 0 {
		return errors.New("no authorization credentials provided")
	} else {
		loginRequired := false
		if _, err := os.Stat(kubeConfigPath); os.IsNotExist(err) {
			loginRequired = true
		} else {
			authorized, err := userIsAuthorized(kubeConfigPath)

			if err != nil {
				return err
			}

			loginRequired = !authorized
		}

		if loginRequired {
			initiateLogin(cmd)
		}
	}
	setConfigFlag(cmd, kubeConfigPath)
	return nil
}

func paasConfigPath() string {
	arvanConfig := config.GetConfigInfo()
	return arvanConfig.GetHomeDir() + "/" + kubeConfigFileName
}

func setConfigFlag(cmd *cobra.Command, kubeConfigPath string) {
	if len(cmd.Flags().Lookup("config").Value.String()) == 0 {
		cmd.Flags().Lookup("config").Value.Set(kubeConfigPath)
	}
}

// #TODO Implement userIsAuthorized
func userIsAuthorized(kubeConfigPath string) (bool, error) {
	if len(kubeConfigPath) > 0 {
		return true, nil
	} else {
		return false, errors.New("No kubeconfig provided.")
	}
}

// #TODO implement initiateLogin, do not use openshift authorization flow
func initiateLogin(cmd *cobra.Command) {
	//#TODO do not outout to os stdout
	in, out, errout := os.Stdin, os.Stdout, os.Stderr
	kubeConfigPath := paasConfigPath()
	arvanConfig := config.GetConfigInfo()
	setConfigFlag(cmd, kubeConfigPath)
	//#TODO disable insecure tls
	oc.InitiateLogin(arvanConfig.GetServer(), "apikey", arvanConfig.GetApiKey(), true, cmd, in, out, errout)
}
