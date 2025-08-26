package constant

const (
	// HeaderUserUIDKey is the context key for the authenticated user.
	HeaderUserUIDKey = "Instill-User-Uid"
	// HeaderRequesterUIDKey is the context key for the requester. An
	// authenticated user can use different namespaces (e.g. an organization
	// they belong to) to make requests, as long as they have permissions.
	HeaderRequesterUIDKey = "Instill-Requester-Uid"
	// HeaderVisitorUIDKey is the context key for the visitor UID when requests
	// are made without authentication.
	HeaderVisitorUIDKey = "Instill-Visitor-Uid"
	// HeaderAuthTypeKey is the context key the authentication type (user or
	// visitor).
	HeaderAuthTypeKey = "Instill-Auth-Type"
	// HeaderUserAgentKey identifies the agent that's making a request. Its
	// accepted values are the string values of
	// github.com/instill-ai/protogen-go/common/run/v1alpha.RunSource.
	HeaderUserAgentKey = "Instill-User-Agent"
	// HeaderServiceKey is the context key for service identity.
	HeaderServiceKey = "Instill-Service"
	// HeaderInstillCodeKey is the context key for shareable link codes.
	HeaderInstillCodeKey = "Instill-Code"
	// ContentTypeJSON is the value for the JSON content type.
	ContentTypeJSON = "application/json"
)
