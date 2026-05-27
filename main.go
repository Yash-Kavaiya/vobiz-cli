package main

import (
	"fmt"
	"os"

	"github.com/yash-kavaiya/vobiz-cli/cmd"
	cliErrors "github.com/yash-kavaiya/vobiz-cli/internal/errors"
)

func main() {
	root := cmd.New()
	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(cliErrors.ExitCode(err))
	}
}
