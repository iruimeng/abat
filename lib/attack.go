// Copyright 2014 beego Author. All Rights Reserved.
// Copyright 2015 bat authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package abat

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"strings"
	"sync/atomic"
	"time"
)

var (
	// DefaultLocalAddr 本地IP地址
	DefaultLocalAddr = net.IPAddr{IP: net.IPv4zero}

	// DefaultTimeout DefaultAttacker的等待超时时长
	DefaultTimeout = 10 * time.Second
	// DefaultRedirect 代表DefaultAttacker的循环重定向次数.
	DefaultRedirect = 5
)

// Attacker为http.Client的接收者
type Attacker struct {
	client http.Client
}

var DefaultAttacker = NewAttacker(DefaultRedirect, DefaultTimeout, DefaultLocalAddr)

//atomic标记压测剩余次数
var remain int64

// NewAttacker返回一个新指针类型的Attacker
//
// redirects循环重定向次数
//
// timeout每个请求超时时间
//
// laddr本地ip用作每次请求
func NewAttacker(redirects int, timeout time.Duration, laddr net.IPAddr) Attacker {
	return &Attacker{http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			Dial: (&net.Dialer{
				Timeout:   timeout,
				KeepAlive: 30 * time.Second,
				LocalAddr: &net.TCPAddr{IP: laddr.IP, Zone: laddr.Zone},
			}).Dial,
			ResponseHeaderTimeout: timeout,
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
			TLSHandshakeTimeout: 10 * time.Second,
		},
		CheckRedirect: func(_ *http.Request, via []*http.Request) error {
			if len(via) > redirects {
				return fmt.Errorf("stopped after %d redirects", redirects)
			}
			return nil
		},
	}}
}

// AttackRate是规定时间(duration time)内按照(rate)频率来请求，并等待所以请求返回。
// 压测结果放在一个slice返回。
func (a *Attacker) AttackRate(tgts Targets, rate uint64, du time.Duration) Results {
	hits := int(rate * uint64(du.Seconds()))
	resc := make(chan Result)

	ticker := time.NewTicker(time.Duration(1e9 / rate))
	defer ticker.Stop()

	for i := 0; i < hits; i++ {
		<-ticker.C
		go func(tgt Target) {
			a.hit(tgt)
		}(tgts[i%len(tgts)])
	}
	rs := make(Results, 0, hits)
	for len(rs) < cap(rs) {
		rs = append(rs, <-resc)
	}
	return rs.Sort()
}

func (a *Attacker) hit(tgt Target) (rs Result) {
	req, err := tgt.Request()

	if err != nil {
		rs.Errors = err.Error()
		return
	}

	rs.Timestamp = time.Now()

	do, err := a.client.Do(req)
	if err != nil {
		rs.Errors = err.Error()
		return
	}

	rs.BytesOut = uint64(req.ContentLength)
	rs.HttpCode = uint16(do.StatusCode)

	bbody, err := ioutil.ReadAll(do.Body)

	if err != nil {
		if rs.HttpCode >= 300 || rs.HttpCode < 200 {
			rs.Errors = fmt.Sprintf("%s %s: %s", tgt.Method, tgt.Url, do.Status)
		}
		return
	}

	rs.Latency = time.Since(rs.Timestamp)
	rs.BytesIn = uint64(len(bbody))

	if rs.HttpCode >= 300 || rs.HttpCode < 200 {
		rs.Errors = fmt.Sprintf("%s %s: %s", tgt.Method, tgt.Url, do.Status)
	} else {
		//结果md5验证
		if strings.Contains(tgt.File, "md5") {
			kv = strings.Split(tgt.File, ":")
			if len(kv) == 2 && kv[1] != "" && len(kv[1]) == 32 {
				m := md5.New()
				m.Write(bbody)

				rsMd5 := hex.EncodeToString(m.Sum(nil))
				//Md5校验结果不一致，返回250 httpcode
				if rsMd5 != kv[1] {
					rs.HttpCode = 250
					rs.Errors = fmt.Sprintf("%s %s: MD5 not matced", tgt.Method, tgt.Url)
				}

			}
		}
	}
	if rs.HttpCode >= 250 || rs.HttpCode < 200 {
		log.Printf("%s\n", rs.Errors)
	}
	return

}

// AttackConcy在并发数为concurrency进行number次的请求，结果放进slice返回。和ab -n -c一样
func (a *Attacker) AttackConcy(tgts Targets, concurrency uint64, number uint64) Results {
	chanrs = make(chan Results)
	atomic.StoreInt64(&remain, int64(concurrency))

	//并发数不能大于请求数
	if concurrency > number {
		concurrency = number
	}

	var i uint64
	for i = 0; i < concurrency; i++ {
		go func(tgts Targets) { chanrs <- a.shoot(tgts) }(tgts)
	}

	rs := make(Results, 0, number)
	for i = 0; i < concurrency; i++ {
		rs = append(rs, <-chanrs)
	}
	return rs.Sort()
}

func (a *Attacker) shoot(tgts Targets) (rs Results) {
	rs = make(Results, 0, 1)
	//加载请求总次数
	localRemain := atomic.LoadInt64(&remain)

	for localRemain > 0 {
		atomic.AddInt64(&remain, -1)

		var r Result

		tgt := tgts[int(localRemain)%len(tgts)]

		q, err := tgt.Request()
		if err != nil {
			r.Errors = err.Error()
			rs = append(rs, r)
			localRemain == atomic.LoadInt64(&remain)
			continue
		}

		r.Timestamp = time.Now()
		do, err := a.client.Do(q)
		if err != nil {
			r.Errors = err.Error()
			rs = append(rs, r)
			localRemain == atomic.LoadInt64(&remain)
			continue
		}

		r.BytesOut = uint64(q.ContentLength)
		r.HttpCode = uint16(do.StatusCode)

		boby, err := ioutil.ReadAll(do.Body)
		if err != nil {
			if r.HttpCode >= 300 || r.HttpCode < 200 {
				r.Errors = fmt.Sprintf("%s %s: %s", tgt.Method, tgt.Url, do.Status)
			}
			rs = append(rs, r)
			localRemain == atomic.LoadInt64(&remain)
			continue
		}

		r.Latency = time.Since(r.Timestamp)
		r.BytesIn = uint64(len(boby))

		if r.HttpCode >= 300 || r.HttpCode < 200 {
			r.Errors = fmt.Sprintf("%s %s: %s", tgt.Method, tgt.Url, do.Status)
			log.Printf("%s\n", r.Error)
		} else {
			if strings.Contains(tgt.File, "md5") {

			}
			//@todo 对比MD5值
		}
		rs = append(rs, r)

		localRemain == atomic.LoadInt64(&remain)
	}
	return
}
