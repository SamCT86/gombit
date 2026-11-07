package database

import "runtime"

// fractionalAllocsPerRun uses the same warm-up and single-P measurement as
// testing.AllocsPerRun, but divides as floating-point rather than truncating
// to an integer. A stray allocation in 50 writes must not read as a whole
// added allocation per write; a genuine extra heap object on every write
// still changes the average by one.
func fractionalAllocsPerRun(runs int, f func()) float64 {
	defer runtime.GOMAXPROCS(runtime.GOMAXPROCS(1))
	f()
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)
	before := mem.Mallocs
	for i := 0; i < runs; i++ {
		f()
	}
	runtime.ReadMemStats(&mem)
	return float64(mem.Mallocs-before) / float64(runs)
}
