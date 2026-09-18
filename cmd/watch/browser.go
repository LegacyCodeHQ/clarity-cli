package watch

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os/exec"
	"runtime"
)

// browserCommand returns the OS command and arguments used to open url in the
// default browser on the given GOOS value.
func browserCommand(goos, url string) (string, []string) {
	switch goos {
	case "darwin":
		return "open", []string{url}
	case "windows":
		return "rundll32", []string{"url.dll,FileProtocolHandler", url}
	default:
		return "xdg-open", []string{url}
	}
}

// openBrowser launches the system default browser at url.
func openBrowser(url string) error {
	name, args := browserCommand(runtime.GOOS, url)
	return exec.Command(name, args...).Start()
}

// openOnEnter waits for a single Enter press on in and calls open(url) in
// response. It reacts to at most one press: once the read returns, the
// goroutine exits and further input on in has no effect for the remainder of
// this process.
func openOnEnter(ctx context.Context, in io.Reader, out io.Writer, url string, open func(string) error) {
	scanner := bufio.NewScanner(in)
	if !scanner.Scan() {
		return
	}

	select {
	case <-ctx.Done():
		return
	default:
	}

	if err := open(url); err != nil {
		fmt.Fprintf(out, "Could not open browser automatically: %v\nOpen manually: %s\n", err, url)
		return
	}
	fmt.Fprintf(out, "Opened %s in your browser\n", url)
}
