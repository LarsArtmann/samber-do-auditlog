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
			string(ProviderTypeLazy):      {Icon: ProviderTypeLazy.Icon(), Label: ProviderTypeLazy.Label()},
			string(ProviderTypeEager):     {Icon: ProviderTypeEager.Icon(), Label: ProviderTypeEager.Label()},
			string(ProviderTypeTransient): {Icon: ProviderTypeTransient.Icon(), Label: ProviderTypeTransient.Label()},
			string(ProviderTypeAlias):     {Icon: ProviderTypeAlias.Icon(), Label: ProviderTypeAlias.Label()},
		},
		Statuses: map[string]StatusMeta{
			string(ServiceStatusRegistered):      {Icon: ServiceStatusRegistered.Icon()},
			string(ServiceStatusActive):          {Icon: ServiceStatusActive.Icon()},
			string(ServiceStatusShutdown):        {Icon: ServiceStatusShutdown.Icon()},
			string(ServiceStatusInvocationError): {Icon: ServiceStatusInvocationError.Icon()},
			string(ServiceStatusShutdownError):   {Icon: ServiceStatusShutdownError.Icon()},
		},
		Events: map[string]EventMeta{
			string(EventTypeRegistration): {Label: EventTypeRegistration.Label(), Color: EventTypeRegistration.Color()},
			string(EventTypeInvocation):   {Label: EventTypeInvocation.Label(), Color: EventTypeInvocation.Color()},
			string(EventTypeShutdown):     {Label: EventTypeShutdown.Label(), Color: EventTypeShutdown.Color()},
			string(EventTypeHealthCheck):  {Label: EventTypeHealthCheck.Label(), Color: EventTypeHealthCheck.Color()},
		},
	}
}
