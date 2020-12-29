package main

import (
	"math/rand"
	"os"
	"runtime"
	"time"

	"github.com/arvancloud/cli/pkg/cli"
)

func main() {

	rand.Seed(time.Now().UTC().UnixNano())
	if len(os.Getenv("GOMAXPROCS")) == 0 {
		runtime.GOMAXPROCS(runtime.NumCPU())
	}

	command := cli.NewCommandCLI()
	if err := command.Execute(); err != nil {
		os.Exit(1)
	}
}
