package logger

type nopLogger struct{}

func NewNop() Logger {
	return &nopLogger{}
}

func (n *nopLogger) Info(msg string, fields ...Field)  { _, _ = msg, fields }
func (n *nopLogger) Error(msg string, fields ...Field) { _, _ = msg, fields }
func (n *nopLogger) Debug(msg string, fields ...Field) { _, _ = msg, fields }
func (n *nopLogger) Warn(msg string, fields ...Field)  { _, _ = msg, fields }

func (n *nopLogger) With(fields ...Field) Logger {
	_ = fields
	return n
}
