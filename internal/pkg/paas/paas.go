package paas

import (
	oc "github.com/openshift/origin/pkg/oc/cli"
	"github.com/spf13/cobra"
)

// NewCmdPaas return new cobra cli for paas
func NewCmdPaas() *cobra.Command {
	paasCommand := oc.InitiatedCommand("paas", "arvan paas")

	return paasCommand
}
