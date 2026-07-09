package middleware

type contextKey string

func (c contextKey) String() string {
	return "middleware context key: " + string(c)
}

const (
	TenantIDKey contextKey = "tenant_id"
	StaffIDKey  contextKey = "staff_id"
)
