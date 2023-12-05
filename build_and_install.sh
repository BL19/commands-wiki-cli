GIT_SHA=$(git rev-parse HEAD)
GIT_BRANCH=$(git rev-parse --abbrev-ref HEAD)
go build -ldflags="-X main.sha=${GIT_SHA} -X main.branch=${GIT_BRANCH}" -o build/cwc
sudo cp build/cwc /usr/local/bin/cwc
sudo mkdir -p /usr/local/share/man/man1/
sudo cp cwc.1 /usr/local/share/man/man1/