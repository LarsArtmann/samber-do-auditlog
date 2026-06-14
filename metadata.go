package auditlog

// TypeMetadata provides display metadata (icons, labels, colors) for all enum
// types used in the HTML visualization. It is injected into the template as JSON
// so that JavaScript reads from a single Go-authoritative source instead of
// maintaining parallel hardcoded constants.
type TypeMetadata struct {
	Providers map[string]ProviderMeta `json:"providers"`
	Statuses  map[string]StatusMeta   `json:"statuses"`
	Events    map[string]EventMeta    `json:"events"`
}

// ProviderMeta holds display info for a ProviderType.
type ProviderMeta struct {
	Icon  string `json:"icon"`
	Label string `json:"label"`
}

// StatusMeta holds display info for a ServiceStatus.
type StatusMeta struct {
	Icon string `json:"icon"`
}

// EventMeta holds display info for an EventType.
type EventMeta struct {
	Label string `json:"label"`
	Color string `json:"color"`
}

// BuildTypeMetadata constructs display metadata from the Go enum constants.
// This is the single source of truth — the HTML template's JavaScript reads
// from the injected JSON rather than maintaining parallel constant definitions.
func BuildTypeMetadata() TypeMetadata {
	return TypeMetadata{
		Providers: map[string]ProviderMeta{
			string(ProviderTypeLazy):      {Icon: ProviderTypeLazy.Icon(), Label: "Lazy"},
			string(ProviderTypeEager):     {Icon: ProviderTypeEager.Icon(), Label: "Eager"},
			string(ProviderTypeTransient): {Icon: ProviderTypeTransient.Icon(), Label: "Transient"},
			string(ProviderTypeAlias):     {Icon: ProviderTypeAlias.Icon(), Label: "Alias"},
		},
		Statuses: map[string]StatusMeta{
			string(ServiceStatusRegistered):      {Icon: "\u26AA"},
			string(ServiceStatusActive):          {Icon: "\U0001F7E2"},
			string(ServiceStatusShutdown):        {Icon: "\U0001F535"},
			string(ServiceStatusInvocationError): {Icon: "\U0001F534"},
			string(ServiceStatusShutdownError):   {Icon: "\U0001F534"},
		},
		Events: map[string]EventMeta{
			string(EventTypeRegistration): {Label: "Registration", Color: "var(--accent)"},
			string(EventTypeInvocation):   {Label: "Invocation", Color: "var(--success)"},
			string(EventTypeShutdown):     {Label: "Shutdown", Color: "var(--warning)"},
			string(EventTypeHealthCheck):  {Label: "Health", Color: "var(--info)"},
		},
	}
}
