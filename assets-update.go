// +build ignore

package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
)

func main() {
	if err := wrapFile("data/providers.json", "assets-autogenerated.go"); err != nil {
		log.Fatal(err)
	}
}

func wrapFile(in, out string) error {
	input, err := os.Open(in)
	if err != nil {
		return err
	}
	defer input.Close()
	var m json.RawMessage
	if err := json.NewDecoder(input).Decode(&m); err != nil {
		return err
	}
	data, err := json.Marshal(m)
	if err != nil {
		return err
	}
	of, err := ioutil.TempFile("", "autogenerate-")
	if err != nil {
		return err
	}
	defer of.Close()
	defer os.Remove(of.Name())
	fmt.Fprintf(of, content, data)
	if err := of.Close(); err != nil {
		return err
	}
	return os.Rename(of.Name(), out)
}

const content = `// do not edit, autogenerated

package unfurlist

const providersData = %#q
`
