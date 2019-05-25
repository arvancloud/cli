package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/openshift/origin/pkg/cmd/util/term"

	"git.arvan.me/arvan/cli/internal/pkg/paas"
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
	out := os.Stdout
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

	paasCommand := paas.NewCmdPaas()

	cmd.AddCommand(paasCommand)
	return cmd
}
