package types

// RaygunApplication represents a Raygun application/project
type RaygunApplication struct {
	Identifier string `json:"identifier"`
	Name       string `json:"name"`
}

// ErrorGroup represents a group of similar errors in Raygun
type ErrorGroup struct {
	Identifier string `json:"identifier"`
	Message    string `json:"message"`
	Status     string `json:"status"`
	Count      int    `json:"count"`
}

// CrashReportDetail contains detailed information about a crash
type CrashReportDetail struct {
	Error   ErrorInfo   `json:"error"`
	Request RequestInfo `json:"request"`
}

// ErrorInfo contains error details
type ErrorInfo struct {
	Message    string       `json:"message"`
	ClassName  string       `json:"className"`
	StackTrace []StackFrame `json:"stackTrace"`
}

// StackFrame represents a single frame in a stack trace
type StackFrame struct {
	FileName   string `json:"fileName"`
	LineNumber int    `json:"lineNumber"`
	MethodName string `json:"methodName"`
}

// RequestInfo contains HTTP request information
type RequestInfo struct {
	URL    string `json:"url"`
	Method string `json:"httpMethod"`
}
