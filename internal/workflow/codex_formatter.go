package workflow

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/fatih/color"
)

type CodexFormatter struct {
	maxLines     int
	writer       io.Writer
	colorEnabled bool
}

type CodexFormatterOptions struct {
	MaxLines     int
	ColorEnabled bool
	Writer       io.Writer
}

type codexFileChange struct {
	action string
	path   string
}

func NewCodexFormatter(maxLines int, writer io.Writer) *CodexFormatter {
	return NewCodexFormatterWithOptions(CodexFormatterOptions{
		MaxLines:     maxLines,
		ColorEnabled: true,
		Writer:       writer,
	})
}

func NewCodexFormatterWithOptions(opts CodexFormatterOptions) *CodexFormatter {
	if opts.MaxLines < 1 {
		opts.MaxLines = 40
	}
	return &CodexFormatter{
		maxLines:     opts.MaxLines,
		writer:       opts.Writer,
		colorEnabled: opts.ColorEnabled,
	}
}

func (f *CodexFormatter) ProcessLine(line string) {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return
	}

	var event map[string]interface{}
	if err := json.Unmarshal([]byte(trimmed), &event); err != nil {
		f.writeLine(line)
		return
	}
	f.writeEvent(event)
}

func (f *CodexFormatter) FormatReader(r io.Reader) error {
	scanner := bufio.NewScanner(r)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)
	for scanner.Scan() {
		f.ProcessLine(scanner.Text())
	}
	return scanner.Err()
}

func (f *CodexFormatter) writeEvent(event map[string]interface{}) {
	eventType := stringField(event, "type")
	item := mapField(event, "item")
	itemType := stringField(item, "type")

	switch {
	case eventType == "error":
		f.writeBlock(errorLabel, "Error", firstText(event, "message", "error"))
	case eventType == "item.completed" && itemType == "agent_message":
		f.writeBlock(agentLabel, "Agent message", firstText(item, "text", "content"))
	case eventType == "item.completed" && itemType == "command_execution":
		f.writeCommand(item)
	case eventType == "item.completed" && itemType == "file_change":
		f.writeFileChange(item)
	case eventType == "item.completed" && itemType == "reasoning":
		f.writeReasoning(item)
	case eventType == "item.completed" && usefulItemType(itemType):
		f.writeLine(f.colorize(toolLabel, titleFor(itemType)))
	case strings.HasPrefix(eventType, "turn.") || strings.HasPrefix(eventType, "thread."):
		return
	}
}

func (f *CodexFormatter) writeCommand(item map[string]interface{}) {
	command := firstText(item, "command", "cmd")
	status := firstText(item, "status")
	if command != "" {
		f.writeLine(f.colorize(commandLabel, "Command:") + " " + f.colorize(commandText, command))
	}
	if shouldShowCommandStatus(status) {
		f.writeLine(f.colorize(statusLabel, "Status:") + " " + f.colorize(statusText(status), status))
	}
	f.writeNamedOutput("stdout", firstText(item, "stdout", "output"))
	f.writeNamedOutput("stderr", firstText(item, "stderr"))
}

func (f *CodexFormatter) writeFileChange(item map[string]interface{}) {
	action := firstText(item, "action", "operation", "kind")
	if action == "" {
		action = "changed"
	}
	changes := fileChanges(item, action)
	if len(changes) > 0 {
		for _, change := range changes {
			f.writeLine(fmt.Sprintf("%s %s %s", f.colorize(fileLabel, "File change:"), change.action, f.colorize(filePathText, change.path)))
		}
		return
	}
	f.writeLine(f.colorize(fileLabel, "File change:") + " " + action)
}

func fileChanges(item map[string]interface{}, defaultAction string) []codexFileChange {
	if path := firstText(item, "path", "file", "filename"); path != "" {
		return []codexFileChange{{action: defaultAction, path: path}}
	}
	for _, key := range []string{"change", "file"} {
		if change := fileChangeFromMap(mapField(item, key), defaultAction); change.path != "" {
			return []codexFileChange{change}
		}
	}
	for _, key := range []string{"changes", "files"} {
		if changes := fileChangesFromArray(arrayField(item, key), defaultAction); len(changes) > 0 {
			return changes
		}
	}
	return nil
}

func fileChangesFromArray(values []interface{}, defaultAction string) []codexFileChange {
	changes := make([]codexFileChange, 0, len(values))
	for _, value := range values {
		if path, ok := value.(string); ok && path != "" {
			changes = append(changes, codexFileChange{action: defaultAction, path: path})
			continue
		}
		change := fileChangeFromMap(interfaceMap(value), defaultAction)
		if change.path != "" {
			changes = append(changes, change)
		}
	}
	return changes
}

func fileChangeFromMap(fields map[string]interface{}, defaultAction string) codexFileChange {
	if fields == nil {
		return codexFileChange{}
	}
	action := firstText(fields, "action", "operation", "kind")
	if action == "" {
		action = defaultAction
	}
	return codexFileChange{
		action: action,
		path:   firstText(fields, "path", "file", "filename"),
	}
}

func (f *CodexFormatter) writeReasoning(item map[string]interface{}) {
	summary := firstText(item, "summary", "text")
	if summary == "" {
		f.writeLine(f.colorize(reasoningLabel, "Reasoning"))
		return
	}
	f.writeBlock(reasoningLabel, "Reasoning", summary)
}

func (f *CodexFormatter) writeNamedOutput(name, text string) {
	if strings.TrimSpace(text) == "" {
		return
	}
	f.writeLine(f.colorize(outputLabel(name), name+":"))
	f.writeLimited(text)
}

func (f *CodexFormatter) writeBlock(style *color.Color, title, text string) {
	if strings.TrimSpace(text) == "" {
		f.writeLine(f.colorize(style, title))
		return
	}
	f.writeLine(f.colorize(style, title+":"))
	f.writeLimited(text)
}

func (f *CodexFormatter) writeLimited(text string) {
	lines := strings.Split(strings.TrimRight(text, "\n"), "\n")
	limit := min(f.maxLines, len(lines))
	for _, line := range lines[:limit] {
		f.writeLine(line)
	}
	if omitted := len(lines) - limit; omitted > 0 {
		f.writeLine(f.colorize(truncationText, fmt.Sprintf("... truncated %d lines; set codex_output.mode: full for complete Codex output", omitted)))
	}
}

func (f *CodexFormatter) colorize(style *color.Color, text string) string {
	if !f.colorEnabled {
		return text
	}
	return style.Sprint(text)
}

func (f *CodexFormatter) writeLine(line string) {
	_, _ = io.WriteString(f.writer, line)
	if !strings.HasSuffix(line, "\n") {
		_, _ = io.WriteString(f.writer, "\n")
	}
}

type CodexFormatterWriter struct {
	formatter *CodexFormatter
	buffer    []byte
}

func NewCodexFormatterWriter(maxLines int, output io.Writer) *CodexFormatterWriter {
	return NewCodexFormatterWriterWithOptions(CodexFormatterOptions{
		MaxLines:     maxLines,
		ColorEnabled: true,
		Writer:       output,
	})
}

func NewCodexFormatterWriterWithOptions(opts CodexFormatterOptions) *CodexFormatterWriter {
	return &CodexFormatterWriter{
		formatter: NewCodexFormatterWithOptions(opts),
		buffer:    make([]byte, 0, 4096),
	}
}

func (w *CodexFormatterWriter) Write(p []byte) (int, error) {
	w.buffer = append(w.buffer, p...)
	for {
		idx := indexOfNewline(w.buffer)
		if idx < 0 {
			break
		}
		line := string(w.buffer[:idx])
		w.buffer = w.buffer[idx+1:]
		w.formatter.ProcessLine(line)
	}
	return len(p), nil
}

func (w *CodexFormatterWriter) Flush() {
	if len(w.buffer) == 0 {
		return
	}
	w.formatter.ProcessLine(string(w.buffer))
	w.buffer = w.buffer[:0]
}

func firstText(fields map[string]interface{}, keys ...string) string {
	for _, key := range keys {
		if value := textValue(fields[key]); value != "" {
			return value
		}
	}
	return ""
}

func textValue(value interface{}) string {
	switch v := value.(type) {
	case string:
		return v
	case []interface{}:
		return textFromArray(v)
	case map[string]interface{}:
		return firstText(v, "text", "content", "summary")
	default:
		return ""
	}
}

func textFromArray(values []interface{}) string {
	var parts []string
	for _, value := range values {
		if text := textValue(value); text != "" {
			parts = append(parts, text)
		}
	}
	return strings.Join(parts, "\n")
}

func stringField(fields map[string]interface{}, key string) string {
	if fields == nil {
		return ""
	}
	value, _ := fields[key].(string)
	return value
}

func mapField(fields map[string]interface{}, key string) map[string]interface{} {
	if fields == nil {
		return nil
	}
	return interfaceMap(fields[key])
}

func interfaceMap(value interface{}) map[string]interface{} {
	mapped, _ := value.(map[string]interface{})
	return mapped
}

func arrayField(fields map[string]interface{}, key string) []interface{} {
	if fields == nil {
		return nil
	}
	value, _ := fields[key].([]interface{})
	return value
}

func usefulItemType(itemType string) bool {
	switch itemType {
	case "mcp_call", "web_search", "plan_update":
		return true
	default:
		return false
	}
}

func titleFor(itemType string) string {
	return strings.ReplaceAll(itemType, "_", " ")
}

var (
	agentLabel     = color.New(color.FgCyan, color.Bold)
	commandLabel   = color.New(color.FgMagenta, color.Bold)
	commandText    = color.New(color.FgWhite)
	errorLabel     = color.New(color.FgRed, color.Bold)
	fileLabel      = color.New(color.FgGreen, color.Bold)
	filePathText   = color.New(color.FgCyan)
	reasoningLabel = color.New(color.FgBlue, color.Bold)
	stderrLabel    = color.New(color.FgRed, color.Bold)
	stdoutLabel    = color.New(color.FgHiBlack, color.Bold)
	statusLabel    = color.New(color.FgYellow, color.Bold)
	statusOK       = color.New(color.FgGreen)
	statusWarn     = color.New(color.FgYellow)
	toolLabel      = color.New(color.FgBlue)
	truncationText = color.New(color.FgYellow, color.Faint)
)

func init() {
	for _, style := range []*color.Color{
		agentLabel,
		commandLabel,
		commandText,
		errorLabel,
		fileLabel,
		filePathText,
		reasoningLabel,
		stderrLabel,
		stdoutLabel,
		statusLabel,
		statusOK,
		statusWarn,
		toolLabel,
		truncationText,
	} {
		style.EnableColor()
	}
}

func outputLabel(name string) *color.Color {
	if name == "stderr" {
		return stderrLabel
	}
	return stdoutLabel
}

func statusText(status string) *color.Color {
	switch strings.ToLower(status) {
	case "completed", "success", "succeeded":
		return statusOK
	default:
		return statusWarn
	}
}

func shouldShowCommandStatus(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "", "completed", "success", "succeeded":
		return false
	default:
		return true
	}
}
