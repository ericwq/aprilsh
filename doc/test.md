# source
[Go: tests with HTML coverage report](https://kenanbek.medium.com/go-tests-with-html-coverage-report-f977da09552d)

## This command will run tests for the whole project.
```sh
go test -cover ./...
```

## In the first command, we use -coverprofile to save coverage results to the file. 
```sh
go test -coverprofile=coverage.out ./...
```

## we print detailed results by using Go’s cover tool.
```sh
go tool cover -func=coverage.out
```

## By using the same cover tool, we can also view coverage result as an HTML page
```sh
go tool cover -html=coverage.out
```

## You can select coverage mode by using -covermode option:

```sh
go test -covermode=count -coverprofile=coverage.out
```

- set: did each statement run?
- count: how many times did each statement run?
- atomic: like count, but counts precisely in parallel programs

## dependency lib

```sh
% apk add ncurses ncurses-terminfo ncurses-terminfo-base
% apk add build-base autoconf automake gzip libtool ncurses-dev openssl-dev perl-dev perl-io-tty protobuf-dev zlib-dev perl-doc
% apk add libxmu-dev mesa-dev freetype-dev
% apk add musl-locales-lang musl-locales utmps-dev
```
## Combined Unit and Integration Code Coverage

```sh
rm -rf coverage
mkdir -p coverage/unit -p coverage/int
```
### Run unit tests to collect coverage

```sh
go test -cover . -args -test.gocoverdir=./coverage/unit
```
### Retrieve total coverage

```sh
go tool covdata percent -i=./coverage/unit,./coverage/int
```

### Convert total coverage to cover profile

```sh
go tool covdata textfmt -i=./coverage/unit,./coverage/int -o coverage/profile
```

### View total coverage

```sh
go tool cover -func coverage/profile
go tool cover -html coverage/profile
```

### start server
```sh
go build -o ~/.local/bin/ashd .
~/.local/bin/apshd -verbose 1 2>> /tmp/apshd.log
GOCOVERDIR=./coverage/int ~/.local/bin/apshd -verbose 1 2>> /tmp/apshd.log
```
### start client
```sh
docker exec -u ide -it nvide ash
cd develop/aprilsh/frontend/client
go build -o apsh .
go build -race -o apsh .
./apsh -verbose 1  -pwd password ide@localhost 2>> /tmp/apsh.log
./apsh -verbose 1 -pwd password ide@172.17.0.3 2>> /tmp/apsh.log
```
### pprof
go tool pprof client cpu.profile
go test -bench=. -count 5 -run=^# -benchmem

## git branch

### create git branch

```sh
git branch new-feature                  ## create new branch
git checkout new-feature                ## switch to new branch
```

### commit to git branch
```sh
git add <changed-files>                 ## add all change to stage
git commit -m "add new feature"         ## commit change to local repo
git push origin new-feature             ## push change to remote repo
```

### merge to main branch
```sh
git checkout main                       ## switch to main branch
git merge new-feature                   ## merge new branch to main
git push origin main                    ## push main branch to remote repo
```

### solve conflict
```sh
git add <conflicted-file>               ## add conflict solved files to stage
git commit -m "resolve merge conflict"  ## commit change to local repo
git merge new-feature                   ## merge continue
```

### delete the branch
```sh
git branch -d new-feature               ## delete local branch
git push origin --delete new-feature    ## delete remote branch
```
