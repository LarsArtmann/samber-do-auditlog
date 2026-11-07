package forward

import (
	"context"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	auditlog "github.com/larsartmann/samber-do-auditlog"
)

// NewWithTarget builds a Forwarder for an explicit target spec (same
// syntax as DO_AUDITLOG_FORWARD_TARGET, comma-separated for fan-out; ""
// arms the default socket).
func NewWithTarget(spec, sourceID string) *Forwarder {
	targets := resolveTargets(spec)
	if len(targets) == 0 {
		return &Forwarder{}
	}

	if sourceID == "" {
		sourceID = os.Getenv(EnvSource)
	}

	if sourceID == "" {
		if executable, err := os.Executable(); err == nil {
			sourceID = filepath.Base(executable)
		} else {
			sourceID = "unknown"
		}
	}

	fwd := &Forwarder{
		targets:  targets,
		apiKey:   os.Getenv(EnvAPIKey),
		sourceID: sourceID,
		batchMax: envInt(EnvBatchMax, defaultMaxBatchEvents, 1, envBatchMaxCeiling),
		flushInterval: time.Duration(
			envInt(EnvFlushMs, int(defaultFlushInterval.Milliseconds()), envFlushMinMs, envFlushMaxMs),
		) * time.Millisecond,
		events:   make(chan auditlog.Event, channelCap),
		complete: make(chan string, completionChanCap),
		stop:     make(chan struct{}),
		stopped:  make(chan struct{}),
	}

	go fwd.loop()

	return fwd
}

// resolveTargets maps a comma-separated target spec onto delivery targets.
// "off"/"disabled" entries drop out; an entirely empty spec arms exactly
// one auto target. Unknown junk falls through to the auto target (same
// tolerance as a bare spec).
func resolveTargets(spec string) []target {
	if strings.TrimSpace(spec) == "" {
		return []target{autoTarget()}
	}

	var targets []target

	for raw := range strings.SplitSeq(spec, ",") {
		one := strings.TrimSpace(raw)
		if one == "" {
			continue
		}

		if t, ok := resolveTarget(one); ok {
			targets = append(targets, t)
		}
	}

	return targets
}

// resolveTarget maps one target entry onto a target, or (!ok) for "off".
func resolveTarget(spec string) (target, bool) {
	switch {
	case strings.EqualFold(spec, "off"), strings.EqualFold(spec, "disabled"):
		return target{}, false

	case strings.HasPrefix(spec, "unix://"):
		path := strings.TrimPrefix(spec, "unix://")

		return unixTarget(path), true

	case strings.HasPrefix(spec, "http://"), strings.HasPrefix(spec, "https://"):
		return target{
			kind:   targetHTTP,
			url:    strings.TrimRight(spec, "/") + ingestPath,
			client: &http.Client{Timeout: requestTimeout},
			active: true,
		}, true

	case strings.HasPrefix(spec, "/"):
		return unixTarget(spec), true

	default:
		return autoTarget(), true
	}
}

func autoTarget() target {
	return target{kind: targetAuto, socketPath: DefaultSocketPath()}
}

func unixTarget(path string) target {
	return target{
		kind:       targetUnix,
		url:        "http://" + unixHost + ingestPath,
		socketPath: path,
		client:     unixClient(path),
		active:     true,
	}
}

// DefaultSocketPath is the conventional PapDashboard audit-run socket,
// mirroring the server's default (runtime dir first, /tmp fallback).
func DefaultSocketPath() string {
	if runtimeDir := os.Getenv("XDG_RUNTIME_DIR"); runtimeDir != "" {
		return filepath.Join(runtimeDir, "papdashboard", "audit-runs.sock")
	}

	return filepath.Join(os.TempDir(), "papdashboard", "audit-runs.sock")
}

// unixClient builds an HTTP client whose dialer ignores the synthetic host
// and connects to the given socket path.
func unixClient(socketPath string) *http.Client {
	return &http.Client{
		Timeout: requestTimeout,
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				dialer := &net.Dialer{Timeout: dialTimeout}

				return dialer.DialContext(ctx, "unix", socketPath)
			},
			MaxIdleConnsPerHost: httpMaxIdleConnsPerHost,
			IdleConnTimeout:     httpIdleConnTimeout,
		},
	}
}

// probeSocket reports whether a unix socket answers a dial.
func probeSocket(path string) bool {
	dialer := &net.Dialer{Timeout: dialTimeout}

	conn, err := dialer.DialContext(context.Background(), "unix", path)
	if err != nil {
		return false
	}

	_ = conn.Close()

	return true
}

// envInt reads a bounded integer knob; missing or invalid values fall back
// to the default, out-of-range values clamp to the bounds.
func envInt(name string, fallback, minValue, maxValue int) int {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return fallback
	}

	value, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}

	return min(max(value, minValue), maxValue)
}

// Enabled reports whether any target is armed (an armed auto target counts
// as enabled — delivery starts when its socket answers).

func targetLabel(t *target) string {
	if t.kind == targetHTTP {
		return t.url
	}

	return t.socketPath
}

// isRootShutdown mirrors the collector's completion derivation: the root
// scope's after-shutdown marks the container run complete.
