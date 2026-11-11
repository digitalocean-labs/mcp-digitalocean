package genai

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
	"time"
)

// fileFetchHTTPClient fetches dataset files from file_url / dataset_file_url.
var fileFetchHTTPClient = &http.Client{Timeout: 2 * time.Minute}

const maxDatasetFileBytes = 50 << 20 // 50 MiB

// fileInputKeys names the mutually exclusive dataset file source arguments.
type fileInputKeys struct {
	Path     string
	Content  string
	URL      string
	FileName string
}

var (
	createDatasetFileKeys = fileInputKeys{
		Path:     "file_path",
		Content:  "file_content",
		URL:      "file_url",
		FileName: "file_name",
	}
	workflowDatasetFileKeys = fileInputKeys{
		Path:     "dataset_file_path",
		Content:  "dataset_file_content",
		URL:      "dataset_file_url",
		FileName: "dataset_file_name",
	}
)

// resolvedFile is dataset bytes plus a file name used for validation and upload.
type resolvedFile struct {
	Data     []byte
	FileName string
	// Display is a human-readable source label for consent prompts.
	Display string
}

func fileSourceProvided(args map[string]any, keys fileInputKeys) bool {
	return strings.TrimSpace(stringArg(args, keys.Path)) != "" ||
		strings.TrimSpace(stringArg(args, keys.Content)) != "" ||
		strings.TrimSpace(stringArg(args, keys.URL)) != ""
}

func fileSourceDisplay(args map[string]any, keys fileInputKeys) string {
	if p := strings.TrimSpace(stringArg(args, keys.Path)); p != "" {
		return p
	}
	if u := strings.TrimSpace(stringArg(args, keys.URL)); u != "" {
		return u
	}
	if name := strings.TrimSpace(stringArg(args, keys.FileName)); name != "" {
		return name
	}
	if strings.TrimSpace(stringArg(args, keys.Content)) != "" {
		return "(inline file_content)"
	}
	return ""
}

// resolveFileInput loads exactly one of path, content, or url from tool args.
func resolveFileInput(ctx context.Context, args map[string]any, keys fileInputKeys) (*resolvedFile, error) {
	pathVal := strings.TrimSpace(stringArg(args, keys.Path))
	contentVal := stringArg(args, keys.Content)
	urlVal := strings.TrimSpace(stringArg(args, keys.URL))
	fileName := strings.TrimSpace(stringArg(args, keys.FileName))

	provided := 0
	if pathVal != "" {
		provided++
	}
	if strings.TrimSpace(contentVal) != "" {
		provided++
	}
	if urlVal != "" {
		provided++
	}
	if provided == 0 {
		return nil, fmt.Errorf("missing dataset file source: provide one of %s, %s, or %s", keys.Path, keys.Content, keys.URL)
	}
	if provided > 1 {
		return nil, fmt.Errorf("provide exactly one of %s, %s, or %s", keys.Path, keys.Content, keys.URL)
	}

	switch {
	case pathVal != "":
		data, err := os.ReadFile(pathVal)
		if err != nil {
			return nil, fmt.Errorf("failed to read %s: %w", keys.Path, err)
		}
		name := fileName
		if name == "" {
			name = getFileName(pathVal)
		}
		return &resolvedFile{Data: data, FileName: name, Display: pathVal}, nil

	case strings.TrimSpace(contentVal) != "":
		if fileName == "" {
			return nil, fmt.Errorf("%s is required when using %s", keys.FileName, keys.Content)
		}
		data := []byte(contentVal)
		if len(data) > maxDatasetFileBytes {
			return nil, fmt.Errorf("%s exceeds maximum size of %d bytes", keys.Content, maxDatasetFileBytes)
		}
		return &resolvedFile{Data: data, FileName: fileName, Display: fileName}, nil

	default:
		data, name, err := fetchFileURL(ctx, urlVal, fileName)
		if err != nil {
			return nil, err
		}
		return &resolvedFile{Data: data, FileName: name, Display: urlVal}, nil
	}
}

func fetchFileURL(ctx context.Context, rawURL, fileName string) ([]byte, string, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return nil, "", fmt.Errorf("invalid file_url: %w", err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, "", fmt.Errorf("file_url must use http or https")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create file_url request: %w", err)
	}

	resp, err := fileFetchHTTPClient.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("failed to fetch file_url: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("failed to fetch file_url: status %d", resp.StatusCode)
	}

	limited := io.LimitReader(resp.Body, maxDatasetFileBytes+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read file_url body: %w", err)
	}
	if len(data) > maxDatasetFileBytes {
		return nil, "", fmt.Errorf("file_url content exceeds maximum size of %d bytes", maxDatasetFileBytes)
	}

	name := fileName
	if name == "" {
		name = getFileName(path.Base(parsed.Path))
	}
	if name == "" || name == "." || name == "/" {
		return nil, "", fmt.Errorf("could not determine file name from file_url; provide file_name")
	}
	return data, name, nil
}
