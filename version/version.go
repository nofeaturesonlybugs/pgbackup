package version

import (
	"fmt"
)

// Version is the application version.
var Version = ""

// Date is the application build date.
var Date = ""

// GoVersion is the Go version.
var GoVersion = ""

// GetVersion returns the version for the package.
func GetVersion(binaryName string) string {
	return binaryName + " version " + Version
}

// GetVersionVerbose returns the version for the package and all of its dependencies.
func GetVersionVerbose(binaryName string) string {
	return GetVersion(binaryName) + fmt.Sprintf("\n\tbuilt with %v\n\tbuilt on %v\n", GoVersion, Date)
}

// init sets vars if empty
func init() {
	if Version == "" {
		Version = "n/a"
	}
	if Date == "" {
		Date = "n/a"
	}
	if GoVersion == "" {
		GoVersion = "n/a"
	}
}
