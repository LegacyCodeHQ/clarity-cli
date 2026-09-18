package watch

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var errBrowserLaunchFailed = errors.New("exec: \"xdg-open\": executable file not found in $PATH")

func TestBrowserCommand_Darwin(t *testing.T) {
	name, args := browserCommand("darwin", "http://localhost:4900")
	assert.Equal(t, "open", name)
	assert.Equal(t, []string{"http://localhost:4900"}, args)
}

func TestBrowserCommand_Windows(t *testing.T) {
	name, args := browserCommand("windows", "http://localhost:4900")
	assert.Equal(t, "rundll32", name)
	assert.Equal(t, []string{"url.dll,FileProtocolHandler", "http://localhost:4900"}, args)
}

func TestBrowserCommand_Linux(t *testing.T) {
	name, args := browserCommand("linux", "http://localhost:4900")
	assert.Equal(t, "xdg-open", name)
	assert.Equal(t, []string{"http://localhost:4900"}, args)
}

func TestBrowserCommand_UnknownGOOSFallsBackToXDGOpen(t *testing.T) {
	name, args := browserCommand("freebsd", "http://localhost:4900")
	assert.Equal(t, "xdg-open", name)
	assert.Equal(t, []string{"http://localhost:4900"}, args)
}

func TestOpenOnEnter_TriggersOnlyOnce(t *testing.T) {
	in := strings.NewReader("\n\n\n") // multiple buffered Enter presses
	var out bytes.Buffer
	var opens atomic.Int32
	open := func(string) error {
		opens.Add(1)
		return nil
	}

	done := make(chan struct{})
	go func() {
		openOnEnter(context.Background(), in, &out, "http://localhost:0", open)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("openOnEnter did not return")
	}

	// Only the first Enter is consumed; the goroutine exits after it, so the
	// open func must run at most once, regardless of how many newlines were
	// buffered on stdin.
	assert.Equal(t, int32(1), opens.Load())
	assert.Contains(t, out.String(), "Opened http://localhost:0 in your browser")
}

func TestOpenOnEnter_NoInputReturnsWithoutOpening(t *testing.T) {
	in := strings.NewReader("") // EOF immediately, no Enter pressed
	var out bytes.Buffer
	var opens atomic.Int32
	open := func(string) error {
		opens.Add(1)
		return nil
	}

	done := make(chan struct{})
	go func() {
		openOnEnter(context.Background(), in, &out, "http://localhost:0", open)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("openOnEnter did not return on EOF")
	}

	assert.Zero(t, opens.Load())
	assert.Empty(t, out.String())
}

func TestOpenOnEnter_CancelledContextSkipsOpen(t *testing.T) {
	in := strings.NewReader("\n")
	var out bytes.Buffer
	var opens atomic.Int32
	open := func(string) error {
		opens.Add(1)
		return nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	done := make(chan struct{})
	go func() {
		openOnEnter(ctx, in, &out, "http://localhost:0", open)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("openOnEnter did not return")
	}

	assert.Zero(t, opens.Load())
	assert.Empty(t, out.String())
}

func TestOpenOnEnter_OpenFailureReportsFallbackMessage(t *testing.T) {
	in := strings.NewReader("\n")
	var out bytes.Buffer
	open := func(string) error {
		return errBrowserLaunchFailed
	}

	done := make(chan struct{})
	go func() {
		openOnEnter(context.Background(), in, &out, "http://localhost:0", open)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("openOnEnter did not return")
	}

	assert.Contains(t, out.String(), "Could not open browser automatically")
	assert.Contains(t, out.String(), "http://localhost:0")
}
