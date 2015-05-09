package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	abat "github.com/iruimeng/abat/lib"
)

func attackCmd() command {
	fs := flag.NewFlagSet("abat attack", flag.ExitOnError)
	opts := &attackOpts{
		headers: headers{http.Header{}},
		laddr:   localAddr{&abat.DefaultLocalAddr},
	}

	fs.StringVar(&opts.targetsf, "targets", "stdin", "Targets file")
	fs.StringVar(&opts.targetsf, "t", "stdin", "Targets file")
	fs.StringVar(&opts.outputf, "output", "result.json", "Output file")
	fs.StringVar(&opts.outputf, "o", "result.json", "Output file")
	fs.StringVar(&opts.bodyf, "body", "", "Requests body file")
	fs.StringVar(&opts.ordering, "ordering", "random", "Attack ordering [sequential, random]")
	fs.DurationVar(&opts.duration, "duration", 10*time.Second, "Duration of the test")
	fs.DurationVar(&opts.timeout, "timeout", 0, "Requests timeout")
	fs.Uint64Var(&opts.rate, "rate", 0, "Requests per second")
	fs.Uint64Var(&opts.rate, "r", 0, "Requests per second")
	fs.Uint64Var(&opts.concurrency, "c", 0, "Concurrency level")
	fs.Uint64Var(&opts.number, "n", 1000, "Requests number")
	fs.IntVar(&opts.redirects, "redirects", 5, "Number of redirects to follow")
	fs.Var(&opts.headers, "header", "Request header")
	fs.Var(&opts.laddr, "laddr", "Local IP address")

	return command{fs, func(args []string) error {
		fs.Parse(args)
		return attack(opts)
	}}
}

// attackOpts压测函数参数配置结果体
type attackOpts struct {
	targetsf    string
	outputf     string
	bodyf       string
	ordering    string
	timeout     time.Duration
	rate        uint64
	duration    time.Duration
	concurrency uint64
	number      uint64
	redirects   int
	headers     headers
	laddr       localAddr
}

// attack参数验证和设置，压测入口并保存结果
func attack(opts *attackOpts) error {
	if opts.rate == 0 && opts.concurrency == 0 {
		return fmt.Errorf("Rate or Concurrency Level: can't be zero")
	} else if opts.rate != 0 && opts.concurrency != 0 {
		return fmt.Errorf("Rate is conflict with Concurrency Level:")
	}

	if opts.rate != 0 && opts.duration == 0 {
		return fmt.Errorf("duration can't be zero")
	}

	if opts.concurrency != 0 && opts.number == 0 {
		return fmt.Errorf("Request number can't be zero")
	}

	in, err := file(opts.targetsf, false)
	if err != nil {
		return fmt.Errorf("Target file (%s): %s", opts.targetsf, err)
	}
	defer in.Close()

	var body []byte
	if opts.bodyf != "" {
		bodyr, err := file(opts.bodyf, false)
		if err != nil {
			return fmt.Errorf("Body file (%s): %s", opts.bodyf, err)
		}
		defer bodyr.Close()

		if body, err = ioutil.ReadAll(bodyr); err != nil {
			return fmt.Errorf("Body file (%s): %s", opts.bodyf, err)
		}
	}

	targets, err := abat.NewTargetFrom(in, body, opts.headers.Header)
	//fmt.Println(targets)
	if err != nil || len(targets) == 0 {
		return fmt.Errorf("Target file (%s): %s", opts.targetsf, err)
	}

	switch opts.ordering {
	case "random":
		targets.Shuffle(time.Now().UnixNano())
	case "sequential":
		break
	default:
		return fmt.Errorf("Ordering `%s` is invalid", opts.ordering)
	}

	out, err := file(opts.outputf, true)
	if err != nil {
		return fmt.Errorf("Output file (%s): %s", opts.outputf, err)
	}
	defer out.Close()

	attacker := abat.NewAttacker(opts.redirects, opts.timeout, *opts.laddr.IPAddr)

	var results abat.Results
	if opts.rate != 0 {
		log.Printf(
			"abat is attacking %d targets in %s order and %d rate for %s...\n",
			len(targets),
			opts.ordering,
			opts.rate,
			opts.duration,
		)
		results = attacker.AttackRate(targets, opts.rate, opts.duration)
	} else if opts.concurrency != 0 {
		concurrency := opts.concurrency
		if opts.concurrency > opts.number {
			concurrency = opts.number
		}
		log.Printf(
			"abat is attacking %d targets in %s order and %d concurrency level for %d times...\n",
			len(targets),
			opts.ordering,
			concurrency,
			opts.number,
		)
		results = attacker.AttackConcy(targets, concurrency, opts.number)
	}

	log.Printf("Done! Writing results to '%s'...", opts.outputf)
	err = results.Encode(out)
	if err != nil {
		return err
	}

	data, err := abat.ReportText(results)
	if err != nil {
		return err
	}

	_, err = os.Stdout.Write(data)

	return err
}

// headers用作每次请求（http.Header）
type headers struct{ http.Header }

func (h headers) String() string {
	buf := &bytes.Buffer{}
	if err := h.Write(buf); err != nil {
		return ""
	}
	return buf.String()
}

func (h headers) Set(value string) error {
	parts := strings.SplitN(value, ":", 2)
	if len(parts) != 2 {
		return fmt.Errorf("Header '%s' has a wrong format", value)
	}
	key, val := strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
	if key == "" || val == "" {
		return fmt.Errorf("Header '%s' has a wrong format", value)
	}
	h.Add(key, val)
	return nil
}

// localAddr代表Flag接口以用来处理net.IPAddr
type localAddr struct{ *net.IPAddr }

func (ip *localAddr) Set(value string) (err error) {
	ip.IPAddr, err = net.ResolveIPAddr("ip", value)
	return
}
