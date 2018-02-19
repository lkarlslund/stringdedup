package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"
	"unsafe"

	"github.com/cespare/xxhash"
	"github.com/lkarlslund/stringdedup"
)

type fileinfo struct {
	folder, basename, extension string
}

var files, files2 []fileinfo

func main() {
	var memstats runtime.MemStats

	runtime.ReadMemStats(&memstats)
	fmt.Printf("Initial usage: %v objects, consuming %v bytes\n", memstats.HeapObjects, memstats.HeapInuse)

	searchDir := "/usr"
	if runtime.GOOS == "windows" {
		searchDir = "c:/"
	}

	filepath.Walk(searchDir, func(path string, f os.FileInfo, err error) error {
		if !f.IsDir() {
			folder := filepath.Dir(path)
			extension := filepath.Ext(path)
			basename := filepath.Base(path)
			basename = basename[:len(basename)-len(extension)]
			files = append(files, fileinfo{
				folder:    folder,
				basename:  extension,
				extension: basename,
			})
		}
		return nil
	})

	runtime.GC()                       // Let garbage collector run, and see memory usage
	time.Sleep(time.Millisecond * 100) // Settle down
	runtime.ReadMemStats(&memstats)
	fmt.Printf("Memory usage for %v fileinfo: %v object, consuming %v bytes\n", len(files), memstats.HeapObjects, memstats.HeapInuse)

	fmt.Printf("Slice reference costs %v x %v bytes - a total of %v bytes\n", len(files), unsafe.Sizeof(fileinfo{}), len(files)*int(unsafe.Sizeof(fileinfo{})))

	checksum := xxhash.New()
	for _, fi := range files {
		checksum.Write([]byte(fi.folder + fi.basename + fi.extension))
	}
	fmt.Printf("Validation checksum on non deduped files is %x\n", checksum.Sum64())

	// NON DEDUPLICATED STATISTICS END

	// A new batch of fileinfo
	files2 = make([]fileinfo, len(files), cap(files))

	// Lets try that again with deduplication
	for i, fi := range files {
		files2[i] = fileinfo{
			folder:    stringdedup.S(fi.folder),
			basename:  stringdedup.S(fi.basename),
			extension: stringdedup.S(fi.extension),
		}
	}

	runtime.ReadMemStats(&memstats)
	fmt.Printf("Double allocated memory usage for %v fileinfo: %v objects, consuming %v bytes\n", len(files2), memstats.HeapObjects, memstats.HeapInuse)

	// Clear original fileinfo
	files = nil

	// Let garbage collector run, and see memory usage
	runtime.GC()
	time.Sleep(time.Millisecond * 100)
	runtime.GC()
	time.Sleep(time.Microsecond * 100)

	runtime.ReadMemStats(&memstats)
	fmt.Printf("Dedup memory usage for %v fileinfo: %v objects, consuming %v bytes\n", len(files2), memstats.HeapObjects, memstats.HeapInuse)

	checksum = xxhash.New()
	for _, fi := range files2 {
		checksum.Write([]byte(fi.folder + fi.basename + fi.extension))
	}
	fmt.Printf("Validation on dedup strings checksum is %x\n", checksum.Sum64())
	checksum = nil

	var bytes int
	for _, file := range files2 {
		bytes += len(file.basename) + len(file.extension) + len(file.folder)
	}
	fmt.Printf("Non-dedup string length count bytes is %v\n", bytes)
	fmt.Printf("Dedup string length count bytes is %v\n", stringdedup.ByteCount())

	// Drop index of deduplicated strings, so you can see how much memory that uses
	stringdedup.Flush()
	runtime.GC()
	time.Sleep(time.Millisecond * 100)

	runtime.ReadMemStats(&memstats)
	fmt.Printf("Flushed index memory usage: %v object, consuming %v bytes\n", memstats.HeapObjects, memstats.HeapInuse)

	// Clear deduped fileinfo
	files2 = nil
	_ = len(files2)

	// Let garbage collector run, and see memory usage
	// Clean up stuff left by finalizers
	runtime.GC()
	time.Sleep(time.Millisecond * 100)
	runtime.GC()

	runtime.ReadMemStats(&memstats)
	fmt.Printf("Cleared memory usage: %v object, consuming %v bytes\n", memstats.HeapObjects, memstats.HeapInuse)
}

// func printmemstats("")
