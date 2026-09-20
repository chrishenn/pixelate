boot-env:
    mise i
    go mod tidy

    sudo chmod -R ug+x .githooks
    git config core.hooksPath githooks

lint:
    git hook run pre-commit
    just --fmt --unstable
