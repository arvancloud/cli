package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/openshift/origin/pkg/cmd/util/term"

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
	config.LoadConfigFile()

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
		BashCompletionFunction: bashCompletionFunc,
	}


	optionsCommand := newCmdOptions()
	cmd.AddCommand(optionsCommand)

	loginCommand := login.NewCmdLogin(in, out, errout)
	cmd.AddCommand(loginCommand)

	paasCommand := paas.NewCmdPaas(in, out, errout)
	cmd.AddCommand(paasCommand)

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
