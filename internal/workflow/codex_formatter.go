package workflow

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

type CodexFormatter struct {
	maxLines int
	writer   io.Writer
}

func NewCodexFormatter(maxLines int, writer io.Writer) *CodexFormatter {
	if maxLines < 1 {
		maxLines = 40
	}
	return &CodexFormatter{maxLines: maxLines, writer: writer}
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
		f.writeBlock("Error", firstText(event, "message", "error"))
	case eventType == "item.completed" && itemType == "agent_message":
		f.writeBlock("Agent message", firstText(item, "text", "content"))
	case eventType == "item.completed" && itemType == "command_execution":
		f.writeCommand(item)
	case eventType == "item.completed" && itemType == "file_change":
		f.writeFileChange(item)
	case eventType == "item.completed" && itemType == "reasoning":
		f.writeReasoning(item)
	case eventType == "item.completed" && usefulItemType(itemType):
		f.writeLine(titleFor(itemType))
	case strings.HasPrefix(eventType, "turn.") || strings.HasPrefix(eventType, "thread."):
		return
	}
}

func (f *CodexFormatter) writeCommand(item map[string]interface{}) {
	command := firstText(item, "command", "cmd")
	status := firstText(item, "status")
	if command != "" {
		f.writeLine("Command: " + command)
	}
	if status != "" {
		f.writeLine("Status: " + status)
	}
	f.writeNamedOutput("stdout", firstText(item, "stdout", "output"))
	f.writeNamedOutput("stderr", firstText(item, "stderr"))
}

func (f *CodexFormatter) writeFileChange(item map[string]interface{}) {
	path := firstText(item, "path", "file")
	action := firstText(item, "action", "operation")
	if action == "" {
		action = "changed"
	}
	if path == "" {
		f.writeLine("File change: " + action)
		return
	}
	f.writeLine(fmt.Sprintf("File change: %s %s", action, path))
}

func (f *CodexFormatter) writeReasoning(item map[string]interface{}) {
	summary := firstText(item, "summary", "text")
	if summary == "" {
		f.writeLine("Reasoning")
		return
	}
	f.writeBlock("Reasoning", summary)
}

func (f *CodexFormatter) writeNamedOutput(name, text string) {
	if strings.TrimSpace(text) == "" {
		return
	}
	f.writeLine(name + ":")
	f.writeLimited(text)
}

func (f *CodexFormatter) writeBlock(title, text string) {
	if strings.TrimSpace(text) == "" {
		f.writeLine(title)
		return
	}
	f.writeLine(title + ":")
	f.writeLimited(text)
}

func (f *CodexFormatter) writeLimited(text string) {
	lines := strings.Split(strings.TrimRight(text, "\n"), "\n")
	limit := min(f.maxLines, len(lines))
	for _, line := range lines[:limit] {
		f.writeLine(line)
	}
	if omitted := len(lines) - limit; omitted > 0 {
		f.writeLine(fmt.Sprintf("... truncated %d lines; set codex_output.mode: full for complete Codex output", omitted))
	}
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
	return &CodexFormatterWriter{
		formatter: NewCodexFormatter(maxLines, output),
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
	value, _ := fields[key].(map[string]interface{})
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
