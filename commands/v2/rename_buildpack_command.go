package v2

import (
	"os"

	"code.cloudfoundry.org/cli/cf/cmd"
	"code.cloudfoundry.org/cli/commands/flags"
)

type RenameBuildpackCommand struct {
	RequiredArgs flags.RenameBuildpackArgs `positional-args:"yes"`
}

func (_ RenameBuildpackCommand) Execute(args []string) error {
	cmd.Main(os.Getenv("CF_TRACE"), os.Args)
	return nil
}
