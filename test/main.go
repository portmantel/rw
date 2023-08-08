package main

import (
	"fmt"
	"github.com/portmantel/rw"
)

func main() {
	for {
		fmt.Print("echo text check >> ")
		l := rw.ReadFromStdin()
		fmt.Println(l)
	}
}
