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

//isjson           = flag.Bool("json", true, "Send the data as a JSON object")
//method           = flag.String("method", "GET", "HTTP method")
//URL              = flag.String("url", "", "HTTP request URL")
//contentJsonRegex = `application/json`
type batOpts struct {
	ver              bool
	form             bool
	pretty           bool
	download         bool
	insecureSSL      bool
	auth             string
	proxy            string
	printV           string
	body             string
	help             bool
	URL              string
	isjson           bool
	method           string
	contentJsonRegex string
	jsonmap          map[string]interface{}
}

//var opts *batOpts = &batOpts{}

func batCmd() command {
	fs := flag.NewFlagSet("abat bat", flag.ExitOnError)
	opts := &batOpts{
		contentJsonRegex: `application/json`,
	}

	fs.BoolVar(&opts.isjson, "json", true, "Send the data as a JSON object")
	fs.BoolVar(&opts.ver, "v", false, "Print Version Number")
	fs.BoolVar(&opts.ver, "version", false, "Print Version Number")
	fs.BoolVar(&opts.pretty, "pretty", true, "Print Json Pretty Fomat")
	fs.BoolVar(&opts.pretty, "p", true, "Print Json Pretty Fomat")
	fs.StringVar(&opts.printV, "print", "A", "Print request and response")
	fs.BoolVar(&opts.form, "form", false, "Submitting as a form")
	fs.BoolVar(&opts.form, "f", false, "Submitting as a form")
	fs.BoolVar(&opts.download, "download", false, "Download the url content as file")
	fs.BoolVar(&opts.download, "d", false, "Download the url content as file")
	fs.BoolVar(&opts.insecureSSL, "insecure", false, "Allow connections to SSL sites without certs")
	fs.BoolVar(&opts.insecureSSL, "i", false, "Allow connections to SSL sites without certs")
	fs.StringVar(&opts.URL, "url", "", "HTTP request URL")
	fs.StringVar(&opts.method, "method", "GET", "HTTP method")
	fs.StringVar(&opts.auth, "auth", "", "HTTP authentication username:password, USER[:PASS]")
	fs.StringVar(&opts.auth, "a", "", "HTTP authentication username:password, USER[:PASS]")
	fs.StringVar(&opts.proxy, "proxy", "", "Proxy host and port, PROXY_URL")
	fs.StringVar(&opts.body, "body", "", "Raw data send as body")
	fs.BoolVar(&opts.help, "help", false, "Print Help Info")
	fs.BoolVar(&opts.help, "h", false, "Print Help Info")
	opts.jsonmap = make(map[string]interface{})

	return command{fs, func(args []string) error {
		fs.Parse(args)
		return bat(opts)
	}}
}

// abat拥有bat命令的全部
func bat(opts *batOpts) error {
	args := flag.Args()

	//fmt.Println("bat args\n", args)
	if len(args) > 0 {
		args = filter(args, opts)
	}
	if opts.ver {
		fmt.Println("Abat Version:", version)
		os.Exit(2)
	}
	if opts.printV != "A" && opts.printV != "B" {
		defaultSetting.DumpBody = false
	}
	var stdin []byte
	if runtime.GOOS != "windows" {
		fi, err := os.Stdin.Stat()
		if err != nil {
			panic(err)
		}
		if fi.Size() != 0 {
			stdin, err = ioutil.ReadAll(os.Stdin)
			if err != nil {
				log.Fatal("Read from Stdin", err)
				return err
			}
		}
	}

	if opts.URL == "" {
		batUsage()
	}
	if strings.HasPrefix(opts.URL, ":") {
		urlb := []byte(opts.URL)
		if opts.URL == ":" {
			opts.URL = "http://localhost/"
		} else if len(opts.URL) > 1 && urlb[1] != '/' {
			opts.URL = "http://localhost" + opts.URL
		} else {
			opts.URL = "http://localhost" + string(urlb[1:])
		}
	}
	if !strings.HasPrefix(opts.URL, "http://") && !strings.HasPrefix(opts.URL, "https://") {
		opts.URL = "http://" + opts.URL
	}
	u, err := url.Parse(opts.URL)
	if err != nil {
		log.Fatal(err)
		return err
	}
	if opts.auth != "" {
		userpass := strings.Split(opts.auth, ":")
		if len(userpass) == 2 {
			u.User = url.UserPassword(userpass[0], userpass[1])
		} else {
			u.User = url.User(opts.auth)
		}
	}
	opts.URL = u.String()
	httpreq := getHTTP(opts.method, opts.URL, args, opts)
	if u.User != nil {
		password, _ := u.User.Password()
		httpreq.GetRequest().SetBasicAuth(u.User.Username(), password)
	}
	// Insecure SSL Support
	if opts.insecureSSL {
		httpreq.SetTLSClientConfig(&tls.Config{InsecureSkipVerify: true})
	}
	// Proxy Support
	if opts.proxy != "" {
		purl, err := url.Parse(opts.proxy)
		if err != nil {
			log.Fatal("Proxy Url parse err", err)
			return err
		}
		httpreq.SetProxy(http.ProxyURL(purl))
	} else {
		eurl, err := http.ProxyFromEnvironment(httpreq.GetRequest())
		if err != nil {
			log.Fatal("Environment Proxy Url parse err", err)
			return err
		}
		httpreq.SetProxy(http.ProxyURL(eurl))
	}
	if opts.body != "" {
		httpreq.Body(opts.body)
	}
	if len(stdin) > 0 {
		var j interface{}
		err = json.Unmarshal(stdin, &j)
		if err != nil {
			httpreq.Body(stdin)
		} else {
			httpreq.JsonBody(j)
		}
	}

	res, err := httpreq.Response()
	if err != nil {
		log.Fatalln("can't get the url", err)
		return err
	}

	// download file
	if opts.download {
		var fl string
		if disposition := res.Header.Get("Content-Disposition"); disposition != "" {
			fls := strings.Split(disposition, ";")
			for _, f := range fls {
				f = strings.TrimSpace(f)
				if strings.HasPrefix(f, "filename=") {
					fl = strings.TrimLeft(f, "filename=")
				}
			}
		}
		if fl == "" {
			_, fl = filepath.Split(u.Path)
		}
		fd, err := os.OpenFile(fl, os.O_RDWR|os.O_CREATE, 0666)
		if err != nil {
			log.Fatal("can't create file", err)
			return err
		}
		if runtime.GOOS != "windowns" {
			fmt.Println(Color(res.Proto, Magenta), Color(res.Status, Green))
			for k, v := range res.Header {
				fmt.Println(Color(k, Gray), ":", Color(strings.Join(v, " "), Cyan))
			}
		} else {
			fmt.Println(res.Proto, res.Status)
			for k, v := range res.Header {
				fmt.Println(k, ":", strings.Join(v, " "))
			}
		}
		fmt.Println("")
		contentLength := res.Header.Get("Content-Length")
		var total int64
		if contentLength != "" {
			total, _ = strconv.ParseInt(contentLength, 10, 64)
		}
		fmt.Printf("Downloading to \"%s\"\n", fl)
		pb := NewProgressBar(total)
		pb.Start()
		multiWriter := io.MultiWriter(fd, pb)
		_, err = io.Copy(multiWriter, res.Body)
		if err != nil {
			log.Fatal("Can't Write the body into file", err)
			return err
		}
		pb.Finish()
		defer fd.Close()
		defer res.Body.Close()
		return err
	}

	if runtime.GOOS != "windows" {
		fi, err := os.Stdout.Stat()
		if err != nil {
			panic(err)
		}
		if fi.Mode()&os.ModeDevice == os.ModeDevice {
			if opts.printV == "A" || opts.printV == "H" || opts.printV == "B" {
				dump := httpreq.DumpRequest()
				if opts.printV == "B" {
					dps := strings.Split(string(dump), "\n")
					for i, line := range dps {
						if len(strings.Trim(line, "\r\n ")) == 0 {
							dump = []byte(strings.Join(dps[i:], "\n"))
							break
						}
					}
				}
				fmt.Println(ColorfulRequest(string(dump), opts))
				fmt.Println("")
			}
			if opts.printV == "A" || opts.printV == "h" {
				fmt.Println(Color(res.Proto, Magenta), Color(res.Status, Green))
				for k, v := range res.Header {
					fmt.Println(Color(k, Gray), ":", Color(strings.Join(v, " "), Cyan))
				}
				fmt.Println("")
			}
			if opts.printV == "A" || opts.printV == "b" {
				body := formatResponseBody(res, httpreq, opts.pretty, opts.contentJsonRegex)
				fmt.Println(ColorfulResponse(body, res.Header.Get("Content-Type"), opts))
			}
		} else {
			body := formatResponseBody(res, httpreq, opts.pretty, opts.contentJsonRegex)
			_, err = os.Stdout.WriteString(body)
			if err != nil {
				log.Fatal(err)
				return err
			}
		}
	} else {
		if opts.printV == "A" || opts.printV == "H" || opts.printV == "B" {
			dump := httpreq.DumpRequest()
			if opts.printV == "B" {
				dps := strings.Split(string(dump), "\n")
				for i, line := range dps {
					if len(strings.Trim(line, "\r\n ")) == 0 {
						dump = []byte(strings.Join(dps[i:], "\n"))
						break
					}
				}
			}
			fmt.Println(string(dump))
			fmt.Println("")
		}
		if opts.printV == "A" || opts.printV == "h" {
			fmt.Println(res.Proto, res.Status)
			for k, v := range res.Header {
				fmt.Println(k, ":", strings.Join(v, " "))
			}
			fmt.Println("")
		}
		if opts.printV == "A" || opts.printV == "b" {
			body := formatResponseBody(res, httpreq, opts.pretty, opts.contentJsonRegex)
			fmt.Println(body)
		}
	}
	return err
}

var usageinfo string = `abat is a Go implemented CLI cURL-like tool for humans.

Usage:

	abat [flags] [METHOD] URL [ITEM [ITEM]]
	
flags:
  -a, -auth=USER[:PASS]       Pass a username:password pair as the argument
  -body=""                    Send RAW data as body
  -f, -form=false             Submitting the data as a form
  -j, -json=true              Send the data in a JSON object
  -p, -pretty=true            Print Json Pretty Fomat
  -i, -insecure=false         Allow connections to SSL sites without certs
  -proxy=PROXY_URL            Proxy with host and port
  -print="A"                  String specifying what the output should contain, default will print all infomation
         "H" request headers
         "B" request body
         "h" response headers
         "b" response body
  -v, -verison=true           Show Version Number 

  attack help                 Usage of abat attack
  report help                 Usage of abat report

METHOD:
   abat defaults to either GET (if there is no request data) or POST (with request data).

URL:
  The only information needed to perform a request is a URL. The default scheme is http://,
  which can be omitted from the argument; example.org works just fine.

ITEM:
  Can be any of:
    Query string   key=value
    Header         key:value
    Post data      key=value
    File upload    key@/path/file

Example:
    
	abat t.tt
	
more help information please refer to https://github.com/iruimeng/abat	
`

func batUsage() {
	fmt.Println(usageinfo)
	os.Exit(2)
}
