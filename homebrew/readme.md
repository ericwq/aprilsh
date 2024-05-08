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

