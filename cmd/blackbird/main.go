package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/jbonatakis/blackbird/internal/cli"
)

// Version is set at build time via -ldflags "-X main.Version=..."
var Version string

func main() {
	args := os.Args[1:]
	if len(args) == 1 && (args[0] == "--version" || args[0] == "-V") {
		if Version == "" {
			Version = "(devel)"
		}
		fmt.Println(Version)
		os.Exit(0)
	}
	if err := cli.Run(args); err != nil {
		var ue cli.UsageError
		if errors.As(err, &ue) {
			fmt.Fprintln(os.Stderr, ue.Error())
			fmt.Fprintln(os.Stderr)
			fmt.Fprintln(os.Stderr, cli.Usage())
			os.Exit(2)
		}
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}
