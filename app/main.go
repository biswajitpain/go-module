package main

import (
	"fmt"

	"github.com/biswajitpain/toolkit"
)

func main() {
	var tools toolkit.Tools
	for i := 0; i < 100; i++ {

		s := tools.RandomString(10)
		fmt.Println(i, s)

	}

}
