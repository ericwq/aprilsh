#
# https://medium.com/illumination/a-full-guide-on-coverage-in-golang-95164cdddcd9
#
go test -coverprofile=c.out .
go test -test.coverprofile c.out .
go tool cover -html=c.out
go tool cover -func=c.out
