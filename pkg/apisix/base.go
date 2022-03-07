package apisix

// Valid Var array:
// ["XXX", "==", "YYY"]
// ["XXX", "in", ["A", "B", "C"]]
// A Var should be string or string array
type Var = []interface{}

// Timeout represents the timeout settings.
// It's worth to note that the timeout is used to control the time duration
// between two successive I/O operations. It doesn't contraint the whole I/O
// operation durations.
type Timeout struct {
	// connect controls the connect timeout in seconds.
	Connect float64 `json:"connect,omitempty"`
	// send controls the send timeout in seconds.
	Send float64 `json:"send,omitempty"`
	// read controls the read timeout in seconds.
	Read float64 `json:"read,omitempty"`
}
