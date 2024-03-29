package cli

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"

	"github.com/arvancloud/cli/pkg/api"
	"github.com/arvancloud/cli/pkg/config"
	"github.com/arvancloud/cli/pkg/paas"
	"github.com/arvancloud/cli/pkg/utl"
	"github.com/inconshreveable/go-update"

	"github.com/openshift/oc/pkg/helpers/term"
	"github.com/spf13/cobra"
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
	_, _ = config.LoadConfigFile()

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

	loginCommand := paas.NewCmdLogin(in, out, errout)
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
			if err != nil {
				update.RollbackError(err)
				fmt.Println(err)
				utl.CheckErr(fmt.Errorf("update failed :("))
			}
			fmt.Println("update finished successfully :)")
		},
	}
	return cmd
}
