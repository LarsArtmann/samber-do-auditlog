package auditlog

// serviceLabel returns a service name with its provider-type icon, if known.
func serviceLabel(svc ServiceInfo) string {
	name := svc.ServiceName

	if svc.ServiceType.IsKnown() {
		name += " " + svc.ServiceType.Icon()
	}

	return name
}

// serviceRefLabel returns the service name from a reference for diagram labels.
func serviceRefLabel(ref ServiceRef) string {
	return ref.ServiceName
}
