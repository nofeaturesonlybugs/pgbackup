package main

import (
	"pgbackup/psql"
	"regexp"
)

// Conf specifies configuration for the command.
type Conf struct {
	// Format specifies the backup format.
	Format psql.Format
	//
	// Database names are matched against this regular expression.
	Regexp *regexp.Regexp
	//
	// SplitSize specifies the size in bytes to split SQL parts; if 0 or less
	// then no splitting occurs.
	SplitSize int
}
