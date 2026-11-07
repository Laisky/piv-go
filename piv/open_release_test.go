package piv

import (
	"errors"
	"os"
	"testing"
)

// nonPIVReaderEnv names a PC/SC reader holding a card that has no PIV applet,
// such as a vsmartcard virtual card. Setting it opts into a non-destructive
// test that connects only to that reader and never touches a YubiKey.
const nonPIVReaderEnv = "PIV_GO_TEST_NON_PIV_READER"

// openFailureAttempts exceeds pcscd's default limit of 200 concurrent client
// contexts (--max-thread), so a leaked context per failed Open exhausts the
// daemon within this loop instead of surviving until process exit.
const openFailureAttempts = 256

// TestOpenReleasesCardOnFailure verifies that an Open which fails after it has
// connected releases the exclusive card handle and the PC/SC context, since it
// returns no YubiKey the caller could close. A leaked handle makes the next
// Open fail with a sharing violation, and leaked contexts make the daemon
// refuse new ones.
func TestOpenReleasesCardOnFailure(t *testing.T) {
	reader := os.Getenv(nonPIVReaderEnv)
	if reader == "" {
		t.Skipf("set %s to a reader holding a card without the PIV applet", nonPIVReaderEnv)
	}

	for i := 1; i <= openFailureAttempts; i++ {
		yk, err := Open(reader)
		if err == nil {
			yk.Close()
			t.Fatalf("Open(%q) succeeded; %s must name a card without the PIV applet", reader, nonPIVReaderEnv)
		}
		if errors.Is(err, ErrNotFound) {
			continue
		}

		var pcscErr *scErr
		if i > 1 && errors.As(err, &pcscErr) && pcscErr.rc == 0x8010000B {
			t.Fatalf("Open attempt %d: an earlier failed Open kept the card connected: %v", i, err)
		}
		t.Fatalf("Open attempt %d: want the PIV applet select to fail with ErrNotFound, got: %v", i, err)
	}
}
