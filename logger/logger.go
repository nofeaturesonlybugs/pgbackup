package logger

// Logger is the basic interface for logging.
type Logger interface {
	// Close the logger if its implementation has open handles or system resources.
	Close()
	// Same as Error() but with fmt.Sprintf() like signature.
	Errorf(fmt string, vals ...interface{})
	// Same as Info() but with fmt.Sprintf() like signature.
	Infof(fmt string, vals ...interface{})
	// Same as Warning() but with fmt.Sprintf() like signature.
	Warningf(fmt string, vals ...interface{})
}

// InfoWriter implements io.Writer and writes message as Infof() calls.
type InfoWriter struct {
	Logger
}

// Write writes the message as an Infof() call.
func (w InfoWriter) Write(p []byte) (int, error) {
	w.Infof(string(p))
	return len(p), nil
}

// WarnWriter implements io.Writer and writes message as Warningf() calls.
type WarnWriter struct {
	Logger
}

// Write writes the message as an Warningf() call.
func (w WarnWriter) Write(p []byte) (int, error) {
	w.Warningf(string(p))
	return len(p), nil
}
