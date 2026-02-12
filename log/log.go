package log

import (
	"context"

	"github.com/pink-tools/pink-otel"
)

type Attr = otel.Attr

var Version = otel.Version

func Init(name, version string)    { otel.Init(name, version) }
func SetServiceNameWidth(w int)    { otel.SetServiceNameWidth(w) }
func PrintServiceLog(line string)  { otel.PrintServiceLog(line) }
func ParseLogMessage(line string) string { return otel.ParseLogMessage(line) }

func Debug(ctx context.Context, body string, attrs ...Attr) { otel.Debug(ctx, body, attrs...) }
func Info(ctx context.Context, body string, attrs ...Attr)  { otel.Info(ctx, body, attrs...) }
func Warn(ctx context.Context, body string, attrs ...Attr)  { otel.Warn(ctx, body, attrs...) }
func Error(ctx context.Context, body string, attrs ...Attr) { otel.Error(ctx, body, attrs...) }
