package observability

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"time"
)

func NewLogger(levelRaw, formatRaw string) *slog.Logger {
	var level slog.Level

	switch strings.ToLower(strings.TrimSpace(levelRaw)) {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	var handler slog.Handler
	switch strings.ToLower(strings.TrimSpace(formatRaw)) {
	case "json":
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: level,
			ReplaceAttr: func(_ []string, attr slog.Attr) slog.Attr {
				if attr.Key == slog.TimeKey {
					attr.Value = slog.StringValue(time.Now().UTC().Format(time.RFC3339))
				}
				return attr
			},
		})
	default:
		handler = NewPlainTextHandler(os.Stdout, level)
	}

	return slog.New(handler)
}

type PlainTextHandler struct {
	writer io.Writer
	level  slog.Leveler
	attrs  []slog.Attr
	groups []string
}

func NewPlainTextHandler(writer io.Writer, level slog.Leveler) *PlainTextHandler {
	return &PlainTextHandler{
		writer: writer,
		level:  level,
	}
}

func (h *PlainTextHandler) Enabled(_ context.Context, level slog.Level) bool {
	if h.level == nil {
		return true
	}
	return level >= h.level.Level()
}

func (h *PlainTextHandler) Handle(_ context.Context, record slog.Record) error {
	if !h.Enabled(context.Background(), record.Level) {
		return nil
	}

	builder := strings.Builder{}
	builder.WriteString(record.Time.UTC().Format(time.RFC3339))
	builder.WriteString(" ")
	builder.WriteString(strings.ToUpper(record.Level.String()))
	builder.WriteString(" ")
	builder.WriteString(record.Message)

	allAttrs := make([]slog.Attr, 0, len(h.attrs)+record.NumAttrs())
	allAttrs = append(allAttrs, h.attrs...)
	record.Attrs(func(attr slog.Attr) bool {
		allAttrs = append(allAttrs, attr)
		return true
	})

	flattened := flattenAttrs(h.groups, allAttrs)
	for _, attr := range flattened {
		builder.WriteString(" ")
		builder.WriteString(attr.Key)
		builder.WriteString("=")
		builder.WriteString(formatValue(attr.Value))
	}
	builder.WriteString("\n")

	_, err := io.WriteString(h.writer, builder.String())
	return err
}

func (h *PlainTextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	clone := *h
	clone.attrs = append(append([]slog.Attr{}, h.attrs...), attrs...)
	return &clone
}

func (h *PlainTextHandler) WithGroup(name string) slog.Handler {
	clone := *h
	clone.groups = append(append([]string{}, h.groups...), name)
	return &clone
}

func flattenAttrs(groups []string, attrs []slog.Attr) []slog.Attr {
	if len(attrs) == 0 {
		return nil
	}

	prefix := strings.Join(groups, ".")
	result := make([]slog.Attr, 0, len(attrs))

	for _, attr := range attrs {
		if attr.Value.Kind() == slog.KindGroup {
			groupAttrs := attr.Value.Group()
			groupPrefix := attr.Key
			if prefix != "" {
				groupPrefix = prefix + "." + attr.Key
			}
			result = append(result, flattenAttrs(strings.Split(groupPrefix, "."), groupAttrs)...)
			continue
		}

		key := attr.Key
		if prefix != "" {
			key = prefix + "." + attr.Key
		}
		result = append(result, slog.Attr{Key: key, Value: attr.Value})
	}

	return result
}

func formatValue(value slog.Value) string {
	switch value.Kind() {
	case slog.KindString:
		return value.String()
	case slog.KindInt64:
		return fmt.Sprintf("%d", value.Int64())
	case slog.KindUint64:
		return fmt.Sprintf("%d", value.Uint64())
	case slog.KindFloat64:
		return fmt.Sprintf("%g", value.Float64())
	case slog.KindBool:
		return fmt.Sprintf("%t", value.Bool())
	case slog.KindTime:
		return value.Time().UTC().Format(time.RFC3339)
	case slog.KindDuration:
		return value.Duration().String()
	default:
		return fmt.Sprint(value.Any())
	}
}
