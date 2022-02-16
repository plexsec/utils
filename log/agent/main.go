// @file main.go
// @brief
// @author zhongtenghui
// @email zhongtenghui@gf.com.cn
// @created 2017-06-27 11:02:47
package main

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	//"gf.com.cn/gflib/discovery"
	"github.com/Shopify/sarama"
	"github.com/plexsec/utils/log/rlog"
)

var count = int32(0)

func traceFunc(file string) {
	f, err := os.OpenFile(file, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Fprintf(f, "begin trace\n")

	last := int32(0)
	for {
		new := atomic.LoadInt32(&count)
		speed := new - last
		last = new
		fmt.Fprintf(f, "total: %d, speed: %d/s\n", new, speed)
		time.Sleep(1 * time.Second)
	}
}

var producer sarama.AsyncProducer
var mutex = &sync.Mutex{}
var connected = int32(0)

func handleError() {
	for err := range producer.Errors() {
		if err.Err == sarama.ErrOutOfBrokers ||
			err.Err == sarama.ErrClosedClient ||
			err.Err == sarama.ErrNotConnected ||
			err.Err == sarama.ErrShuttingDown {
			// 断线了？重连
			atomic.StoreInt32(&connected, 0)
		}
	}
}

func initProducer() {
	mutex.Lock()
	defer mutex.Unlock()

	conf := sarama.NewConfig()
	conf.Net.KeepAlive = 30 * time.Second
	conf.Producer.Return.Errors = true
	conf.Producer.Return.Successes = false
	conf.ChannelBufferSize = 1024

	kafkaHosts := os.Getenv("LOG_AGENT_KAFKA")
	hosts := strings.Split(kafkaHosts, ",")
	// if len(hosts) == 0 {
	// 	// 不存在环境变量时，从consul 里去取
	// 	services := discovery.FindService("gf.com.cn.logcenter.kafka")
	// 	for _, s := range services {
	// 		hosts = append(hosts, net.JoinHostPort(s.Address, strconv.Itoa(s.Port)))
	// 	}
	// }

	if len(hosts) == 0 {
		fmt.Println("kafka hosts is empty")
		return
	}

	// 当kafka还没启动完时，这里直接就退出了，所以这里要进行重试
	for {
		var err error
		producer, err = sarama.NewAsyncProducer(hosts, conf)
		if err == sarama.ErrAlreadyConnected {
			return
		}

		if err != nil {
			fmt.Println(err)
			fmt.Printf("NewAsyncProducer error: %v, hosts: %s\n", err, hosts)
			time.Sleep(1 * time.Second)
			continue
		} else {
			go handleError()
			atomic.StoreInt32(&connected, 1)
			break
		}
	}
}

func main() {
	err := rlog.Init()
	if err != nil {
		fmt.Printf("init rlog error: %v\n", err)
		os.Exit(-1)
	}

	trace := os.Getenv("LOG_AGENT_TRACE")
	if trace != "" {
		go traceFunc(trace)
	}

	initProducer()

	const remoteTopic = "logstash"

	var pid string
	if cid := os.Getenv("HOSTNAME"); cid != "" {
		pid = fmt.Sprintf("%s|", cid)
	} else {
		pid = fmt.Sprintf("%d|", os.Getpid())
	}

	sleepIndex := 0
	for {
		connected := atomic.LoadInt32(&connected)
		if connected <= 0 {
			initProducer()
			continue
		}

		module, msg, err := rlog.Read()
		if err != nil {
			if err == rlog.ErrorQueryEmpty {
				// 队列空，等待一定时间后重试
				time.Sleep(100 * time.Millisecond)
				sleepIndex++
				continue
			} else if err == rlog.ErrorCasError {
				// 直接重试
				sleepIndex = 0
				continue
			}
		}
		sleepIndex = 0
		pm := &sarama.ProducerMessage{}
		pm.Topic = remoteTopic
		pm.Key = sarama.StringEncoder(module)
		pm.Partition = 1
		pm.Value = sarama.ByteEncoder(pid + msg)
		producer.Input() <- pm
		atomic.AddInt32(&count, 1)
	}
}
