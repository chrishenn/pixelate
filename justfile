boot-env:
    go install golang.org/x/tools/cmd/goimports@latest
    go install golang.org/x/lint/golint@latest
    go install github.com/segmentio/golines@latest
    go get .

    sudo chmod -R ug+x .githooks
    git config core.hooksPath .githooks

lint:
    git hook run pre-commit
    just --fmt --unstable

test:
    go build
    go test

bench:
    go build
    go test -bench BenchmarkPixelate -run ^$ -count 1
