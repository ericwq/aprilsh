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
