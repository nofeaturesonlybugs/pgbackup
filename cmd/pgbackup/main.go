package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"pgbackup/logger"
	"pgbackup/psql"

	"github.com/dustin/go-humanize"
)

const (
	// Backup and restore uses Postgres directory format.
	FlagFormatDirectory = "dir"
	// Backup and restore uses Postgres SQL script format.
	FlagFormatScript = "sql"
)

func main() {
	var describe, exe string
	var err error
	if exe, err = os.Executable(); err != nil {
		fmt.Println(err)
		os.Exit(255)
	}
	//
	home := filepath.Dir(exe)
	backups := filepath.Join(home, "backups")
	app := &App{
		Args: &Flags{},
		Conf: Conf{
			Format: psql.Directory,
		},
		Files: Files{
			Binary: filepath.Base(exe),
		},
		Paths: Paths{
			Home:    home,
			Backups: backups,
		},
		Logger: &logger.STDOut{},
	}
	flag.BoolVar(&app.Args.Backup, "backup", false, "Backup all databases or specified databases.")
	flag.BoolVar(&app.Args.Clear, "clear", false, "Clear all backups from disk.")
	describe = `
Specify backup or restore format.
    dir     Backups are created as directories; restores occur from existing directories.
    sql     Backups are created as SQL script files; restores occur from existing files.
`
	flag.StringVar(&app.Args.Format, "format", FlagFormatDirectory, strings.TrimSpace(describe))
	flag.BoolVar(&app.Args.Help, "h", false, "")
	flag.BoolVar(&app.Args.Help, "help", false, "Print help and exit.")
	describe = `
When enabled this flag tells -restore to use the split SQL scripts as data sources.
`
	flag.BoolVar(&app.Args.Join, "join", false, strings.TrimSpace(describe))
	flag.BoolVar(&app.Args.List, "list", false, "List all databases that will be backed up.")
	flag.StringVar(&app.Args.Regexp, "regexp", ".*", "Optional regexp used to match targets for backup or restore.")
	flag.BoolVar(&app.Args.Restore, "restore", false, "Restore all databases or specified databases.")
	describe = `
Splits SQL script files into numbered parts of -split size in bytes.
    Use KiB, MiB, and GiB for sizes in powers of 1024.
    Use KB, MB, and GB for sizes in powers of 10.
    Only valid when "-backup -format sql" are also set and ignored otherwise.
`
	flag.StringVar(&app.Args.Split, "split", "8MiB", strings.TrimSpace(describe))
	flag.BoolVar(&app.Args.Verbose, "verbose", false, "Print psql commands as they are executed.")
	flag.BoolVar(&app.Args.Version, "v", false, "Print version information and exit.")
	flag.BoolVar(&app.Args.Version, "version", false, "Print version information and exit.")
	flag.Parse()
	app.Args.Remaining = flag.Args()
	if flag.NFlag() == 0 {
		app.Args.Help = true
	}
	if app.Args.Help {
		flag.PrintDefaults()
		os.Exit(0)
	}
	flag.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "format":
			switch f.Value.String() {
			case FlagFormatDirectory:
				app.Conf.Format = psql.Directory
			case FlagFormatScript:
				app.Conf.Format = psql.Script
			default:
				app.Infof("-format expected to be one of: %v, %v", FlagFormatDirectory, FlagFormatScript)
				os.Exit(255)
			}

		case "regexp":
			exp := f.Value.String()
			if exp == ".*" {
				return
			}
			re, err := regexp.Compile(exp)
			if err != nil {
				app.Infof("-regexp %v is invalid: %v", exp, err)
				os.Exit(255)
			}
			app.Conf.Regexp = re

		case "split":
			split := f.Value.String()
			if split == "" {
				split = "8MiB"
			}
			parsed, err := humanize.ParseBytes(split)
			if err != nil {
				app.Infof("Unable to parse -split %v : %v", split, err)
				os.Exit(255)
			}
			app.Conf.SplitSize = int(parsed)
		}
	})
	app.Run()
}
