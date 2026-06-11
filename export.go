package auditlog

import (
	"fmt"
	"io"
)

func writeSortedLines(writer io.Writer, lines []string) error {
	for _, line := range lines {
		_, err := fmt.Fprintln(writer, line)
		if err != nil {
			return fmt.Errorf("write line: %w", err)
		}
	}

	return nil
}

func serviceLabel(svc ServiceInfo) string {
	name := svc.ServiceName

	if svc.ServiceType.IsKnown() {
		name += " " + svc.ServiceType.Icon()
	}

	return name
}

func serviceRefLabel(ref ServiceRef) string {
	return ref.ServiceName
}
