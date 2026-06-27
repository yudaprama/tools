package builtin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"strings"
	"time"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

// PythonCodeInput defines input for code interpreter tool
type PythonCodeInput struct {
	Code     string   `json:"code" jsonschema:"description=The Python code to execute"`
	Packages []string `json:"packages" jsonschema:"description=Python packages to install before running (e.g. ['pandas'&#44; 'numpy'])"`
}

// ============================================================================
// Response Types (matching frontend expected format)
// ============================================================================

// CodeInterpreterOutput represents a single output item
type CodeInterpreterOutput struct {
	Type string `json:"type"` // "stdout" | "stderr"
	Data string `json:"data"`
}

// CodeInterpreterFileItem represents a generated file
type CodeInterpreterFileItem struct {
	Filename   string `json:"filename"`
	FileId     string `json:"fileId,omitempty"`
	PreviewUrl string `json:"previewUrl,omitempty"`
}

// CodeInterpreterResponse matches frontend CodeInterpreterResponse
type CodeInterpreterResponse struct {
	Result string                    `json:"result,omitempty"`
	Output []CodeInterpreterOutput   `json:"output,omitempty"`
	Files  []CodeInterpreterFileItem `json:"files,omitempty"`
}

// CodeInterpreterService provides Python code execution capabilities
type CodeInterpreterService struct {
	timeout time.Duration
}

// NewCodeInterpreterService creates a new code interpreter service
func NewCodeInterpreterService() *CodeInterpreterService {
	return &CodeInterpreterService{
		timeout: 60 * time.Second,
	}
}

// ExecutePython executes Python code and returns the result
func (s *CodeInterpreterService) ExecutePython(code string, packages []string) (*CodeInterpreterResponse, error) {
	// Install packages if needed (using pip)
	if len(packages) > 0 {
		for _, pkg := range packages {
			// Skip empty packages
			if strings.TrimSpace(pkg) == "" {
				continue
			}

			log.Printf("📦 Installing package: %s", pkg)

			installCmd := exec.Command("pip", "install", "-q", pkg)
			if err := installCmd.Run(); err != nil {
				log.Printf("⚠️  Failed to install %s: %v (continuing anyway)", pkg, err)
			}
		}
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), s.timeout)
	defer cancel()

	// Wrap code to capture result
	wrappedCode := fmt.Sprintf(`
import sys
import io

# Capture stdout
_stdout_capture = io.StringIO()
_stderr_capture = io.StringIO()
_old_stdout = sys.stdout
_old_stderr = sys.stderr
sys.stdout = _stdout_capture
sys.stderr = _stderr_capture

_result = None
try:
    # Execute user code
    exec('''%s''')
    
    # Try to get the last expression result
    _code_lines = '''%s'''.strip().split('\n')
    _last_line = _code_lines[-1].strip() if _code_lines else ''
    if _last_line and not _last_line.startswith(('import ', 'from ', 'def ', 'class ', 'if ', 'for ', 'while ', 'try:', 'with ', '#', 'print(', 'return ')):
        try:
            _result = eval(_last_line)
        except:
            pass
except Exception as e:
    print(str(e), file=sys.stderr)

# Restore stdout/stderr
sys.stdout = _old_stdout
sys.stderr = _old_stderr

# Output results as JSON
import json
print("__RESULT_START__")
print(json.dumps({
    "result": repr(_result) if _result is not None else None,
    "stdout": _stdout_capture.getvalue(),
    "stderr": _stderr_capture.getvalue()
}))
print("__RESULT_END__")
`, escapeCode(code), escapeCode(code))

	// Execute Python
	cmd := exec.CommandContext(ctx, "python3", "-c", wrappedCode)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	// Parse output
	output := stdout.String()
	response := &CodeInterpreterResponse{
		Output: []CodeInterpreterOutput{},
		Files:  []CodeInterpreterFileItem{},
	}

	// Extract structured result
	startMarker := "__RESULT_START__"
	endMarker := "__RESULT_END__"

	if startIdx := strings.Index(output, startMarker); startIdx != -1 {
		if endIdx := strings.Index(output, endMarker); endIdx > startIdx {
			jsonStr := strings.TrimSpace(output[startIdx+len(startMarker) : endIdx])

			var parsed struct {
				Result string `json:"result"`
				Stdout string `json:"stdout"`
				Stderr string `json:"stderr"`
			}

			if jsonErr := json.Unmarshal([]byte(jsonStr), &parsed); jsonErr == nil {
				if parsed.Result != "" && parsed.Result != "None" {
					response.Result = parsed.Result
				}
				if parsed.Stdout != "" {
					response.Output = append(response.Output, CodeInterpreterOutput{
						Type: "stdout",
						Data: parsed.Stdout,
					})
				}
				if parsed.Stderr != "" {
					response.Output = append(response.Output, CodeInterpreterOutput{
						Type: "stderr",
						Data: parsed.Stderr,
					})
				}
			}
		}
	}

	// Add raw stderr if execution failed
	if err != nil {
		stderrStr := stderr.String()
		if stderrStr != "" {
			response.Output = append(response.Output, CodeInterpreterOutput{
				Type: "stderr",
				Data: stderrStr,
			})
		}
	}

	// If no output captured, add raw stdout
	if len(response.Output) == 0 && output != "" {
		// Remove markers from output
		cleanOutput := output
		if idx := strings.Index(cleanOutput, startMarker); idx != -1 {
			cleanOutput = strings.TrimSpace(cleanOutput[:idx])
		}
		if cleanOutput != "" {
			response.Output = append(response.Output, CodeInterpreterOutput{
				Type: "stdout",
				Data: cleanOutput,
			})
		}
	}

	return response, nil
}

// escapeCode escapes code for embedding in Python string
func escapeCode(code string) string {
	// Escape backslashes and single quotes
	code = strings.ReplaceAll(code, "\\", "\\\\")
	code = strings.ReplaceAll(code, "'", "\\'")
	return code
}

// ============================================================================
// Tool Registration
// ============================================================================

// NewCodeInterpreter creates the lobe-code-interpreter tool.
func NewCodeInterpreter(_ context.Context) ([]tool.InvokableTool, error) {
	service := NewCodeInterpreterService()

	pythonTool, err := utils.InferTool("lobe-code-interpreter__python",
		"Execute Python code. Use this to run Python scripts, perform calculations, data analysis, or generate files.",
		func(ctx context.Context, input *PythonCodeInput) (string, error) {
			if input.Code == "" {
				return "", fmt.Errorf("code is required")
			}

			log.Printf("🐍 Executing Python code (%d chars, %d packages)", len(input.Code), len(input.Packages))

			response, err := service.ExecutePython(input.Code, input.Packages)
			if err != nil {
				return "", err
			}

			resultJSON, _ := json.Marshal(response)
			log.Printf("✅ Python execution complete (result: %v, outputs: %d, files: %d)",
				response.Result != "", len(response.Output), len(response.Files))

			return string(resultJSON), nil
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to infer python tool: %w", err)
	}

	return []tool.InvokableTool{pythonTool}, nil
}
