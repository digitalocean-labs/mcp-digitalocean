package functions

// Models of the OpenWhisk data-plane responses, mirroring the definitions in
// the Apache OpenWhisk API spec that DigitalOcean Functions implements.
//
// These exist only to generate output schemas. No handler decodes into them:
// the tools forward the bytes OpenWhisk returned, so a field missing here
// still reaches the caller. That is why every field is optional and why the
// types err towards `any` wherever OpenWhisk documents a value as "any JSON"
// — the schema should describe what is known without constraining the rest.
//
// The spec distinguishes list and detail views (ActionMeta vs Action,
// ActivationBrief vs Activation), but a list view is a strict subset of its
// detail view, so one type per resource covers both.

// owKeyValue is the spec's KeyValue: a named value of any JSON type, used for
// annotations and parameter bindings throughout.
type owKeyValue struct {
	Key   string `json:"key,omitempty"`
	Value any    `json:"value,omitempty"`
}

// owAction is the spec's Action. Listing actions returns the same shape minus
// the executable's code.
type owAction struct {
	Namespace   string          `json:"namespace,omitempty"`
	Name        string          `json:"name,omitempty"`
	Version     string          `json:"version,omitempty"`
	Publish     bool            `json:"publish,omitempty"`
	Exec        *owActionExec   `json:"exec,omitempty"`
	Annotations []owKeyValue    `json:"annotations,omitempty"`
	Parameters  []owKeyValue    `json:"parameters,omitempty"`
	Limits      *owActionLimits `json:"limits,omitempty"`
	Updated     int64           `json:"updated,omitempty"`
}

// owActionExec is the spec's ActionExec: what the action runs, whether that is
// source code, a container image, or a sequence of other actions.
type owActionExec struct {
	Kind       string   `json:"kind,omitempty"`
	Code       string   `json:"code,omitempty"`
	Image      string   `json:"image,omitempty"`
	Main       string   `json:"main,omitempty"`
	Binary     bool     `json:"binary,omitempty"`
	Components []string `json:"components,omitempty"`
}

// owActionLimits is the spec's ActionLimits.
type owActionLimits struct {
	Timeout     int `json:"timeout,omitempty"`
	Memory      int `json:"memory,omitempty"`
	Logs        int `json:"logs,omitempty"`
	Concurrency int `json:"concurrency,omitempty"`
	Instances   int `json:"instances,omitempty"`
}

// owPackage is the spec's Package.
type owPackage struct {
	Namespace   string            `json:"namespace,omitempty"`
	Name        string            `json:"name,omitempty"`
	Version     string            `json:"version,omitempty"`
	Publish     bool              `json:"publish,omitempty"`
	Annotations []owKeyValue      `json:"annotations,omitempty"`
	Parameters  []owKeyValue      `json:"parameters,omitempty"`
	Binding     *owPackageBinding `json:"binding,omitempty"`
	Actions     []owPackageAction `json:"actions,omitempty"`
	Feeds       []any             `json:"feeds,omitempty"`
	Updated     int64             `json:"updated,omitempty"`
}

// owPackageBinding is the spec's PackageBinding, naming the package this one
// binds to. OpenWhisk sends an empty object for an unbound package.
type owPackageBinding struct {
	Namespace string `json:"namespace,omitempty"`
	Name      string `json:"name,omitempty"`
}

// owPackageAction is the spec's PackageAction: the restricted action view used
// when listing the actions a package contains.
type owPackageAction struct {
	Name        string       `json:"name,omitempty"`
	Version     string       `json:"version,omitempty"`
	Annotations []owKeyValue `json:"annotations,omitempty"`
	Parameters  []owKeyValue `json:"parameters,omitempty"`
}

// owActivation is the spec's Activation, the record of one invocation.
// Listing activations returns the same shape minus the response and logs.
type owActivation struct {
	Namespace    string              `json:"namespace,omitempty"`
	Name         string              `json:"name,omitempty"`
	Version      string              `json:"version,omitempty"`
	Publish      bool                `json:"publish,omitempty"`
	Annotations  []owKeyValue        `json:"annotations,omitempty"`
	Subject      string              `json:"subject,omitempty"`
	ActivationID string              `json:"activationId,omitempty"`
	Start        int64               `json:"start,omitempty"`
	End          int64               `json:"end,omitempty"`
	Duration     int64               `json:"duration,omitempty"`
	Response     *owActivationResult `json:"response,omitempty"`
	Logs         []string            `json:"logs,omitempty"`
	Cause        string              `json:"cause,omitempty"`
	StatusCode   int                 `json:"statusCode,omitempty"`
}

// owActivationResult is the spec's ActivationResult. Result is whatever the
// invoked function returned, so it stays untyped; the keys around it are
// OpenWhisk's own, which is what makes this shape safe to describe.
type owActivationResult struct {
	Status  string `json:"status,omitempty"`
	Result  any    `json:"result,omitempty"`
	Success bool   `json:"success,omitempty"`
	Size    int64  `json:"size,omitempty"`
}

// owActivationLogs is the spec's ActivationLogs: interleaved stdout and stderr.
type owActivationLogs struct {
	Logs []string `json:"logs,omitempty"`
}
