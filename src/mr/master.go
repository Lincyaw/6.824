package mr

import (
	"io/ioutil"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

func init() {
	log.SetLevel(log.ErrorLevel)
}

type WorkState int32
type Master struct {
	// 锁
	mu sync.Mutex

	mapWork     int
	reduceWork  int
	WorkState   []WorkState
	FinishedMap int

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

func (m *Master) emitReduce() {
	for i := 0; i < m.reduceWork; i++ {
		work := Work{}
		work.WorkId = m.mapWork + i
		work.WorkType = "reduce"
		m.ReduceWorks <- work
		m.WorkState = append(m.WorkState, NotStarted)
	}
}
func (m *Master) ReceiveStatus(args *WorkStatus, reply *ReplyWorker) error {
	if args.Done {
		m.mu.Lock()
		if m.WorkState[args.WorkId] == InTheMid {
			m.WorkState[args.WorkId] = Finish
			m.FinishedMap++
			if m.FinishedMap == m.mapWork {
				m.emitReduce()
			}
			if m.FinishedMap == m.mapWork+m.reduceWork {
				m.done = true
				log.Info("All Work finished.")
			}
		}
		m.mu.Unlock()
	} else {
		if args.WorkType == "map" {
			m.MapWorks <- args.Work
			m.mu.Lock()
			m.WorkState[args.WorkId] = NotStarted
			m.mu.Unlock()
		} else {
			m.ReduceWorks <- args.Work
			m.mu.Lock()
			m.WorkState[args.WorkId] = NotStarted
			m.mu.Unlock()
		}
	}
	return nil
}

func waitJob(m *Master, work Work) {
	// 睡 8 s
	time.Sleep(time.Duration(8) * time.Second)
	m.mu.Lock()
	log.Warn("length of workState: ", len(m.WorkState), " ,work id: ", work.WorkId)
	if m.WorkState[work.WorkId] == InTheMid {
		m.WorkState[work.WorkId] = NotStarted
		if work.WorkType == "map" {
			m.MapWorks <- work
		} else {
			m.ReduceWorks <- work
		}
	}
	m.mu.Unlock()
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
	reply.ReduceWork = m.reduceWork
	return
}
func (m *Master) SendTask(args *Args, reply *ReplyWorker) error {
	// 每次调用这个函数，只分配一个任务
	select {
	case work := <-m.MapWorks:
		log.Info("map, ", work)
		*reply = m.sendMap(work)
		m.mu.Lock()
		m.WorkState[work.WorkId] = InTheMid
		m.mu.Unlock()
		go waitJob(m, work)
	case work := <-m.ReduceWorks:
		log.Info("reduce, ", work)

		// deal with reduce
		reply.WorkType = work.WorkType
		reply.Id = work.WorkId
		reply.MapWork = m.mapWork
		reply.Valid = true
		m.mu.Lock()
		m.WorkState[work.WorkId] = InTheMid
		m.mu.Unlock()
		go waitJob(m, work)

	default:
		reply.Valid = false
	}
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
	log.Info("init")
	m.MapWorks = make(chan Work, len(files))
	m.ReduceWorks = make(chan Work, nReduce)
	m.mapWork = len(files)
	m.reduceWork = nReduce
	m.FinishedMap = 0
	for idx, file := range files {
		work := Work{}
		work.FileName = file
		work.WorkId = idx
		work.WorkType = "map"
		m.MapWorks <- work
		m.WorkState = append(m.WorkState, NotStarted)
	}
	m.server()
	return &m
}
