package benchmark

import (
"fmt"
"runtime"
)

// MemoryBenchmark mide el uso de memoria
type MemoryBenchmark struct {
URL         string
Requests    int
Concurrency int
}

// NewMemoryBenchmark crea un nuevo benchmark de memoria
func NewMemoryBenchmark(url string, requests, concurrency int) *MemoryBenchmark {
return &MemoryBenchmark{
URL:         url,
Requests:    requests,
Concurrency: concurrency,
}
}

// Run ejecuta el benchmark de memoria
func (m *MemoryBenchmark) Run() MemoryStats {
// Medir memoria inicial
runtime.GC()
var initial runtime.MemStats
runtime.ReadMemStats(&initial)

// Ejecutar carga
overhead := NewOverheadBenchmark(m.URL, m.URL, m.Concurrency, m.Requests)
_, _, _ = overhead.Run()

// Medir memoria final
runtime.GC()
var final runtime.MemStats
runtime.ReadMemStats(&final)

return MemoryStats{
Alloc:      final.Alloc,
TotalAlloc: final.TotalAlloc,
Sys:        final.Sys,
NumGC:      final.NumGC - initial.NumGC,
}
}

// MemoryStats contiene estadísticas de memoria
type MemoryStats struct {
Alloc      uint64
TotalAlloc uint64
Sys        uint64
NumGC      uint32
}

func (s MemoryStats) String() string {
return fmt.Sprintf(
"Alloc: %.2f MB, TotalAlloc: %.2f MB, Sys: %.2f MB, GC: %d",
float64(s.Alloc)/(1024*1024),
float64(s.TotalAlloc)/(1024*1024),
float64(s.Sys)/(1024*1024),
s.NumGC,
)
}
