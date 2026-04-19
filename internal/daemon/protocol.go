package daemon

// Request is the JSON-RPC-lite message body sent by a client to the daemon.
type Request struct {
	Method string         `json:"method"`
	Params map[string]any `json:"params,omitempty"`
}

// Response carries either a result or an error string.
type Response struct {
	OK     bool           `json:"ok"`
	Error  string         `json:"error,omitempty"`
	Result map[string]any `json:"result,omitempty"`
}

const (
	MethodGetSecret    = "get_secret"
	MethodGetConfig    = "get_config"
	MethodList         = "list"
	MethodDescribe     = "describe"
	MethodSetSecret    = "set_secret"
	MethodSetConfig    = "set_config"
	MethodDelete       = "delete"
	MethodStatus       = "status"
	MethodSourceList   = "source_list"
	MethodReloadSource = "reload_source"
	MethodHelp         = "help"
)
