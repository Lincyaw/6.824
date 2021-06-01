package mr

import (
	"io/ioutil"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"sync"
	"sync/atomic"
	"time"

	log "github.com/sirupsen/logrus"
)

type Master struct {
	// 锁
	mu sync.Mutex
	// 未做的任务列表
	Works chan Work
	// 已经发送的任务列表，不确定是否能够做完
	UnDoneWorks chan Work
	// the number of reduce tasks to use.
	NReduce int
	// 所有工作都完成了吗
	done bool
}

type Work struct {
	WorkId   int
	WorkType string
	FileName string
}

const (
	NotStarted = 0
	InTheMid   = 1
	Finish     = 2
)

func (m *Master) ReceiveStatus(args *WorkStatus, reply *ReplyWorker) error {
	if args.WorkType == "map" {
		if args.Done {
			atomic.StoreInt32(&m.nWorker[m.taskMap[args.WorkerId]], Finish)
			log.Info("Map", m.taskMap[args.WorkerId], " finished")
		} else {
			// 归零
			atomic.StoreInt32(&m.nWorker[m.taskMap[args.WorkerId]], NotStarted)
			log.Fatal("Map", m.taskMap[args.WorkerId], " failed")
		}
	} else {
		if args.Done {
			atomic.StoreInt32(&m.reduceWork[m.taskMap[args.WorkerId]], Finish)
			log.Info("Reduce", m.taskMap[args.WorkerId], " finished")
		} else {
			// 归零
			atomic.StoreInt32(&m.reduceWork[m.taskMap[args.WorkerId]], NotStarted)
			log.Fatal("Reduce", m.taskMap[args.WorkerId], " failed")
		}
	}
	m.mu.Lock()
	// Send map work first
	for i := range m.nWorker {
		if m.nWorker[i] != Finish {
			m.mu.Unlock()
			return nil
		}
	}
	for i := range m.reduceWork {
		if m.reduceWork[i] != Finish {
			m.mu.Unlock()
			return nil
		}

	}
	m.done = true
	m.mu.Unlock()
	return nil
}

func waitJob(m *Master, workType, workerId string) {
	// 睡 10 s
	mu := sync.Mutex{}
	mu.Lock()
	defer mu.Unlock()
	time.Sleep(time.Duration(10) * time.Second)
	switch workType {
	case "map":
		m.mu.Lock()
		if m.nWorker[m.taskMap[workerId]] == InTheMid {
			m.nWorker[m.taskMap[workerId]] = NotStarted
		}
		m.mu.Unlock()
	case "reduce":
		m.mu.Lock()
		if m.reduceWork[m.taskMap[workerId]] == InTheMid {
			m.reduceWork[m.taskMap[workerId]] = NotStarted
		}
		m.mu.Unlock()
	}

}

// Your code here -- RPC handlers for the worker to call.
//
// an example RPC handler.
//
// the RPC argument and reply types are defined in rpc.go.
//
// Worker request a task to master
func (m *Master) SendTask(args *Args, reply *ReplyWorker) error {
	// 每次调用这个函数，只分配一个任务
	// work := <-m.Works

	for i := range m.nWorker {
		if m.nWorker[i] == NotStarted {
			// 发送任务时需要考虑并发，保证每个任务在同一时刻只能有一个 worker 在访问
			m.mu.Lock()
			m.nWorker[i] = InTheMid
			m.mu.Unlock()
			reply.Filename = m.files[i]
			file, err := os.Open(reply.Filename)
			if err != nil {
				log.Fatalf("cannot open %v", reply.Filename)
			}
			defer file.Close()
			content, err := ioutil.ReadAll(file)
			if err != nil {
				log.Fatalf("cannot read %v", reply.Filename)
			}
			// 将id和对应的任务映射起来
			m.taskMap[args.WorkerId] = i

			reply.Content = string(content) // send all the doc.
			reply.WorkType = "map"
			reply.Id = i
			reply.NReduce = m.nReduce
			reply.Valid = true

			// 发射一个定时器，来检查 10 秒后任务是否完成，否则就任务这个 worker 挂掉了
			go waitJob(m, "map", args.WorkerId)
			return nil
		}
	}

	// 如果到这一步，说明map任务都发送出去了
	for k := 0; k < m.nReduce; k++ {
		if m.reduceWork[k] == NotStarted {
			m.taskMap[args.WorkerId] = k
			reply.Valid = true
			reply.WorkType = "reduce"
			reply.Id = k
			reply.NMap = len(m.files)
			reply.NReduce = m.nReduce
			m.reduceWork[k] = InTheMid
			go waitJob(m, "reduce", args.WorkerId)
			return nil
		}
	}
	reply.Valid = false
	return nil
}

//
// start a thread that listens for RPCs from worker.go
//
func (m *Master) server() {
	rpc.Register(m)
	rpc.HandleHTTP()
	//l, e := net.Listen("tcp", ":1234")
	sockname := masterSock()
	os.Remove(sockname)
	l, e := net.Listen("unix", sockname)
	if e != nil {
		log.Fatal("listen error:", e)
	}
	go http.Serve(l, nil)
}

//
// main/mrmaster.go calls Done() periodically to find out
// if the entire job has finished.
//
func (m *Master) Done() bool {
	//ret := false

	// Your code here.

	return m.done
}

// MakeMaster
// create a Master.
// main/mrmaster.go calls this function.
// nReduce is the number of reduce tasks to use.
//
func MakeMaster(files []string, nReduce int) *Master {
	m := Master{}
	for idx, file := range files {
		work := Work{}
		work.FileName = file
		work.WorkId = idx
		work.WorkType = "map"
		m.Works <- work
	}
	m.NReduce = nReduce
	m.server()
	return &m
}
