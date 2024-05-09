grab the URL 
```sh
brew create https://github.com/ericwq/aprilsh/archive/refs/tags/0.6.40.tar.gz
```
update formula definition
```sh
cd ~/dev/aprilsh/homebrew
cp aprilsh.rb /usr/local/Homebrew/Library/Taps/homebrew/homebrew-core/Formula/a/aprilsh.rb
```
build and install formula
```sh
HOMEBREW_NO_INSTALL_FROM_API=1 brew install --build-from-source --verbose --debug aprilsh
```
uninstakl aprilsh
```sh
brew uninstall aprilsh
```
audit the formula
```sh
brew audit --strict --online aprilsh
brew audit --new --formula aprilsh
```
the second audit reports:
```
qiwang@Qi15Pro homebrew % brew audit --new --formula aprilsh
aprilsh
  * GitHub repository not notable enough (<30 forks, <30 watchers and <75 stars)
Error: 1 problem in 1 formula detected.
qiwang@Qi15Pro homebrew %
```
