package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"
	"unsafe"

	"github.com/OneOfOne/xxhash"
	"github.com/lkarlslund/stringdedup"
)

type fileinfo struct {
	folder, basename, extension string
}

var files, files2 []fileinfo

func main() {
	fmt.Println("String deduplication demonstration")
	fmt.Println("---")

	d := stringdedup.New(func(in []byte) uint32 {
		return xxhash.Checksum32(in)
	})

	var memstats runtime.MemStats

	runtime.ReadMemStats(&memstats)
	fmt.Printf("Initial memory usage at start of program: %v objects, consuming %v bytes\n", memstats.HeapObjects, memstats.HeapInuse)
	fmt.Println("---")

	searchDir := "/usr"
	if runtime.GOOS == "windows" {
		searchDir = "c:/windows"
	}

	fmt.Printf("Scanning and indexing files in %v - hang on ...\n", searchDir)

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

	fmt.Println("Scanning done!")
	fmt.Println("---")

	runtime.GC()                       // Let garbage collector run, and see memory usage
	time.Sleep(time.Millisecond * 100) // Settle down
	runtime.ReadMemStats(&memstats)
	fmt.Printf("Memory usage for %v fileinfo: %v object, consuming %v bytes\n", len(files), memstats.HeapObjects, memstats.HeapInuse)

	undedupbytes := memstats.HeapInuse

	fmt.Printf("Slice reference costs %v x %v bytes - a total of %v bytes\n", len(files), unsafe.Sizeof(fileinfo{}), len(files)*int(unsafe.Sizeof(fileinfo{})))

	checksum := xxhash.New64()
	for _, fi := range files {
		checksum.Write([]byte(fi.folder + fi.basename + fi.extension))
	}
	csum := checksum.Sum64()
	fmt.Printf("Validation checksum on non deduped files is %x\n", csum)
	fmt.Println("---")

	// NON DEDUPLICATED STATISTICS END

	// A new batch of fileinfo
	files2 = make([]fileinfo, len(files), cap(files))

	// Lets try that again with deduplication
	for i, fi := range files {
		files2[i] = fileinfo{
			folder:    d.S(fi.folder),
			basename:  d.S(fi.basename),
			extension: d.S(fi.extension),
		}
	}

	runtime.ReadMemStats(&memstats)
	fmt.Println("Both a duplicated and non-deduplicated slice is now in memory")
	fmt.Printf("Double allocated memory usage for %v fileinfo: %v objects, consuming %v bytes\n", len(files2), memstats.HeapObjects, memstats.HeapInuse)

	// Let garbage collector run, and see memory usage
	runtime.KeepAlive(files)
	files = nil
	runtime.GC()
	time.Sleep(time.Millisecond * 1000)

	runtime.ReadMemStats(&memstats)
	fmt.Println("---")
	fmt.Printf("Dedup memory usage for %v fileinfo: %v objects, consuming %v bytes\n", len(files2), memstats.HeapObjects, memstats.HeapInuse)

	dedupbytes := memstats.HeapInuse
	fmt.Printf("Reduction in memory usage: %.2f\n", float32(dedupbytes)/float32(undedupbytes))

	// Drop indexes and let's see
	d.Flush()
	runtime.GC()
	time.Sleep(time.Millisecond * 1000)

	runtime.ReadMemStats(&memstats)
	fmt.Println("---")
	fmt.Printf("Flushed index memory usage: %v object, consuming %v bytes\n", memstats.HeapObjects, memstats.HeapInuse)
	fmt.Printf("Reduction in memory usage (after dropping indexes): %.2f\n", float32(memstats.HeapInuse)/float32(undedupbytes))

	// Validate that deduped files are the same as non deduped files
	checksum = xxhash.New64()
	for _, fi := range files2 {
		checksum.Write([]byte(fi.folder + fi.basename + fi.extension))
	}
	fmt.Println("---")
	csum2 := checksum.Sum64()
	fmt.Printf("Validation on dedup strings checksum is %x\n", csum2)
	checksum = nil

	if csum != csum2 {
		fmt.Println("!!! VALIDATION FAILED. DEDUPED STRINGS ARE NOT THE SAME AS NON DEDUPED STRINGS !!!")
	}

	var bytes int
	for _, file := range files2 {
		bytes += len(file.basename) + len(file.extension) + len(file.folder)
	}

	// Let garbage collector run, and see memory usage
	// Clean up stuff left by finalizers
	files2 = nil
	runtime.GC()
	time.Sleep(time.Millisecond * 100)
	runtime.GC()

	runtime.ReadMemStats(&memstats)
	fmt.Println("---")
	fmt.Printf("Cleared memory usage: %v object, consuming %v bytes\n", memstats.HeapObjects, memstats.HeapInuse)
}

// func printmemstats("")
