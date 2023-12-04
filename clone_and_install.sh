cd /tmp
echo -e "\033[1;34mInstalling cwc..."
echo -e "\033[0;90m"
git clone https://github.com/BL19/commands-wiki-cli
cd commands-wiki-cli
go build -o build/cwc
echo -e "\033[1;33m"
sudo cp build/cwc /usr/local/bin/cwc
sudo mkdir -p /usr/local/share/man/man1/
sudo cp cwc.1 /usr/local/share/man/man1/
cd ..
sudo rm -rf commands-wiki-cli
echo -e "\033[1;32mInstallation successful!"
echo -e "\033[0m"
