package main

import (
	"fmt"
)

func main() {
	sbt, err := New()
	if err != nil {
		panic(fmt.Errorf("init tray error:\n\t%w", err))
	}
	if err := sbt.Run(); err != nil {
		sbt.Fatal(err.Error())
	}
}
