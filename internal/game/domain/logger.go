package domain

// Logger domain 層的日誌介面，避免依賴外部日誌框架
type Logger interface {
	Info(msg string, keysAndValues ...interface{})
	Warn(msg string, keysAndValues ...interface{})
	Error(msg string, keysAndValues ...interface{})
}

// noopLogger 預設的空日誌實作
type noopLogger struct{}

func (noopLogger) Info(string, ...interface{})  {}
func (noopLogger) Warn(string, ...interface{})  {}
func (noopLogger) Error(string, ...interface{}) {}

// NewNoopLogger 建立空日誌（用於測試或預設值）
func NewNoopLogger() Logger {
	return noopLogger{}
}
