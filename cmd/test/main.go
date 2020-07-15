package main

import (
	"log"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/demosdemon/shop/pkg/secrets"
)

const test = `
obj:
  username: !secret /path/to/username
  password: !secret /path/to/password
  regular: this is just a regular string
`

type Test struct {
	Obj struct {
		Username secrets.Secret
		Password secrets.Secret
		Regular  secrets.Secret
	} `yaml:"obj"`
}

func main() {
	r := strings.NewReader(test)
	dec := yaml.NewDecoder(r)
	var t Test
	err := dec.Decode(&t)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("%#v", t)
}
