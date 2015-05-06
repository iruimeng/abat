// Copyright 2015 bat authors
//
// Licensed under the Apache License, Version 2.0 (the "License"): you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
// WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
// License for the specific language governing permissions and limitations
// under the License.

// Bat is a Go implemented CLI cURL-like tool for humans
// bat [flags] [METHOD] URL [ITEM [ITEM]]
package main

import (
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

func main() {
	commands := map[string]command{"accack": attackCmd(), "report": reportCmd(), "bat": batCmd()}

	flag.Usage = func() {
		fmt.Println("Usage: abat [flags] [METHOD] URL [ITEM [ITEM]]")
		for name, cmd := range commands {
			fmt.Printf("\n%s command:\n", name)
			cmd.fs.PrintDefaults()
		}
		fmt.Printf("\nglobal flags:\n  -cpus=%d Number of CPUs to use\n", runtime.NumCPU())
		fmt.Println(examples)
	}
	cpus := flag.Int("cpus", runtime.NumCPU(), "Number of CPUs to use")
	flag.Parse()

	runtime.GOMAXPROCS(*cpus)

	args := flag.Args()
	if len(args) == 0 {
		flag.Usage()
		os.Exit(1)
	}
	if cmd, ok := commands[args[0]]; !ok {
		//默认走bat命令
		if err := batCmd(args); err != nil {
			log.Fatal(err)
		}
	} else if err := cmd.fn(args[1:]); err != nil {
		log.Fatal(err)
	}
}

var examples string = `
examples:
	abat so.com
	abat accact help
	abat report help
`

//abat report -input=results.bin -reporter=json > metrics.json
//abat attack -target=targets.txt > log.bin

type command struct {
	fs *flag.FlagSet
	fn func(args []string) error
}
