package api

// Debug is the API message for debug info.
type Debug struct {
	IsDebug bool `jsonapi:"attr,isDebug"`
}

// DebugPatch is the API message for patching debug info.
type DebugPatch struct {
	IsDebug bool `jsonapi:"attr,isDebug"`
}

// DebugLog is the API message for debug log.
type DebugLog struct {
	ID          int    `jsonapi:"primary,log"`
	RecordTs    int64  `jsonapi:"attr,recordTs"`
	Method      string `jsonapi:"attr,method"`
	RequestPath string `jsonapi:"attr,requestPath"`
	Role        Role   `jsonapi:"attr,role"`
	Error       string `jsonapi:"attr,error"`
	StackTrace  string `jsonapi:"attr,stackTrace"`
}
