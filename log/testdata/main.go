// @file main.go
// @brief
// @author zhongtenghui
// @email zhongtenghui@gf.com.cn
// @created 2017-06-23 17:44:46
package main

import (
	"flag"
	"fmt"
	"math/rand"
	"os"
	"sync/atomic"
	"time"

	"gf.com.cn/gflib/log"
)

var sleep = flag.Duration("sleep", 1*time.Second, "sleep time")

var count = int32(0)

func trace() {
	last := int32(0)
	for {
		new := atomic.LoadInt32(&count)
		fmt.Printf("total: %d, speed: %d/s\n", new, new-last)
		last = new
		time.Sleep(1 * time.Second)
	}
}

var r *rand.Rand // Rand for this package.

func init() {
	r = rand.New(rand.NewSource(time.Now().UnixNano()))
}

const chars = "abcdefghijklmnopqrstuvwxyz0123456789"

func randomString(strlen int) string {
	result := make([]byte, strlen)
	for i := range result {
		result[i] = chars[r.Intn(len(chars))]
	}
	return string(result)
}

func main() {
	flag.Parse()
	log.SetRemoteLogLevel(log.LOGLEVEL_VERBOSE)
	log.SetLogModuleName("testdata")
	err := log.InitRemoteLog()
	if err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}

	if *sleep == 0 {
		*sleep = 1 * time.Second
	}

	// 初始化一个随机字符串列表
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	var table [1000]string
	i := 0
	for i < len(table) {
		table[i] = randomString(r.Intn(1000))
		i++
	}

	// go trace()

	for {
		log.Verbose("%s", table[r.Intn(len(table))])
		atomic.AddInt32(&count, 1)
		time.Sleep(*sleep)
	}
}
