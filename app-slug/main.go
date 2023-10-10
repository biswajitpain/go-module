package main

import (
	"log"

	"github.com/biswajitpain/toolkit"
)

func main() {
	toSlug := "now is the time 123"
	var tools toolkit.Tools

	slugified, err := tools.Sligify(toSlug)
	if err != nil {
		log.Println(err)
	}
	log.Println(slugified)
}
