package psql

import (
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/nofeaturesonlybugs/fscopy"

	"pgbackup"
	"pgbackup/logger"
)

// DB links a dbname to a PSQL type.
type DB struct {
	DBName string
	PSQL
}

// Backup performs a backup of DB.
func (db DB) Backup(ctx context.Context, format Format) (string, error) {
	var cmd *exec.Cmd
	var dst string
	var out []byte
	var err error
	//
	dst, cmd = db.PSQL.Backup(ctx, db.DBName, format)
	//
	// Before running cmd we have to remove anything currently existing at dst.
	if _, err = os.Stat(dst); err == nil {
		if err = os.RemoveAll(dst); err != nil {
			return dst, err
		}
	}
	//
	if out, err = cmd.CombinedOutput(); err != nil {
		db.LogOutput(out)
		return dst, err
	}
	db.LogOutput(out)
	//
	// If format is Script then compute a hash as well.
	if format == Script {
		if err = pgbackup.File(dst).SHA512(); err != nil {
			db.Warningf("While hashing %v: %v", dst, err)
		}
	}
	//
	return dst, nil
}

// Chunk splits the given backup.sql file into chunks of the given size in bytes.
func (db DB) Chunk(src string, size int) error {
	if !strings.HasSuffix(src, ".sql") {
		return nil
	}
	var err error
	//
	basename := strings.TrimSuffix(src, ".sql")
	dir := basename + ".chunk"
	dst := filepath.Join(dir, filepath.Base(basename))
	//
	srcHash := basename + ".sha512"
	dstHash := filepath.Join(dir, "hash."+filepath.Base(srcHash))
	//
	if err = os.RemoveAll(dir); err != nil {
		return err
	} else if err = os.MkdirAll(dir, 0777); err != nil {
		return err
	} else if err = pgbackup.File(src).Split(dst, size, 9); err != nil {
		return err
	} else if err = os.Remove(src); err != nil {
		return err
	} else if err = fscopy.File(dstHash, srcHash); err != nil {
		return err
	} else if err = os.Remove(srcHash); err != nil {
		return err
	}
	return nil
}

// Join joins a backup.chunk directory back to a backup.sql file.  It is the converse of Chunk().
func (db DB) Join(src string) error {
	if !strings.HasSuffix(src, ".chunk") {
		return nil
	}
	//
	var globs []string
	var dfd *os.File
	var err error
	plain := strings.TrimSuffix(src, ".chunk")
	dst := plain + ".sql"
	//
	if globs, err = filepath.Glob(filepath.Join(src, filepath.Base(plain)+".*")); err != nil {
		return err
	}
	sort.Strings(globs)
	//
	if dfd, err = os.Create(dst); err != nil {
		return err
	}
	defer dfd.Close()
	//
	for _, part := range globs {
		err = func() error {
			var sfd *os.File
			var err error
			if sfd, err = os.Open(part); err != nil {
				return err
			}
			defer sfd.Close()
			//
			if _, err = io.Copy(dfd, sfd); err != nil {
				return err
			}
			//
			if err = sfd.Close(); err != nil {
				db.Warningf("Closing %v: %v", part, err)
			}
			//
			return nil
		}()
		if err != nil {
			return err
		}
	}
	//
	if err = dfd.Close(); err != nil {
		return err
	}
	//
	return nil
}

// Restore performs a restore of DB.
func (db DB) Restore(ctx context.Context, format Format) error {
	var cmd *exec.Cmd
	var out []byte
	var err error
	//
	cmd = db.Drop(ctx, db.DBName)
	if out, err = cmd.CombinedOutput(); err != nil {
		db.Warningf("While dropping %v; database may not exist.", db.DBName)
	}
	db.LogOutput(out)
	//
	cmd = db.Create(ctx, db.DBName)
	if out, err = cmd.CombinedOutput(); err != nil {
		db.LogOutput(out)
		return err
	}
	db.LogOutput(out)
	//
	cmd = db.PSQL.Restore(ctx, db.DBName, format)
	if format == Script {
		var stderr io.ReadCloser
		// Restore from script can become very verbose; limit logging to just stderr.
		if stderr, err = cmd.StderrPipe(); err != nil {
			return err
		}
		go func() {
			defer stderr.Close()
			log := logger.WarnWriter{Logger: db.PSQL.Logger}
			if _, err := io.Copy(log, stderr); err != nil {
				db.Warningf("Piping stderr %v", err)
			}
		}()
		if err = cmd.Run(); err != nil {
			db.LogOutput(out)
			return err
		}
	} else {
		if out, err = cmd.CombinedOutput(); err != nil {
			db.LogOutput(out)
			return err
		}
		db.LogOutput(out)
	}
	//
	return nil
}

// LogOutput logs the output from a command.
func (db DB) LogOutput(out []byte) {
	s := strings.TrimSpace(string(out))
	if s != "" {
		db.Infof(s)
	}
}
