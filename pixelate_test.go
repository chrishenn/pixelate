package main

import (
	"image"
	"log"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"sync"
	"testing"

	"github.com/sebdah/goldie/v2"
)

func TestGoldens(t *testing.T) {

	wd, err := os.Getwd()
	if err != nil {
		log.Fatalln(err)
	}
	testdata := path.Join(wd, "testdata")
	dstDir := "./out"

	for chunkSize := 10; chunkSize < 60; chunkSize += 10 {

		tests := map[string]struct {
			input  string
			golden string
		}{
			"Aaron Eckhart": {
				input:  "Aaron_Eckhart.png",
				golden: "Aaron_Eckhart_chunksize" + strconv.Itoa(chunkSize),
			},
			"Adrien Brody": {
				input:  "Adrien_Brody.jpg",
				golden: "Adrien_Brody_chunksize" + strconv.Itoa(chunkSize),
			},
		}

		for name, tc := range tests {
			t.Run(name, func(t *testing.T) {

				g := goldie.New(t)

				outn := "pixelated_" + strconv.Itoa(chunkSize) + "_" + filepath.Base(tc.input)
				outp := path.Join(dstDir, outn)

				nproc := 1
				mode := ioSilent
				pixelate(&pixelateOpt{
					srcDir:    &testdata,
					dstDir:    &dstDir,
					filter:    &tc.input,
					chunkSize: &chunkSize,
					nRead:     &nproc,
					nWrite:    &nproc,
					nChunk:    &nproc,
					nAssemble: &nproc,
					imgBuff:   &nproc,
					iomode:    &mode,
				})

				content, err := os.ReadFile(outp)
				if err != nil {
					t.Fail()
				}

				g.Assert(t, tc.golden, content)
			})
		}
	}

	if err := os.RemoveAll(dstDir); err != nil {
		log.Fatalf("failed to remove tmp test img output dir: %s\n", err)
	}

}

func verifyImgPaths(paths chan string, wg *sync.WaitGroup) {
	for pth := range paths {

		file, err := os.Open(pth)
		if err != nil {
			log.Fatalln("verifyImgPaths: file open error:", err)
		}
		_, _, err = image.Decode(file)
		if err := file.Close(); err != nil {
			log.Fatalln("verifyImgPaths: file close error:", err)
		}
		if err != nil {
			log.Fatalln("verifyImgPaths: decoding error:", err)
		}
	}
	wg.Done()
}

func TestEncodingErrors(t *testing.T) {
	// would benefit from a larger set of images to coax out concurrency bugs.
	srcDir := "./testdata"
	dstDir := "./out"
	filter := "*.png"
	chunksize := 10
	nread := 24
	nwrite := 24
	nproc := 156
	nassemble := 156
	imgbuff := 8000
	iomode := ioSilent

	pixelate(&pixelateOpt{
		srcDir:    &srcDir,
		dstDir:    &dstDir,
		filter:    &filter,
		chunkSize: &chunksize,
		nRead:     &nread,
		nWrite:    &nwrite,
		nChunk:    &nproc,
		nAssemble: &nassemble,
		imgBuff:   &imgbuff,
		iomode:    &iomode,
	})

	paths := getpaths(&dstDir, &filter)
	close(paths)
	ngo := max(2, len(paths)/1000)
	wg := new(sync.WaitGroup)
	wg.Add(ngo)
	for range ngo {
		go verifyImgPaths(paths, wg)
	}
	wg.Wait()

	if err := os.RemoveAll(dstDir); err != nil {
		log.Fatalf("failed to remove tmp test img output dir: %s\n", err)
	}

}
