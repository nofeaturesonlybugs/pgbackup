package main

import (
	"bufio"
	"bytes"
	"context"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"pgbackup"
	"pgbackup/logger"
	"pgbackup/psql"
	"pgbackup/version"
)

// Files is the application files.
type Files struct {
	Binary string
}

// Paths is the application paths.
type Paths struct {
	Home    string
	Backups string
}

// App is the application.
type App struct {
	Args  *Flags
	Conf  Conf
	Files Files
	Paths Paths
	//
	PSQL psql.PSQL
	//
	// Number of CPUs reported by system.
	CPUs int
	// Number of backup or restore jobs that can occur concurrently.
	Ops int
	// Number of jobs (-j) that can be launched by pg_dump or pg_restore when
	// the backup format itself is a concurrent format.
	Jobs int
	//
	Ctx context.Context
	logger.Logger
}

// Run runs the application.
func (app *App) Run() {
	var cancelCtx func()
	var err error
	//
	app.Ctx, cancelCtx = context.WithCancel(context.Background())
	//
	app.CPUs, app.Ops, app.Jobs = pgbackup.CalcConcurrency()
	//
	err = os.MkdirAll(app.Paths.Backups, 0770)
	app.Error(err)
	//
	app.PSQL = psql.PSQL{
		DirBackups: app.Paths.Backups,
		Jobs:       app.Jobs,
		Logger:     logger.Nil,
	}
	if app.Args.Verbose {
		app.PSQL.Logger = app.Logger
	}
	//
	// SIGINT
	go func() {
		sigCh := make(chan os.Signal, 8)
		signal.Notify(sigCh, os.Interrupt)
		for {
			select {
			case <-sigCh:
				cancelCtx()
				return
			case <-app.Ctx.Done():
				return
			}
		}
	}()
	//
	switch true {
	case app.Args.Backup:
		app.ExecBackup()
	case app.Args.Restore:
		app.ExecRestore()
	case app.Args.Clear:
		app.ExecClear()
	case app.Args.List:
		app.ExecList()
	case app.Args.Version:
		app.ExecVersion()
	}
}

// Error exits the application if err is non-nil.
func (app *App) Error(err error) {
	if err != nil {
		app.Errorf("%v", err)
		os.Exit(255)
	}
}

// GetList returns a list of databases to backup.
func (app *App) GetList() []string {
	var rv []string
	//
	cmd := exec.Command("psql", "-l")
	if app.Args.Verbose {
		app.Infof(strings.Join(cmd.Args, " "))
	}
	//
	stdout, err := cmd.Output()
	scanner := bufio.NewScanner(bytes.NewBuffer(stdout))
	skip := true
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "------") {
			skip = false
			continue
		} else if skip {
			continue
		}
		//
		parts := strings.Split(line, "|")
		if len(parts) < 2 {
			continue
		}
		name := strings.TrimSpace(parts[0])
		switch name {
		case "":
		case "postgres":
		case "template0":
		case "template1":
		default:
			if app.Conf.Regexp == nil || app.Conf.Regexp.MatchString(name) {
				rv = append(rv, name)
			}
		}
	}
	app.Error(err)
	return rv
}

func (app *App) ExecBackup() {
	app.Infof("Start backups...")
	defer app.Infof("\tdone")
	//
	app.Summarize()
	//
	var dbs []string
	var dbsCh chan string
	if len(app.Args.Remaining) > 0 {
		// Explicitly named databases...
		dbs = append([]string(nil), app.Args.Remaining...)
		// Any databases matching the -regexp flag but only if it was set.
		if app.Conf.Regexp != nil {
			dbs = append(dbs, app.GetList()...)
		}
	} else {
		// All dbs
		dbs = app.GetList()
	}
	dbsCh = make(chan string, len(dbs))
	for _, db := range dbs {
		dbsCh <- db
	}
	close(dbsCh)
	//
	var wg sync.WaitGroup
	for k := 0; k < app.Ops; k++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var dst string
			var err error
			//
			for {
				select {
				case dbname, ok := <-dbsCh:
					if !ok {
						return
					}
					app.Infof("Starting %v...", dbname)
					db := psql.DB{
						DBName: dbname,
						PSQL:   app.PSQL,
					}
					if dst, err = db.Backup(app.Ctx, app.Conf.Format); err != nil {
						app.Warningf("Backing up %v failed: %v", dbname, err)
						continue
					}
					//
					if app.Conf.Format == psql.Script && app.Conf.SplitSize > 0 {
						if err = db.Chunk(dst, app.Conf.SplitSize); err != nil {
							app.Warningf("Split %v failed: %v", dbname, err)
							continue
						}
					}
					//
					app.Infof("Finished %v", dbname)

				case <-app.Ctx.Done():
					return
				}
			}
		}()
	}
	wg.Wait()
}

func (app *App) ExecRestore() {
	app.Infof("Start restore...")
	defer app.Infof("\tdone")
	//
	app.Summarize()
	//
	var paths []string
	var dbsCh chan string
	//
	// Source extensions depending on -format & -join flags.
	ext := ".backup"
	if app.Conf.Format == psql.Script {
		ext = ".sql"
		if app.Args.Join {
			ext = ".chunk"
		}
	}
	//
	// glob returns matches for the given extension in the backups directory.
	glob := func(extension string, re *regexp.Regexp) []string {
		var rv []string
		globs, err := filepath.Glob(filepath.Join(app.Paths.Backups, "*"+extension))
		app.Error(err)
		for _, glob := range globs {
			if re == nil || re.MatchString(filepath.Base(glob)) {
				rv = append(rv, glob)
			}
		}
		return rv
	}
	//
	if len(app.Args.Remaining) > 0 {
		// Explicitly listed databases...
		for _, path := range app.Args.Remaining {
			paths = append(paths, filepath.Join(app.Paths.Backups, path+ext))
		}
		// Plus those matching the -regexp flag but only if the regexp was specified.
		if app.Conf.Regexp != nil {
			paths = append(paths, glob(ext, app.Conf.Regexp)...)
		}
	} else {
		paths = append(paths, glob(ext, app.Conf.Regexp)...)
	}
	if len(paths) == 0 {
		return
	}
	dbsCh = make(chan string, len(paths))
	for _, path := range paths {
		dbsCh <- filepath.Base(strings.TrimSuffix(path, filepath.Ext(path)))
	}
	close(dbsCh)
	//
	//
	var wg sync.WaitGroup
	for k := 0; k < app.Ops; k++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var err error
			for {
				select {
				case dbname, ok := <-dbsCh:
					if !ok {
						return
					}
					path := filepath.Join(app.Paths.Backups, dbname+ext)
					app.Infof("Restoring %v from %v", dbname, path)
					//
					db := psql.DB{
						DBName: dbname,
						PSQL:   app.PSQL,
					}
					//
					if strings.HasSuffix(path, ".chunk") {
						db.Join(path)
					}
					//
					if err = db.Restore(app.Ctx, app.Conf.Format); err != nil {
						app.Warningf("Backing up %v failed: %v", dbname, err)
						continue
					}
					//
					app.Infof("Finished %v", dbname)

				case <-app.Ctx.Done():
					return
				}
			}
		}()
	}
	wg.Wait()
}

func (app *App) ExecClear() {
	backups, err := filepath.Glob(filepath.Join(app.Paths.Backups, "*.backup"))
	app.Error(err)
	chunks, err := filepath.Glob(filepath.Join(app.Paths.Backups, "*.chunk"))
	app.Error(err)
	scripts, err := filepath.Glob(filepath.Join(app.Paths.Backups, "*.sql"))
	app.Error(err)
	hashes, err := filepath.Glob(filepath.Join(app.Paths.Backups, "*.sha512"))
	app.Error(err)
	for _, path := range append(backups, append(chunks, append(scripts, hashes...)...)...) {
		app.Infof("Removing %v", filepath.Base(path))
		if err = os.RemoveAll(path); err != nil {
			app.Warningf("%v", err)
		}
	}
}

func (app *App) ExecList() {
	for _, name := range app.GetList() {
		app.Infof(name)
	}
}

func (app *App) ExecVersion() {
	if app.Args.Version {
		if strings.Contains(strings.Join(os.Args, " "), "-version") {
			app.Infof(version.GetVersionVerbose(app.Files.Binary))
			return
		}
		app.Infof(version.GetVersion(app.Files.Binary))
	}
}

// Summarize prints a log line describing what the application is doing with
// concurrency information.
func (app *App) Summarize() {
	if app.Args.Backup {
		app.Infof("Backing up with %v concurrent backups across %v CPUs", app.Ops, app.CPUs)
		if app.Conf.Format == psql.Directory {
			app.Infof("\tEach backup uses %v jobs.", app.Jobs)
		}
	} else if app.Args.Restore {
		app.Infof("Restoring with %v concurrent restores across %v CPUs", app.Ops, app.CPUs)
		if app.Conf.Format == psql.Directory {
			app.Infof("\tEach restore uses %v jobs.", app.Jobs)
		}
	}
}
