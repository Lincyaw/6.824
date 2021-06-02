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

	mapWork    int
	reduceWork int

	// 未做的 map 任务列表
	MapWorks chan Work
	// 未做的 reduce 任务列表
	ReduceWorks chan Work
	// 已经发送的任务列表，不确定是否能够做完
	UnDoneWorks chan Work
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
func (m *Master) sendMap(work Work) (reply ReplyWorker) {
	reply.Filename = work.FileName
	file, err := os.Open(reply.Filename)
	if err != nil {
		log.Fatalf("cannot open %v", reply.Filename)
	}
	defer file.Close()
	content, err := ioutil.ReadAll(file)
	if err != nil {
		log.Fatalf("cannot read %v", reply.Filename)
	}
	reply.Content = string(content)
	reply.WorkType = work.WorkType
	reply.Id = work.WorkId
	reply.Valid = true
	return
}
func (m *Master) SendTask(args *Args, reply *ReplyWorker) error {
	// 每次调用这个函数，只分配一个任务
	for {
		select {
		case work := <-m.MapWorks:
			*reply = m.sendMap(work)

		case work := <-m.ReduceWorks:
		priority:
			for {
				select {
				case work := <-m.MapWorks:
					*reply = m.sendMap(work)
				default:
					break priority
				}
			}
			// deal with reduce
			reply.WorkType = work.WorkType
			reply.Id = work.WorkId
			reply.MapWork = m.mapWork
			reply.Valid = true
		default:
			reply.Valid = false
		}
	}
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
	m.MapWorks = make(chan Work, len(files))
	m.ReduceWorks = make(chan Work, nReduce)
	m.mapWork = len(files)
	m.reduceWork = nReduce
	for idx, file := range files {
		work := Work{}
		work.FileName = file
		work.WorkId = idx
		work.WorkType = "map"
		m.MapWorks <- work
	}
	m.server()
	return &m
}
