package utils

import (
	"log"
	"os"
)

func SaveSonDisk(fName string, data string) error {
	f, err := os.Create(fName)
	if err != nil {
		log.Fatal(err)
		return err
	}
	defer f.Close()
	_, err2 := f.WriteString(data)

	if err2 != nil {
		log.Fatal(err2)
		return err2
	}
	return nil
}
