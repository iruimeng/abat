package abat

import (
	"encoding/json"
	"io"
	"sort"
	"time"
)

// Result结构体定义了每个http.Client请求的指标
type Result struct {
	HttpCode  uint16
	Timestamp time.Time
	Latency   time.Duration
	BytesIn   uint64
	BytesOut  uint64
	Errors    string
}

// 结构体Result的slice集合，用作排序、编码等
type Results []Result

//Encode 用来编码Results并写到io.Writer out
func (r Results) Encode(out io.Writer) error {
	return json.NewEncoder(out).Encode(r)
}

func (r *Results) Decode(in io.Reader) error {
	return json.NewDecoder(in).Decode(r)
}

func (r *Results) Sort() Results {
	sort.Sort(r)
}

func (r *Results) Len() int           { return len(r) }
func (r *Results) Less(i, j int) bool { return r[i].Timestamp.Before(r[j].Timestamp) }
func (r *Results) Swap(i, j int)      { r[i], r[j] = r[j], r[i] }
