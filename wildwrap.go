package main

import (
	"github.com/zlm2012/wildwrap/ts"
	"log"
	"os"
)

func main() {
	file, err := os.Open(os.Args[1])
	if err != nil {
		log.Fatalln(err)
	}
	decoder := ts.NewDecoder(file)
	eitSucceededCount := 0
	for {
		log.Println("try get next eit")
		eitFrame, err := decoder.ReadNextCurrentStreamEITFrame()
		if err != nil {
			log.Fatalln(err)
		}
		if eitFrame != nil {
			log.Println(*eitFrame)
			eitSucceededCount++
			if eitSucceededCount > 1 {
				return
			}
		}
	}
}
