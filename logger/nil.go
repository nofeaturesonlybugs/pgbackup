package logger

// Nil implements the Logger interface but every call is a no-op.
var Nil Logger = &_nil{}

// _nil implements the Logger interface and is the concrete type behind Nil.
type _nil struct{}

// Close the logger if its implementation has open handles or system resources.
func (me *_nil) Close() {}

// Same as Error() but with fmt.Sprintf() like signature.
func (me *_nil) Errorf(fmt string, vals ...interface{}) {}

// Same as Info() but with fmt.Sprintf() like signature.
func (me *_nil) Infof(fmt string, vals ...interface{}) {}

// Same as Warning() but with fmt.Sprintf() like signature.
func (me *_nil) Warningf(fmt string, vals ...interface{}) {}
