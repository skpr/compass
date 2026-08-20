package events

// ConnectionState describes the state of the connection to a trace stream.
type ConnectionState string

const (
	// ConnectionStateConnecting is set while we are establishing a connection.
	ConnectionStateConnecting ConnectionState = "connecting"
	// ConnectionStateConnected is set while traces are streaming.
	ConnectionStateConnected ConnectionState = "connected"
	// ConnectionStateRetrying is set while we are waiting to reconnect.
	ConnectionStateRetrying ConnectionState = "retrying"
)

// Connection is sent when the state of the trace stream changes.
type Connection struct {
	State ConnectionState
	Err   error
}
