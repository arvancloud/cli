package cli

import (
	"fmt"
	"git.arvan.me/arvan/cli/pkg/api"
	"git.arvan.me/arvan/cli/pkg/utl"
	"github.com/inconshreveable/go-update"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"

	"github.com/spf13/cobra"

	"github.com/openshift/oc/pkg/helpers/term"

	"git.arvan.me/arvan/cli/pkg/config"
	"git.arvan.me/arvan/cli/pkg/login"
	"git.arvan.me/arvan/cli/pkg/paas"
)

var (
	cliName = "arvan"
	cliLong = `
    Arvan Client

    This client helps you manage your resources in Arvan Cloud Services`

	cliExplain = `
    To use manage any of arvan services run arvan command with your service name:

        arvan paas --help

    This will show manual for managing Arvan Platform as a service.

    To see the full list of commands supported, run 'arvan --help'.`
)

// NewCommandCLI return new cobra cli
func NewCommandCLI() *cobra.Command {
	// Load ConfigInfo from default path if exists
	_, err := config.LoadConfigFile()
	if err != nil {
		log.Println(err)
	}

	in, out, errout := os.Stdin, os.Stdout, os.Stderr
	// Main command
	cmd := &cobra.Command{
		Use:   cliName,
		Short: "Command line tools for managing Arvan services",
		Long:  cliLong,
		Run: func(c *cobra.Command, args []string) {
			explainOut := term.NewResponsiveWriter(out)
			c.SetOutput(explainOut)
			fmt.Fprintf(explainOut, "%s\n\n%s\n", cliLong, cliExplain)
		},
	}

	optionsCommand := newCmdOptions()
	cmd.AddCommand(optionsCommand)

	loginCommand := login.NewCmdLogin(in, out, errout)
	cmd.AddCommand(loginCommand)

	paasCommand := paas.NewCmdPaas()
	cmd.AddCommand(paasCommand)

	cmd.AddCommand(updateCmd())
	return cmd
}

// newCmdOptions implements the OpenShift cli options command
func newCmdOptions() *cobra.Command {
	cmd := &cobra.Command{
		Use: "options",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Usage()
		},
	}

	return cmd
}

// updateCmd updates cli
func updateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update arvan cli",
		Run: func(cmd *cobra.Command, args []string) {
			newVersion, err := api.CheckUpdate()
			utl.CheckErr(err)
			if newVersion == nil {
				fmt.Println("arvan cli is up to date ")
				return
			}
			fmt.Println("update started ...")
			resp, err := http.Get(newVersion.URL)
			if err != nil {
				utl.CheckErr(err)
			}
			defer resp.Body.Close()
			var cliName string
			if runtime.GOOS == "windows" {
				_, err := utl.Unzip(os.TempDir(), resp.Body)
				utl.CheckErr(err)
				cliName = "arvan.exe"
			} else {
				err = utl.Untar(os.TempDir(), resp.Body)
				utl.CheckErr(err)
				cliName = "arvan"
			}

			reader, err := os.Open(filepath.Join(os.TempDir(), cliName))
			utl.CheckErr(err)
			err = update.Apply(reader, update.Options{})
			fmt.Println("update finished successfully :)")
		},
	}
	return cmd
}
