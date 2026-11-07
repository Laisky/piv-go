package piv

import (
	"bytes"
	"os"
	"strconv"
	"strings"
	"testing"
)

// TestConnectFreesReaderName verifies that Connect frees the C copy of the
// reader name it passes to SCardConnect. pcsc-lite rejects a name longer than
// MAX_READERNAME before any IPC, so every call below only allocates that copy
// and fails; resident memory grows by the full copies if they leak.
func TestConnectFreesReaderName(t *testing.T) {
	if os.Getenv(nonPIVReaderEnv) == "" {
		t.Skipf("set %s to run tests that need a PC/SC daemon", nonPIVReaderEnv)
	}

	const (
		nameSize = 1 << 20
		connects = 64
	)
	ctx, err := newSCContext()
	if err != nil {
		t.Fatalf("creating context: %v", err)
	}
	defer ctx.Close()

	reader := strings.Repeat("x", nameSize)
	before := residentBytes(t)
	for i := 0; i < connects; i++ {
		h, err := ctx.Connect(reader)
		if err == nil {
			h.Close()
			t.Fatalf("connecting to a %d byte reader name succeeded", nameSize)
		}
	}

	// A leak keeps connects*nameSize (64 MiB) resident; allow half of it for
	// unrelated runtime growth.
	if growth := residentBytes(t) - before; growth > connects*nameSize/2 {
		t.Fatalf("resident memory grew by %d MiB over %d connects with a %d KiB reader name; Connect leaks the name",
			growth>>20, connects, nameSize>>10)
	}
}

// residentBytes returns the process's resident set size from /proc/self/statm.
func residentBytes(t *testing.T) int {
	t.Helper()
	data, err := os.ReadFile("/proc/self/statm")
	if err != nil {
		t.Fatalf("reading statm: %v", err)
	}
	fields := bytes.Fields(data)
	if len(fields) < 2 {
		t.Fatalf("unexpected statm: %q", data)
	}
	pages, err := strconv.Atoi(string(fields[1]))
	if err != nil {
		t.Fatalf("parsing resident pages %q: %v", fields[1], err)
	}
	return pages * os.Getpagesize()
}
