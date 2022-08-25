package api

// ErrorRecord is the struct to record an error's useful details.
type ErrorRecord struct {
	RecordTs    string `json:"recordTs"`
	Method      string `json:"method"`
	RequestPath string `json:"requestPath"`
	Role        Role   `json:"role"`
	Error       string `json:"error"`
	StackTrace  string `json:"stackTrace"`
}
