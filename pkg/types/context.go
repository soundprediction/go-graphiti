package types

type ContextKey string

const (
	ContextKeyUserID          ContextKey = "user_id"
	ContextKeySessionID       ContextKey = "session_id"
	ContextKeyRequestSource   ContextKey = "request_source"
	ContextKeyIngestionSource ContextKey = "ingestion_source"
	ContextKeySystemCall      ContextKey = "system_call"
)
