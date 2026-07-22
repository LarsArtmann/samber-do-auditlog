package auditlog

import (
	"encoding/json"
	"fmt"
	"io"
)

// writeEventsNDJSON encodes each event as a line-delimited JSON object to writer.
// Shared by Plugin.WriteEventsNDJSON and Report.WriteNDJSON to keep error
// wrapping consistent across both entry points.
func writeEventsNDJSON(writer io.Writer, events []Event) error {
	enc := json.NewEncoder(writer)

	for _, event := range events {
		err := enc.Encode(event)
		if err != nil {
			return fmt.Errorf("encode event %d: %w", event.Sequence, err)
		}
	}

	return nil
}

// serviceLabel returns a service name with its provider-type icon, if known.
func serviceLabel(svc ServiceInfo) string {
	name := string(svc.ServiceName)

	if svc.ServiceType.IsKnown() {
		name += " " + svc.ServiceType.Icon()
	}

	return name
}
