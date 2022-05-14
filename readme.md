# Pixelate

A simple Go project to pixelate images. 

This is a very simple demo program to better understand concurrent producers/consumers using go-channels and go-funcs.  
Simply demonstrates the flexibility and (some) semantics of a go channel.


# Usage 

Only linux-amd64 is supported

Download the latest binary release from [releases](https://github.com/chrishenn/pixelate/releases/latest)

```bash
sudo chmod ug+x pixelate

./pixelate help

# Usage of ./pixelate:
#  -chunksize int
#        for NxN pixellated regions, chunkSize size 'N' in pixels (default 10)
#  -filter string
#        glob to match input image names that are in source dir (default "*")
#  -fio int
#        number of file read/write workers [a minimum 2 will be used] (default 48)
#  -i string
#        source dir of input images. Can be a relative path
#  -imgbuff int
#        max size of internal image-pointer queue (default 8000)
#  -iomode string
#        print to stdout with mode in {silent, basic, fancy}.
#        Note: there's a ~25% performance penalty for 'fancy' print on large numbers of input
#        images (default "fancy")
#  -o string
#        dir to write output images into (default: source from -i <src>)
#  -proc int
#        number of internal image processing workers [a minimum 2 will be used] (default 312)

# pixelate a folder full of images in `./testdata` and save the results to `./out`
./pixelate -i testdata -o out -filter '*.png'
```


# Features

- Read and pixelate most (input) image types
- Highly concurrent


# Not Supported

- Writing to image formats other than png
- OS's other than linux-amd64
- Files in the source folder are not properly filtered for valid images (panics on decode err)
  - Golang glob match filtering for input files is clunky
- Exposing internal go-func numbers and channel sizes as cli args seems hacky
- Cursory testing
- Benchmark dataset not provided
- Fancy print incurs a measurable performance penalty
- Project tools should be version-pinned


---


# Build

Set up a build env
- [install mise](https://mise.jdx.dev/getting-started.html)
- activate mise in the current shell
- `just boot-env`

Build and run
```bash
go get .
go build
go test
just bench
./pixelate help
```


# Project Tooling

I've used [mise](https://mise.jdx.dev) to expose these project tools to the active shell:
```bash
go@latest
golangci-lint@latest
just@latest
```

See justfile for runnable recipes
```bash
just --dump

# boot-env:
#    go install golang.org/x/tools/cmd/goimports@latest
#    go install golang.org/x/lint/golint@latest
#    go install github.com/segmentio/golines@latest
#    go mod download
#
#    sudo chmod -R ug+x .githooks
#    git config core.hooksPath .githooks
#
# lint:
#    git hook run pre-commit
#    just --fmt --unstable
#
# test:
#    go build
#    go test
#
# bench:
#    go build
#    go test -bench BenchmarkPixelate -run ^$ -count 1
```
Note that the "bench" recipe will look for a folder full of images at "./benchdata". See "Benchmarks" below.

The pre-commit hook, also included in `just lint`, runs various linting steps:
```bash
git hook run pre-commit
# go mod tidy
# golines --base-formatter goimports -w -m 120 .
# golint .
# go vet .
# golangci-lint run
```


# Benchmarks

Although the structure of this program is purposefully contrived, it is nonetheless surprising and delightful to see 
performance scaling with go-func numbers on various subtasks. Crucially, I've nearly 0 understanding of the 
characteristics of the go runtime, and a surface-level understanding of the language. Yet the simplicity of go 
semantics, coupled with its excellent tooling, make for near-trivial concurrency and "good enough" performance for 
"real" work. Delightful.   

The facenet image dataset (circa 2015) that I had handy includes 7864 jpg images of roughly 160x200 pixels. I would 
have included a script to download it, but I couldn't find it online. Any folder full of images will do.

There's a ~25% penalty for using "fancy" print (https://github.com/charmbracelet/bubbletea) while processing this 
dataset - it formats and prints each output image filename, and attempts (and mostly fails) to display a progress bar.

The "avtime" field holds the average number of milliseconds per pixelate call over "nloop" calls. Filesystem 
operations are included in the timing. Stddev for timings are not included.

All run with chunkSize=10, nloop=10.

```bash
goos: linux
goarch: amd64
pkg: github.com/chrishenn/pixelate
cpu: AMD Ryzen 9 9950X 16-Core Processor

varying processing, assembling numbers of gofuncs 
{read:24, write:24, chunk:1,   assemble:1,   imgbuff:8000, avtime:5782.2}
{read:24, write:24, chunk:4,   assemble:4,   imgbuff:8000, avtime:1796.7}
{read:24, write:24, chunk:32,  assemble:32,  imgbuff:8000, avtime:1067.5}
{read:24, write:24, chunk:156, assemble:156, imgbuff:8000, avtime:1078.8}

varying readers/writers numbers of gofuncs 
{read:1,  write:1,  chunk:156, assemble:156, imgbuff:8000, avtime:3969.1}
{read:4,  write:4,  chunk:156, assemble:156, imgbuff:8000, avtime:1497.7}
{read:8,  write:8,  chunk:156, assemble:156, imgbuff:8000, avtime:1230.4}
{read:24, write:24, chunk:156, assemble:156, imgbuff:8000, avtime:1078.8}

varying the size of channels queueing active image pointers 
{read:24, write:24, chunk:156, assemble:156, imgbuff:1,    avtime:1254.9}
{read:24, write:24, chunk:156, assemble:156, imgbuff:32,   avtime:1198.2}
{read:24, write:24, chunk:156, assemble:156, imgbuff:4096, avtime:1092.5}
{read:24, write:24, chunk:156, assemble:156, imgbuff:8000, avtime:1078.8}

varying the progress print mode 
{read:24, write:24, chunk:156, assemble:156, imgbuff:8000, iomode:"silent", avtime:1070.2}
{read:24, write:24, chunk:156, assemble:156, imgbuff:8000, iomode:"basic",  avtime:1081.6}
{read:24, write:24, chunk:156, assemble:156, imgbuff:8000, iomode:"fancy",  avtime:1345.6}
```

