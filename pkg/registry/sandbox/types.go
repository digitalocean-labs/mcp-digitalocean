package sandbox

import (
	"strconv"
	"strings"
	"time"

	"mcp-digitalocean/pkg/registry/common"
)

// int64String decodes the API's string-encoded 64-bit integers ("123") and
// accepts bare numbers too. It encodes as a JSON number, so the published
// output schema (integer) matches what the tool returns.
type int64String int64

func (n *int64String) UnmarshalJSON(b []byte) error {
	if string(b) == "null" {
		return nil
	}
	v, err := strconv.ParseInt(strings.Trim(string(b), `"`), 10, 64)
	if err != nil {
		return err
	}
	*n = int64String(v)
	return nil
}

// Sandbox is a microVM sandbox.
type Sandbox struct {
	SandboxID string    `json:"sandbox_id"`
	Template  string    `json:"template"`
	State     string    `json:"state"`
	SizeSlug  string    `json:"size_slug,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// Snapshot is a point-in-time capture of a sandbox.
type Snapshot struct {
	SnapshotID   string      `json:"snapshot_id"`
	SandboxID    string      `json:"sandbox_id"`
	Name         string      `json:"name,omitempty"`
	State        string      `json:"state"`
	SizeBytes    int64String `json:"size_bytes,omitempty"`
	CreatedAt    time.Time   `json:"created_at"`
	ErrorMessage string      `json:"error_message,omitempty"`
	TTL          string      `json:"ttl,omitempty"`
}

// Template is a team sandbox template.
type Template struct {
	TemplateID string         `json:"template_id"`
	Name       string         `json:"name"`
	Spec       map[string]any `json:"spec,omitempty"`
	Status     string         `json:"status"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
}

// TemplateBuild is one build of a sandbox template.
type TemplateBuild struct {
	BuildID    string         `json:"build_id"`
	TemplateID string         `json:"template_id"`
	Name       string         `json:"name,omitempty"`
	Spec       map[string]any `json:"spec,omitempty"`
	Status     string         `json:"status"`
	Error      string         `json:"error,omitempty"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
}

// Command is a process started inside a sandbox.
type Command struct {
	PID int64  `json:"pid"`
	Cmd string `json:"cmd"`
	Cwd string `json:"cwd,omitempty"`
}

// FileEntry describes one file or directory in a sandbox.
type FileEntry struct {
	Name          string      `json:"name"`
	Type          string      `json:"type"`
	Path          string      `json:"path"`
	Size          int64String `json:"size"`
	Mode          int64       `json:"mode"`
	Permissions   string      `json:"permissions,omitempty"`
	Owner         string      `json:"owner,omitempty"`
	Group         string      `json:"group,omitempty"`
	ModifiedTime  time.Time   `json:"modified_time"`
	SymlinkTarget string      `json:"symlink_target,omitempty"`
}

// --- Request bodies ---

type createSandboxRequest struct {
	Template            string            `json:"template"`
	SizeSlug            string            `json:"size_slug"`
	AllowInternetAccess *bool             `json:"allow_internet_access,omitempty"`
	EnvVars             map[string]string `json:"env_vars,omitempty"`
}

type createSnapshotRequest struct {
	Name string `json:"name,omitempty"`
	TTL  string `json:"ttl,omitempty"`
}

type templateCreateRequest struct {
	Name         string `json:"name"`
	BaseTemplate string `json:"base_template"`
	SourceOCIRef string `json:"source_oci_ref"`
}

type templateUpdateRequest struct {
	BaseTemplate string `json:"base_template,omitempty"`
	SourceOCIRef string `json:"source_oci_ref,omitempty"`
}

type commandStartRequest struct {
	Cmd            string            `json:"cmd"`
	Args           []string          `json:"args,omitempty"`
	Cwd            string            `json:"cwd,omitempty"`
	EnvVars        map[string]string `json:"env_vars,omitempty"`
	Stdin          bool              `json:"stdin,omitempty"`
	TimeoutSeconds int64             `json:"timeout_seconds,string,omitempty"`
}

type commandSignalRequest struct {
	Signal string `json:"signal"`
}

type commandStdinRequest struct {
	Data string `json:"data"`
	EOF  bool   `json:"eof,omitempty"`
}

type execRequest struct {
	Argv           []string          `json:"argv"`
	Workdir        string            `json:"workdir,omitempty"`
	EnvVars        map[string]string `json:"env_vars,omitempty"`
	TimeoutSeconds int64             `json:"timeout_seconds,string,omitempty"`
	Stdin          string            `json:"stdin,omitempty"`
}

type fileRemoveRequest struct {
	Path string `json:"path"`
}

type fileMakeDirectoryRequest struct {
	Path string `json:"path"`
}

type fileMoveRequest struct {
	Source      string `json:"source"`
	Destination string `json:"destination"`
}

// --- Response envelopes ---

type sandboxResponse struct {
	Sandbox Sandbox `json:"sandbox"`
}

type sandboxListResponse struct {
	Sandboxes     []Sandbox `json:"sandboxes"`
	NextPageToken string    `json:"next_page_token,omitempty"`
}

type snapshotResponse struct {
	Snapshot Snapshot `json:"snapshot"`
}

type snapshotListResponse struct {
	Snapshots     []Snapshot `json:"snapshots"`
	NextPageToken string     `json:"next_page_token,omitempty"`
}

type templateResponse struct {
	Template Template `json:"template"`
}

type templateListResponse struct {
	Templates     []Template `json:"templates"`
	NextPageToken string     `json:"next_page_token,omitempty"`
}

type templateBuildResponse struct {
	Build TemplateBuild `json:"build"`
}

type templateBuildListResponse struct {
	Builds        []TemplateBuild `json:"builds"`
	NextPageToken string          `json:"next_page_token,omitempty"`
}

type commandListResponse struct {
	Commands []Command `json:"commands"`
}

type commandStartResponse struct {
	PID int64 `json:"pid"`
}

type fileListResponse struct {
	Entries []FileEntry `json:"entries"`
}

type fileInfoResponse struct {
	Entry FileEntry `json:"entry"`
}

type fileExistsResponse struct {
	Exists bool `json:"exists"`
}

type fileMakeDirectoryResponse struct {
	Created bool      `json:"created"`
	Entry   FileEntry `json:"entry"`
}

type fileMoveResponse struct {
	Entry FileEntry `json:"entry"`
}

type fileUploadResponse struct {
	Path         string      `json:"path"`
	BytesWritten int64String `json:"bytes_written"`
}

// FileDownloadResult is a file read from a sandbox. Content is UTF-8 text when
// the payload is valid UTF-8, and base64 otherwise (Encoding says which).
type FileDownloadResult struct {
	Path      string `json:"path"`
	SizeBytes int    `json:"size_bytes"`
	SHA256    string `json:"sha256"`
	Encoding  string `json:"encoding"`
	Content   string `json:"content"`
}

// ExecResult is the outcome of a one-shot command run with sandbox exec.
type ExecResult struct {
	ExecID          string `json:"exec_id,omitempty"`
	ExitCode        int32  `json:"exit_code"`
	DurationMS      int64  `json:"duration_ms,omitempty"`
	Status          string `json:"status,omitempty"`
	Error           string `json:"error,omitempty"`
	Stdout          string `json:"stdout"`
	Stderr          string `json:"stderr"`
	StdoutTruncated bool   `json:"stdout_truncated,omitempty"`
	StderrTruncated bool   `json:"stderr_truncated,omitempty"`
}

// --- Output schemas ---

var (
	sandboxOut      = common.NewObjectOutput[sandboxResponse]()
	sandboxListOut  = common.NewObjectOutput[sandboxListResponse]()
	snapshotOut     = common.NewObjectOutput[snapshotResponse]()
	snapshotListOut = common.NewObjectOutput[snapshotListResponse]()
	templateOut     = common.NewObjectOutput[templateResponse]()
	templateListOut = common.NewObjectOutput[templateListResponse]()
	buildOut        = common.NewObjectOutput[templateBuildResponse]()
	buildListOut    = common.NewObjectOutput[templateBuildListResponse]()
	commandListOut  = common.NewObjectOutput[commandListResponse]()
	commandStartOut = common.NewObjectOutput[commandStartResponse]()
	execOut         = common.NewObjectOutput[ExecResult]()
	fileListOut     = common.NewObjectOutput[fileListResponse]()
	fileInfoOut     = common.NewObjectOutput[fileInfoResponse]()
	fileExistsOut   = common.NewObjectOutput[fileExistsResponse]()
	fileDownloadOut = common.NewObjectOutput[FileDownloadResult]()
	fileUploadOut   = common.NewObjectOutput[fileUploadResponse]()
	fileMkdirOut    = common.NewObjectOutput[fileMakeDirectoryResponse]()
	fileMoveOut     = common.NewObjectOutput[fileMoveResponse]()
)
