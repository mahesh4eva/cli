package main

import (
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path"
	"strings"

	"github.com/cli/cli/command"
	"github.com/cli/cli/context"
	"github.com/cli/cli/update"
	"github.com/cli/cli/utils"
	"github.com/mattn/go-isatty"
	"github.com/mgutz/ansi"
	"github.com/spf13/cobra"
)

var updaterEnabled = ""

func main() {
	currentVersion := command.Version
	updateMessageChan := make(chan *update.ReleaseInfo)
	go func() {
		rel, _ := checkForUpdate(currentVersion)
		updateMessageChan <- rel
	}()

	hasDebug := os.Getenv("DEBUG") != ""

	if cmd, err := command.RootCmd.ExecuteC(); err != nil {
		printError(os.Stderr, err, cmd, hasDebug)
		os.Exit(1)
	}

	newRelease := <-updateMessageChan
	if newRelease != nil {
		msg := fmt.Sprintf("%s %s → %s\n%s",
			ansi.Color("A new release of gh is available:", "yellow"),
			ansi.Color(currentVersion, "cyan"),
			ansi.Color(newRelease.Version, "cyan"),
			ansi.Color(newRelease.URL, "yellow"))

		stderr := utils.NewColorable(os.Stderr)
		fmt.Fprintf(stderr, "\n\n%s\n\n", msg)
	}
}

func printError(out io.Writer, err error, cmd *cobra.Command, debug bool) {
	var dnsError *net.DNSError
	if errors.As(err, &dnsError) {
		fmt.Fprintf(out, "error connecting to %s\n", dnsError.Name)
		if debug {
			fmt.Fprintln(out, dnsError)
		}
		fmt.Fprintln(out, "check your internet connection or githubstatus.com")
		return
	}

	fmt.Fprintln(out, err)

	var flagError *command.FlagError
	if errors.As(err, &flagError) || strings.HasPrefix(err.Error(), "unknown command ") {
		if !strings.HasSuffix(err.Error(), "\n") {
			fmt.Fprintln(out)
		}
		fmt.Fprintln(out, cmd.UsageString())
	}
}

func shouldCheckForUpdate() bool {
	errFd := os.Stderr.Fd()
	return updaterEnabled != "" && (isatty.IsTerminal(errFd) || isatty.IsCygwinTerminal(errFd))
}

func checkForUpdate(currentVersion string) (*update.ReleaseInfo, error) {
	if !shouldCheckForUpdate() {
		return nil, nil
	}

	client, err := command.BasicClient()
	if err != nil {
		return nil, err
	}

	repo := updaterEnabled
	stateFilePath := path.Join(context.ConfigDir(), "state.yml")
	return update.CheckForUpdate(client, stateFilePath, repo, currentVersion)
}
