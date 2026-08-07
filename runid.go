package auditlog

import (
	"crypto/rand"
	"encoding/hex"
)

// runIDBytes is the length of the random portion of a generated run ID, in
// bytes. 16 bytes (128 bits) of crypto-random data matches OpenTelemetry trace
// IDs, so audit run IDs can be correlated with distributed traces.
const runIDBytes = 16

// newRunID returns a random, lowercase hex-encoded run identifier. It is used
// as the default RunID when the caller does not supply one via [Config.RunID].
//
// The value is 32 hex characters (128 bits) of entropy — collision-resistant
// for any practical workload and compatible with observability tooling that
// keys on trace-style IDs.
func newRunID() RunID {
	b := make([]byte, runIDBytes) //nolint:makezero // crypto/rand.Read fills the buffer; zero-init is required
	// crypto/rand.Read is documented to always succeed with a nil error when
	// the slice is non-empty; a failure would indicate a broken entropy source.
	_, _ = rand.Read(b)

	return RunID(hex.EncodeToString(b))
}
