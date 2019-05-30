# Arvan CLI

Command line for managing Arvan Services developed using [cobra](https://github.com/spf13/cobra) library.

## Usage

Login to arvan using `arvan login` command.

Type `arvan --help` to get list of all commands.

## Development

### Build

Building of project can be done using `build.sh` script:

```bash
./scripts/build/build.sh ~/go/bin/arvan
```
Where `~/go/bin/arvan` would be output of the project.

#### Environment Variables

`GOEXEC`: path to `go` executable. Change it if you don't want to use `go` executable defined in your PATH.

Example:

```bash
GOEXEC=/usr/lib/go-1.10/bin/go ./scripts/build/build.sh ~/go/bin/arvan
```

`BUILD_TAGS`: add build tags to `go build` process using `-tags` option.

Example:

```bash
BUILD_TAGS="include_gcs include_oss" ./scripts/build/build.sh ~/go/bin/arvan
```

### Subcommand Development:

You can add any subcommand to arvan by developing a [cobra](https://github.com/spf13/cobra) command and add to to arvan as subcommand.

#### Example Subcommand

Develope your subcommand and it's functionalities in separated git repository using [cobra](https://github.com/spf13/cobra). Assume your repository is `github.com/example/examplecli` and it have a `InitiatedCommand` function that returns your base command with type of `github.com/spf13/cobra`.`Command`. Add your project to to this repository vendor using `git submodule`:

```bash
git submodule add -b master https://github.com/example/examplecli vendor/github.com/example/examplecli
```

Create an `example` package in pkg and prepare what you need to for initializing you module in `pkg/example/example.go`:

```go
package example

import (
	example "github.com/example/examplecli"
)

// NewCmdPaas return new cobra cli for paas
func NewCmdExample(in io.Reader, out, errout io.Writer) *cobra.Command {

    exampleCommand := example.InitiatedCommand(in, out, errout)
    
    // Do whatever you need to prepare your command. e.g. login preparation.

	return exampleCommand
}
```

You can access all authentication information and general configurations of `arvan cli` by getting arvan config object using `config.GetConfigInfo()` from `git.arvan.me/arvan/cli/pkg/config`.

After you initialized and prepared your command add it to `arvan cli` as subcommand in `pkg/cli/cli.go`:

```go
package cli

import (
    .
    .
    .
	"git.arvan.me/arvan/cli/pkg/example"
)

func NewCommandCLI() *cobra.Command {
    .
    .
    .

	exampleCommand := example.NewCmdExample(in, out, errout)
	cmd.AddCommand(exampleCommand)

	return cmd
}
```

