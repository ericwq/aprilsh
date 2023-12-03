package main

import (
	"fmt"
	"os"
	"strings"
)

func main() {

	f, _ := os.Create("asb.txt")

	for i := 1; i < 200; i++ {
		row := strings.Repeat(fmt.Sprintf("%4d", i), 30)
		f.WriteString(fmt.Sprintf("%s\n", row))
	}

	f.Close()
}
