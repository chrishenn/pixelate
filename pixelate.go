package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	_ "image/gif"
	_ "image/jpeg"
	"image/png"
	"log"
	"math"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/tiff"
	_ "golang.org/x/image/webp"
)

type ioMode string

const (
	ioSilent ioMode = "silent"
	ioBasic  ioMode = "basic"
	ioFancy  ioMode = "fancy"
)

type pixelateOpt struct {
	srcDir    *string
	dstDir    *string
	filter    *string
	chunkSize *int
	nRead     *int
	nWrite    *int
	nChunk    *int
	nAssemble *int
	imgBuff   *int
	iomode    *ioMode
}

type chunkt struct {
	chunkColor  *color.RGBA
	chunkStartY int
	chunkStartX int
	chunkEndY   int
	chunkEndX   int
}

type imgt struct {
	fname     string
	imgsrc    *image.Image
	imgassm   *image.RGBA
	imgHt     int
	imgWt     int
	chunks    chan *chunkt
	chunkSize int
}

func getpaths(imgDir *string, pattern *string) chan string {

	imgPath, err := filepath.Abs(*imgDir)
	if err != nil {
		log.Fatalln(err)
	}

	ppath := filepath.Join(imgPath, *pattern)
	matches, err := filepath.Glob(ppath)
	if err != nil {
		log.Fatalln(err)
	}

	paths := make(chan string, len(matches))
	for _, mpath := range matches {
		abs, err := filepath.Abs(mpath)
		if err != nil {
			log.Fatalln()
		}
		paths <- abs
	}
	return paths
}

func read(paths chan string, decoded chan *imgt, wg *sync.WaitGroup) {

	for pth := range paths {

		file, err := os.Open(pth)
		if err != nil {
			log.Fatalln(err)
		}
		img, _, err := image.Decode(file)
		if err := file.Close(); err != nil {
			log.Fatalln(err)
		}
		if err != nil {
			log.Fatalln("decoding error:", err)
		}

		imgBound := img.Bounds()
		decoded <- &imgt{
			imgsrc: &img,
			fname:  pth,
			imgHt:  imgBound.Max.Y,
			imgWt:  imgBound.Max.X,
		}
	}
	wg.Done()
}

func chunk(decoded chan *imgt, chunked chan *imgt, chunkSize int, killSig chan int, wg *sync.WaitGroup) {

	for {
		select {
		case img := <-decoded:

			imgsrc := *img.imgsrc
			nBlocksX := (img.imgWt / chunkSize) + 1
			nChunks := ((img.imgHt / chunkSize) + 1) * nBlocksX
			chunks := make(chan *chunkt, nChunks)

			chunked <- &imgt{
				fname:     img.fname,
				imgHt:     img.imgHt,
				imgWt:     img.imgWt,
				chunks:    chunks,
				chunkSize: chunkSize,
			}

			for chunkI := 0; chunkI < nChunks; chunkI++ {

				startY := (chunkI / nBlocksX) * chunkSize
				startX := (chunkI % nBlocksX) * chunkSize
				endY := min(img.imgHt, startY+chunkSize)
				endX := min(img.imgWt, startX+chunkSize)

				var rSum, gSum, bSum, aSum float64 = 0, 0, 0, 0
				for y := startY; y < endY; y++ {
					for x := startX; x < endX; x++ {
						colr := color.RGBAModel.Convert(imgsrc.At(x, y)).(color.RGBA)
						rSum += float64(colr.R)
						gSum += float64(colr.G)
						bSum += float64(colr.B)
						aSum += float64(colr.A)
					}
				}
				npixl := float64((endY - startY) * (endX - startX))
				rAv := uint8(math.Round(rSum / npixl))
				gAv := uint8(math.Round(gSum / npixl))
				bAv := uint8(math.Round(bSum / npixl))
				aAv := uint8(math.Round(aSum / npixl))

				chunks <- &chunkt{
					chunkStartY: startY,
					chunkStartX: startX,
					chunkEndY:   endY,
					chunkEndX:   endX,
					chunkColor:  &color.RGBA{R: rAv, G: gAv, B: bAv, A: aAv},
				}
			}
			close(chunks)

		case <-killSig:
			wg.Done()
			return
		}
	}
}

func assemble(chunked chan *imgt, assembled chan *imgt, killSig chan int, wg *sync.WaitGroup) {

	for {
		select {
		case img := <-chunked:
			imgOut := image.NewRGBA(image.Rect(0, 0, img.imgWt, img.imgHt))

			for chunk := range img.chunks {
				for y := chunk.chunkStartY; y < chunk.chunkEndY; y++ {
					for x := chunk.chunkStartX; x < chunk.chunkEndX; x++ {
						imgOut.Set(x, y, chunk.chunkColor)
					}
				}
			}

			assembled <- &imgt{
				fname:     img.fname,
				chunkSize: img.chunkSize,
				imgassm:   imgOut,
			}

		case <-killSig:
			wg.Done()
			return
		}
	}
}

func write(assembled chan *imgt, dstDir string, done chan *string, killSig chan int, wg *sync.WaitGroup) {
	for {
		select {
		case img := <-assembled:

			dstAbs, err := filepath.Abs(dstDir)
			if err != nil {
				log.Fatalln(err)
			}

			dstf := path.Join(dstAbs, "pixelated_"+strconv.Itoa(img.chunkSize)+"_"+filepath.Base(img.fname))
			f, err := os.Create(dstf)
			if err != nil {
				log.Fatalln(err)
			}
			if err := png.Encode(f, img.imgassm); err != nil {
				log.Fatalln("encoding error:", err)
			}
			if err := f.Close(); err != nil {
				log.Fatalln(err)
			}
			done <- &dstf

		case <-killSig:
			wg.Done()
			return
		}
	}
}

func pixelate(opt *pixelateOpt) int {

	paths := getpaths(opt.srcDir, opt.filter)
	close(paths)
	nImgs := len(paths)
	if nImgs < 1 {
		log.Fatalf("No images in dir %s found matching pattern %s\n", *opt.srcDir, *opt.filter)
	}

	dstAbs, err := filepath.Abs(*opt.dstDir)
	if err != nil {
		log.Fatalln(err)
	}
	if err := os.MkdirAll(dstAbs, 0750); err != nil {
		log.Fatalln(err)
	}

	nWorker := *opt.nRead + *opt.nWrite + *opt.nChunk + *opt.nAssemble
	killSig := make(chan int, nWorker)
	wg := new(sync.WaitGroup)
	wg.Add(nWorker)

	decoded := make(chan *imgt, min(nImgs, *opt.imgBuff))
	chunked := make(chan *imgt, min(nImgs, *opt.imgBuff))
	assembled := make(chan *imgt, min(nImgs, *opt.imgBuff))
	done := make(chan *string, min(nImgs, *opt.imgBuff))

	for i := 0; i < *opt.nRead; i++ {
		go read(paths, decoded, wg)
	}
	for i := 0; i < *opt.nChunk; i++ {
		go chunk(decoded, chunked, *opt.chunkSize, killSig, wg)
	}
	for i := 0; i < *opt.nAssemble; i++ {
		go assemble(chunked, assembled, killSig, wg)
	}
	for i := 0; i < *opt.nWrite; i++ {
		go write(assembled, *opt.dstDir, done, killSig, wg)
	}

	switch *opt.iomode {
	case ioSilent:
		for i := 0; i < nImgs; i++ {
			<-done
		}
	case ioBasic:
		for i := 0; i < nImgs; i++ {
			log.Println(*<-done)
		}
	case ioFancy:
		if _, err := tea.NewProgram(newModel(done, nImgs)).Run(); err != nil {
			fmt.Println("Error running program:", err)
			os.Exit(1)
		}
	default:
		log.Fatalf("iomode must have value in {SILENT, BASIC, FANCY}; got: %s\n", *opt.iomode)
	}

	for i := 0; i < nWorker; i++ {
		killSig <- 1
	}
	wg.Wait()

	return nImgs
}

func nonemptyStrArgs(args ...*string) {
	for _, arg := range args {
		if *arg == "" {
			flag.Usage()
			os.Exit(1)
		}
	}
}

func main() {
	srcDir := flag.String("i", "", "source dir of input images. Can be a relative path")
	dstDir := flag.String("o", "", "dir to write output images into (default: source from -i <src>)")
	filter := flag.String("filter", "*", "glob to match input image names that are in source dir")
	chunkSize := flag.Int("chunksize", 10, "for NxN pixellated regions, chunkSize size 'N' in pixels")
	nfio := flag.Int("fio", 48, "number of file read/write workers [a minimum 2 will be used]")
	nproc := flag.Int("proc", 312, "number of internal image processing workers [a minimum 2 will be used]")
	imgBuff := flag.Int("imgbuff", 8000, "max size of internal image-pointer queue")
	iom := flag.String("iomode", "fancy", "print to stdout with mode in {silent, basic, fancy}.\n"+
		"Note: there's a ~25% performance penalty for 'fancy' print on large numbers of input images")
	flag.Parse()

	if *dstDir == "" {
		*dstDir = *srcDir
	}
	nonemptyStrArgs(srcDir, dstDir, filter, iom)

	var iomode ioMode
	switch ioMode(*iom) {
	case ioSilent, ioBasic, ioFancy:
		iomode = ioMode(*iom)
	default:
		log.Printf("iomode must be one of {silent, basic, fancy}; got: %s\n", *iom)
		flag.Usage()
		os.Exit(1)
	}

	nprocmin := max(2, *nproc)
	nChunk, nAssemble := nprocmin/2, nprocmin/2

	nfiomin := max(2, *nfio)
	nRead, nWrite := nfiomin/2, nfiomin/2

	opt := &pixelateOpt{
		srcDir:    srcDir,
		dstDir:    dstDir,
		filter:    filter,
		chunkSize: chunkSize,
		nRead:     &nRead,
		nWrite:    &nWrite,
		nChunk:    &nChunk,
		nAssemble: &nAssemble,
		imgBuff:   imgBuff,
		iomode:    &iomode,
	}
	tic := time.Now()
	nimgs := pixelate(opt)
	log.Printf("Processed %d images in %d ms", nimgs, time.Since(tic).Milliseconds())
}
