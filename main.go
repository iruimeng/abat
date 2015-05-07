// Copyright 2015 abat authors
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

// Abat is a Go implemented CLI cURL-like tool for humans
// abat [flags] [METHOD] URL [ITEM [ITEM]]
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"
)

func main() {
	commands := map[string]command{"bat": batCmd(), "attack": attackCmd(), "report": reportCmd()}

	flag.Usage = func() {
		fmt.Println("Usage: abat [METHOD] [flags] URL [ITEM [ITEM]]")
		for name, cmd := range commands {
			if name == "bat" {
				fmt.Printf("\n[%s] command:\n", name)
			} else {

				fmt.Printf("\n%s command:\n", name)
			}
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
		if err := batCmd().fn(args); err != nil {
			log.Fatal(err)
		}
	} else if err := cmd.fn(args[1:]); err != nil {
		log.Fatal(err)
	}
}

var examples string = `
examples:
	abat t.tt
	abat attack -h
	echo "POST http://127.0.0.1:8081/ form:filename:1.jpeg" | abat attack -duration=5s -rate=1 | tee results.bin | abat report
    abat attack -targets=targets.txt > results.bin
    abat report -input=results.bin -reporter=json > metrics.json
    cat results.bin | stress report -reporter=plot > plot.html
`

type command struct {
	fs *flag.FlagSet
	fn func(args []string) error
}
