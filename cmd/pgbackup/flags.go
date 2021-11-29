package main

// Flags are the command line options.
type Flags struct {
	// Backup all databases.
	Backup bool
	// Clear all backups from backup directory.
	Clear bool
	// Format defines the backup format; one of "dir" or "sql".
	Format string
	// Print help message and exit.
	Help bool
	// Join tells -restore to restore from backup.chunk sources.
	Join bool
	// List databases to backup.
	List bool
	// Regexp used to match databases for backup or restore.
	Regexp string
	// Restore all databases.
	Restore bool
	// Specifies the size when splitting backup.sql files.
	Split string
	// Print commands as they are executed.
	Verbose bool
	// Print version information and exit.
	Version bool
	// Any remaining flags after parsing.
	Remaining []string
}
