package abat

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"mime/multipart"
	"net/http"
	"os"
	"strings"
)

//单个http请求的结构
type Target struct {
	Method string
	Url    string
	Body   []byte
	File   string
	Header http.Header
}

//Targets is a slice of Target
type Targets []Target

//单次请求
func (t *Target) Request() (req *http.Request, err error) {
	method := strings.ToLower(t.Method)
	if method == "post" && t.File != "" {
		// 文件包含form子串
		if strings.Contains(t.File, "form") {
			buf := &bytes.Buffer{}
			wtr := multipart.NewWriter(buf)

			tmpAr := strings.Split(t.File, ":")
			var fileKey, fileName string

			if len(tmpAr) == 2 {
				fileKey = "ffile"
				fileName = tmpAr[1]
			} else if len(tmpAr) == 3 {
				fileKey = tmpAr[1]
				fileName = tmpAr[2]
			} else {
				err = fmt.Errorf("Form file: %s is illegal!", t.File)
				return
			}

			var (
				fw     io.Writer
				fd     *os.File
				e1, e2 error
			)
			if fw, e1 = wtr.CreateFormFile(fileKey, fileName); e1 != nil {
				err = e1
				return
			}

			if fd, e2 = os.Open(fileName); e2 != nil {
				err = e2
				return
			}
			defer fd.Close()
			_, err = io.Copy(fw, fd)
			if err != nil {
				return
			}
			wtr.Close()
			req, err = http.NewRequest(t.Method, t.Url, buf)
			req.Header.Set("Content-Type", wtr.FormDataContentType())

		} else {
			fbody, e := os.Open(t.File)
			defer fbody.Close()
			if e != nil {
				err = fmt.Errorf("Post file: (%s): %s", t.File, e)
				return
			}

			var bbody []byte

			if bbody, err = ioutil.ReadAll(fbody); err != nil {
				return
			}

			req, err = http.NewRequest(t.Method, t.Url, bytes.NewBuffer(bbody))
			req.Header.Set("Content-Length", fmt.Sprint(len(bbody)))

		}
	} else {
		req, err = http.NewRequest(t.Method, t.Url, bytes.NewBuffer(t.Body))
	}

	if err != nil {
		return
	}
	for key, val := range t.Header {
		req.Header[key] = make([]string, len(val))
		copy(req.Header[key], val)
	}
	req.Header.Set("User-Agent", "abat 0.1")

	fmt.Println(req.Header)
	return
}

// NewTargetForm 设置所有http.Header的头。
func NewTargetForm(source io.Reader, bbody []byte, header http.Header) (tgts Targets, err error) {
	scanner := bufio.NewScanner(source)
	var lines []string
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" && line[0:2] != "//" {
			lines = append(lines, line)
		}
	}
	if err = scanner.Err(); err != nil {
		return
	}
	return NewTargets(lines, bbody, header)
}

type headerMap map[string]string

// 初始化slice里面的target。并设置合法的body和http.Header信息
func NewTargets(lines []string, bbody []byte, header http.Header) (tgts Targets, err error) {
	for _, line := range lines {
		tmpAr := strings.Split(line, " ")
		argc := len(tmpAr)

		if argc < 2 {
			err = fmt.Errorf("Invalid request format: `%s`", line)
			return
		}

		newHeader := http.Header{}
		for k, v := range header {
			newHeader[k] = make([]string, len(v))
			copy(newHeader[k], v)
		}

		ii := 1

		if strings.Contains(tmpAr[ii], "http") == false {
			for ; strings.Contains(tmpAr[ii], "http") == false; ii++ {
				kv := strings.Split(tmpAr[ii], ":")
				if len(kv) == 2 {
					newHeader.Set(kv[0], kv[1])
				} else {
					continue
				}
			}
		}

		var url, file string

		if ii < argc {
			url = tmpAr[ii]
		} else {
			url = ""
		}
		ii++
		if ii < argc {
			file = tmpAr[ii]
		} else {
			file = ""
		}
		if url != "" {
			tgts = append(tgts, Target{Method: tmpAr[0], Url: url, File: file, Body: bbody, Header: newHeader})
		}
	}
	return
}

// Shuffle randomly alters the order of Targets with the provided seed
func (t Targets) Shuffle(seed int64) {
	rand.Seed(seed)
	for i, rnd := range rand.Perm(len(t)) {
		t[i], t[rnd] = t[rnd], t[i]
	}
}
