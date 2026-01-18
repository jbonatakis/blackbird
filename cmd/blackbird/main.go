package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/jbonatakis/blackbird/internal/cli"
)

func main() {
	if err := cli.Run(os.Args[1:]); err != nil {
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
