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
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

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
	log.Printf("processing %d pages, in chunks of: %d\n", totalPages, *chunkSizeF)
	remPages := endPage - startPage
	curStart, curEnd := startPage, intMin(startPage+*chunkSizeF, totalPages)
	count := 0
	for remPages > 0 {
		count = processChunk(curStart, curEnd, *inFileF, *outDirF, count)
		remPages -= curEnd - curStart
		curStart, curEnd = curEnd, intMin(curEnd+*chunkSizeF, totalPages)
	}

	fmt.Println("done!")
}

func processChunk(start int, end int, f string, opath string, cnt int) int {
	doc, err := fitz.New(f)
	checkError(err)
	defer doc.Close()

	// Extract pages as images
	count := cnt
	for n := start; n < end; n++ {
		logProgress(n, count)

		img, err := doc.Image(n)
		checkError(err)

		f, err := os.Create(filepath.Join(opath, fmt.Sprintf("%03d.jpg", n+1)))
		checkError(err)

		err = jpeg.Encode(f, img, &jpeg.Options{Quality: 100})
		checkError(err)

		f.Close()
		count++
	}

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

func clearTerminal() {
	switch o := runtime.GOOS; o {
	case "linux":
		cmd := exec.Command("clear")
		cmd.Stdout = os.Stdout
		if err := cmd.Run(); err != nil {
			log.Fatal(err)
		}
	case "darwin":
		cmd := exec.Command("clear")
		cmd.Stdout = os.Stdout
		if err := cmd.Run(); err != nil {
			return
		}
	case "windows":
		cmd := exec.Command("cmd", "/c", "cls")
		cmd.Stdout = os.Stdout
		if err := cmd.Run(); err != nil {
			return
		}
	default:
		return
	}
}

func logProgress(n int, c int) {
	if c > 5 {
		clearTerminal()
	}
	log.Printf("working on pg: %d\n", n+1)
}

func checkError(e error) {
	if e != nil {
		log.Fatal(e)
	}
}

func intMin(a, b int) int {
	if a < b {
		return a
	}
	return b
}
