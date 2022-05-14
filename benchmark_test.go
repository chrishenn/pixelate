package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"testing"
	"time"
)

type benchopt struct {
	nimgs     int
	chunkSize int
	read      int
	write     int
	chunk     int
	assemble  int
	imgbuff   int
	nloop     int
	avtime    float64
	iomode    ioMode
}

func BenchmarkPixelate(t *testing.B) {
	srcDir := "./benchdata"
	dstDir := "./out"
	filter := "*"

	matches, err := filepath.Glob(srcDir + "/*")
	if err != nil {
		log.Fatalf("BenchmarkPixelate: expected images in test folder at %s; %s", srcDir, err)
	}

	opt := &benchopt{
		nimgs:     len(matches),
		chunkSize: 10,
		read:      24,
		write:     24,
		chunk:     156,
		assemble:  156,
		imgbuff:   8000,
		nloop:     10,
		avtime:    0,
		iomode:    ioSilent,
	}

	var tsum int64 = 0
	for i := 0; i < opt.nloop; i++ {
		start := time.Now()
		pixelate(&pixelateOpt{
			srcDir:    &srcDir,
			dstDir:    &dstDir,
			filter:    &filter,
			chunkSize: &opt.chunkSize,
			nRead:     &opt.read,
			nWrite:    &opt.write,
			nChunk:    &opt.chunk,
			nAssemble: &opt.assemble,
			imgBuff:   &opt.imgbuff,
			iomode:    &opt.iomode,
		})
		tsum += time.Since(start).Milliseconds()

		if err := os.RemoveAll(dstDir); err != nil {
			log.Fatalf("failed to remove tmp test img output dir: %s\n", err)
		}
	}
	opt.avtime = float64(tsum) / float64(opt.nloop)
	fmt.Printf("%#v\n", opt)

}
