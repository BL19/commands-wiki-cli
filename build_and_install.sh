go build -o build/cwc
sudo cp build/cwc /usr/local/bin/cwc
sudo mkdir -p /usr/local/share/man/man1/
sudo cp cwc.1 /usr/local/share/man/man1/