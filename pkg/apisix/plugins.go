package apisix

type FaultInjectionAbort struct {
	HttpStatus uint32 `json:"http_status,omitempty"`
	Body       string `json:"body,omitempty"`
	Percentage uint32 `json:"percentage,omitempty"`
	Vars       []*Var `json:"vars,omitempty"`
}

type FaultInjectionDelay struct {
	// Duration in seconds
	Duration   int64  `json:"duration,omitempty"`
	Percentage uint32 `json:"percentage,omitempty"`
	Vars       []*Var `json:"vars,omitempty" json:"vars,omitempty"`
}

type FaultInjection struct {
	Abort *FaultInjectionAbort `json:"abort,omitempty"`
	Delay *FaultInjectionDelay `json:"delay,omitempty"`
}
