package stat

import (
	"bytes"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/plexsec/utils/stat/sync"
)

type attrType int

const (
	ATTR_TYPE_ACC_PERIOD attrType = iota
	ATTR_TYPE_ACC_PERMANENT
	ATTR_TYPE_INSTANT
	ATTR_TYPE_MAX
)

type attrNode struct {
	Typ  attrType
	Attr string
	Cnt  int64
	Val  int64
	Max  int64
	Min  int64
	Mean int64
}

func (n *attrNode) clear() {
	if n.Typ == ATTR_TYPE_ACC_PERMANENT {
		return
	}
	aw.sl.Lock()
	delete(aw.nmap, n.Attr)
	aw.sl.Unlock()
}

type attrWorker struct {
	proc     string
	tags     []string
	hostname string

	cpuTag  string
	memTag  string
	heapTag string
	sl      sync.SpinLock
	sock    *net.UDPConn
	nmap    map[string]*attrNode
}

var aw *attrWorker

type Config struct {
	Addr string
	Tags []string
}

/*初始化属性上报
addr:属性收集svr地址,格式"ip:port"
tag:实例（进程，server）标记；如果为空，默认设置为程序名
prefix:表名前缀，不为空，则表名为 prefix-tablename
*/
func Init(cfg *Config) {
	if cfg == nil {
		fmt.Println("Stat config is empty, disable stat.")
		return
	}

	file, _ := exec.LookPath(os.Args[0])
	proc := filepath.Base(file)

	worker := &attrWorker{
		proc: proc,
		tags: cfg.Tags,
	}

	if hostname, err := os.Hostname(); err != nil {
		worker.hostname = "unknown"
	} else {
		worker.hostname = hostname
	}

	udpAddr, err := net.ResolveUDPAddr("udp", cfg.Addr)
	if err != nil {
		fmt.Printf("Stat address error: %v\n", err)
		return
	}
	sock, err := net.DialUDP("udp",
		nil,
		udpAddr,
	)
	if err != nil {
		fmt.Printf("Connect to stat address %v error: %v\n", udpAddr, err)
		return
	}
	worker.sock = sock
	worker.nmap = make(map[string]*attrNode)

	if aw == nil {
		aw = worker
		go loop()
	} else {
		aw = worker
	}

	return
}

// 设置属性累加值, 每周期清零
func Add(attr string, v int64) {
	add(attr, v, ATTR_TYPE_ACC_PERIOD)
}

// 设置属性累加值, 不清零
func AddPermanent(attr string, v int64) {
	add(attr, v, ATTR_TYPE_ACC_PERMANENT)
}

func add(attr string, v int64, t attrType) {
	if attr == "" || v == 0 || aw == nil {
		return
	}
	aw.sl.Lock()
	if node, ok := aw.nmap[attr]; ok {
		node.Val += v
		node.Cnt++
		if node.Max < v {
			node.Max = v
		}
		if node.Min > v {
			node.Min = v
		}
	} else {
		aw.nmap[attr] = &attrNode{
			Attr: attr,
			Typ:  t,
			Val:  v,
			Cnt:  1,
			Max:  v,
			Min:  v,
		}
	}
	aw.sl.Unlock()
}

//设置属性瞬时值
//attr:通过"."区分多个属性，格式为 tablename[.attr1]
//上报格式为 瞬时值,次数,最大值,最小值
func Instant(attr string, v int64) {
	if attr == "" || aw == nil {
		return
	}
	aw.sl.Lock()
	if node, ok := aw.nmap[attr]; ok {
		node.Val = v
		node.Cnt++
		if node.Max < v {
			node.Max = v
		}
		if node.Min > v {
			node.Min = v
		}
	} else {
		aw.nmap[attr] = &attrNode{
			Typ:  ATTR_TYPE_INSTANT,
			Attr: attr,
			Val:  v,
			Cnt:  1,
			Max:  v,
			Min:  v,
		}
	}
	aw.sl.Unlock()
}

func Duration(attr string, t time.Time) time.Duration {
	dura := time.Now().Sub(t)
	Add(attr, int64(dura))
	return dura
}

func loop() {
	var sendBuf bytes.Buffer
	tick := time.Tick(1 * time.Second)

	for {
		select {
		case <-tick:
			if aw == nil {
				break
			}

			aw.sl.Lock()
			al := make([]*attrNode, len(aw.nmap))
			i := 0
			for _, node := range aw.nmap {
				al[i] = node
				i++
			}
			aw.sl.Unlock()

			for _, a := range al {
				attrs := strings.Split(a.Attr, ".")
				if len(attrs) == 0 {
					continue
				}

				sendBuf.Reset()

				for i := 0; i < len(aw.tags); i++ {
					sendBuf.WriteString(fmt.Sprintf(",t%d=%s", i+1, aw.tags[i]))
				}

				for i := 0; i < len(attrs); i++ {
					sendBuf.WriteString(fmt.Sprintf(",l%d=%s", i+1, attrs[i]))
				}

				sendBuf.WriteString(fmt.Sprintf(" count=%d,value=%d,max=%d,min=%d\n",
					a.Cnt, a.Val, a.Max, a.Min))

				aw.sock.Write(sendBuf.Bytes())
				a.clear()
			}
		}
	}
}
