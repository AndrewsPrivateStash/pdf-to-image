/*
	take a pdf file and produce a directory of jpeg images
	$ pdfToImg -f myPDF -o myDir -s 0 -e 10 -a=true
*/

package main

import (
	"flag"
	"fmt"
	"image/jpeg"
	"log"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gen2brain/go-fitz"
)

func main() {
	inFileF := flag.String("f", "", "pdf input file path")
	outDirF := flag.String("o", "out", "name of output directory")
	startPgF := flag.Int("s", 0, "the starting page to convert")
	endPgF := flag.Int("e", -1, "the ending page to convert (-1 is all)")
	appendF := flag.Bool("a", false, "add files to directory without removing old ones")
	chunkSizeF := flag.Int("c", 100, "the chunksize to process before unloading the doc (avoids mem-leak)")

	flag.Parse()
	if flag.NFlag() < 1 {
		log.Fatal("for options: $ pdfToImg -h\nrequires at least a pdf file path.\n$ pdfToImg -f my_pdf.pdf")
	}

	// make output folder
	if err := os.MkdirAll(*outDirF, 0755); err != nil {
		log.Fatal(err)
	}

	// clean up output folder if already exists
	if !*appendF {
		log.Printf("removing files in %s\n", *outDirF)
		if _, err := os.Stat(*outDirF); !os.IsNotExist(err) {
			err = removeAllFiles(*outDirF)
			checkError(err)
		}
	}

	doc, err := fitz.New(*inFileF)
	checkError(err)
	totalPages := doc.NumPage()
	doc.Close()

	// Determine bounds
	startPage := 0
	if *startPgF > 0 && *startPgF <= *endPgF {
		startPage = *startPgF - 1
	}

	endPage := *endPgF
	if *endPgF < 0 || *endPgF > totalPages {
		endPage = totalPages
	}

	// process chunks
	startTime := time.Now()
	log.Printf("processing %d page(s), in chunks of: %d\n", endPage-startPage, *chunkSizeF)
	remPages := endPage - startPage
	curStart, curEnd := startPage, intMin(startPage+remPages, startPage+*chunkSizeF, totalPages)
	count := 0
	for remPages > 0 {
		count = processChunk(curStart, curEnd, *inFileF, *outDirF, count, endPage-startPage)
		remPages -= curEnd - curStart
		curStart, curEnd = curEnd, intMin(curEnd+*chunkSizeF, totalPages)
	}
	fmt.Printf("\nconversion took: %v\n", time.Since(startTime))
	fmt.Println("done! \xf0\x9f\x99\x8c")
}

func processChunk(start int, end int, f string, opath string, cnt int, tot int) int {
	doc, err := fitz.New(f)
	checkError(err)
	defer doc.Close()

	const pollFreq = 5

	// Extract pages as images
	count := cnt
	for n := start; n < end; n++ {

		img, err := doc.Image(n)
		checkError(err)

		f, err := os.Create(filepath.Join(opath, fmt.Sprintf("%03d.jpg", n+1)))
		checkError(err)

		err = jpeg.Encode(f, img, &jpeg.Options{Quality: 100})
		checkError(err)

		if count%pollFreq == 0 {
			logProgress(tot, count)
		}

		f.Close()
		count++
	}
	logProgress(tot, count)
	return count
}

func removeAllFiles(dir string) error {
	d, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer d.Close()
	names, err := d.Readdirnames(-1)
	if err != nil {
		return err
	}
	for _, name := range names {
		err = os.RemoveAll(filepath.Join(dir, name))
		if err != nil {
			return err
		}
	}
	return nil
}

func logProgress(tot int, cur int) {

	const SYMBOL_WIDTH = 20
	const SYMBOL = "#"

	progress := float64(cur) / float64(tot)
	outStr := fmt.Sprintf("\rthrough pg: %d\t[", cur)

	symCnt := int(math.Ceil(SYMBOL_WIDTH * progress))
	outStr += strings.Repeat(SYMBOL, symCnt)
	outStr += strings.Repeat(" ", SYMBOL_WIDTH-symCnt) + "]"
	outStr += string(" ") + fmt.Sprintf("%.1f%%", progress*100)
	fmt.Printf("%s", outStr)
}

func checkError(e error) {
	if e != nil {
		log.Fatal(e)
	}
}

func intMin(vals ...int) int {
	if len(vals) == 0 { //should not happen, break don't handle
		panic("no arguments passed to 'min'")
	}

	if len(vals) == 1 {
		return vals[0]
	}

	best := vals[0]
	for _, val := range vals[1:] {
		if val < best {
			best = val
		}
	}

	return best
}
