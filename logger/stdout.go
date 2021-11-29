package logger

import (
	"fmt"
	"unicode"
	"unicode/utf8"
)

// STDOut sends messages to stdout.
type STDOut struct{}

// Close release any resources used by the logger.
func (l *STDOut) Close() {}

// Log an error with fmt.Sprintf() like signature.
func (l *STDOut) Errorf(format string, vals ...interface{}) {
	fmt.Println(l.sanitizeStr(fmt.Sprintf("[ERROR] "+format, vals...)))
}

// Log an info with fmt.Sprintf() like signature.
func (l *STDOut) Infof(format string, vals ...interface{}) {
	fmt.Println(l.sanitizeStr(fmt.Sprintf(format, vals...)))
}

// Log a warning with fmt.Sprintf() like signature.
func (l *STDOut) Warningf(format string, vals ...interface{}) {
	fmt.Println(l.sanitizeStr(fmt.Sprintf("[WARN] "+format, vals...)))
}

// sanitizeStr for printing on a console; this means stripping out control
// and non-printable characters so the console doesn't get "ruined."
func (l *STDOut) sanitizeStr(str string) string {
	rv := ""
	for len(str) > 0 {
		r, size := utf8.DecodeRuneInString(str)
		if r == utf8.RuneError {
			str = str[size:]
		} else {
			if unicode.IsPrint(r) || r == '\r' || r == '\n' || r == '\t' {
				rv = rv + string(r)
			} else {
				rv = rv + fmt.Sprintf("\\%02x", string(r))
			}
			str = str[size:]
		}
	}
	return rv
}
