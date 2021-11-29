package pgbackup

import "runtime"

// CalcConcurrency calculates the number of CPUs and simultaneous backup or restore
// operations to run concurrently.  In the case of backup and restore operations that can
// run concurrently the third argument is for PSQL -j argument (i.e. number of jobs pg_dump or
// pg_restore can use.)
func CalcConcurrency() (CPUs int, Ops int, Jobs int) {
	CPUs = runtime.NumCPU()
	if CPUs <= 2 {
		Ops, Jobs = 1, 1
		return
	} else if CPUs <= 4 {
		Ops, Jobs = 1, 2
		return
	}
	//
	Jobs = (CPUs - 4) / 4
	if Jobs > 4 {
		Jobs = 4
	}
	//
	Ops = (CPUs - 4) / Jobs
	//
	if Jobs < 1 {
		Jobs = 1
	}
	if Ops < 1 {
		Ops = 1
	}
	//
	return
}
