package psql

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"pgbackup/logger"
)

// Format specifies backup format.
type Format int

const (
	// Backups are created in a directory named dbname.backup
	Directory Format = iota
	// Backups are created as a SQL script named dbname.sql
	Script
)

// PSQL is the wrapper to psql, pg_dump, and pg_restore.
type PSQL struct {
	// Directory where backups are stored.
	DirBackups string
	// For backup or restore operations that can run concurrently this specifies the
	// -j argument to pg_dump and pg_restore.
	Jobs int
	//
	logger.Logger
}

// Backup returns the backup destination and the command to execute for backing up a database
// to that destination.
func (p PSQL) Backup(ctx context.Context, dbname string, format Format) (string, *exec.Cmd) {
	var dest string
	var args []string
	binary := "pg_dump"
	if format == Script {
		dest = filepath.Join(p.DirBackups, dbname+".sql")
		args = []string{
			"--column-inserts",
			"-d", dbname,
			"-f", dest,
		}
	} else {
		dest = filepath.Join(p.DirBackups, dbname+".backup")
		args = []string{
			"-Fd",
			"-j", fmt.Sprintf("%v", p.Jobs),
			"-f", dest,
			dbname,
		}
	}
	//
	p.Infof("%v %v", binary, strings.Join(args, " "))
	//
	return dest, exec.CommandContext(ctx, binary, args...)
}

// Create returns the command to execute for creating a database.
func (p PSQL) Create(ctx context.Context, dbname string) *exec.Cmd {
	binary := "psql"
	args := []string{
		"-c",
		"create database \"" + dbname + "\"",
	}
	//
	p.Infof("%v %v", binary, strings.Join(args, " "))
	//
	return exec.CommandContext(ctx, binary, args...)
}

// Drop returns the command to execute for dropping a database.
func (p PSQL) Drop(ctx context.Context, dbname string) *exec.Cmd {
	binary := "psql"
	args := []string{
		"-c",
		"drop database \"" + dbname + "\"",
	}
	//
	p.Infof("%v %v", binary, strings.Join(args, " "))
	//
	return exec.CommandContext(ctx, binary, args...)
}

// Restore returns the data source and command to execute for restoring a database.
//
// Note that when format is Script the src needs to be piped into the commands StdinPipe
// during execution.
func (p PSQL) Restore(ctx context.Context, dbname string, format Format) *exec.Cmd {
	var binary string
	var args []string
	if format == Script {
		binary = "psql"
		args = []string{
			"-d", dbname,
			"-f", filepath.Join(p.DirBackups, dbname+".sql"),
		}
	} else {
		binary = "pg_restore"
		args = []string{
			"-Fd",
			"-j", fmt.Sprintf("%v", p.Jobs),
			"-d", dbname,
			filepath.Join(p.DirBackups, dbname+".backup"),
		}
	}
	//
	p.Infof("%v %v", binary, strings.Join(args, " "))
	//
	return exec.CommandContext(ctx, binary, args...)
}
