package raft

//
// this is an outline of the API that raft must expose to
// the service (or tester). see comments below for
// each of these functions for more details.
//
// rf = Make(...)
//   create a new Raft server.
// rf.Start(command interface{}) (index, term, isleader)
//   start agreement on a new log entry
// rf.GetState() (term, isLeader)
//   ask a Raft for its current term, and whether it thinks it is leader
// ApplyMsg
//   each time a new entry is committed to the log, each Raft peer
//   should send an ApplyMsg to the service (or tester)
//   in the same server.
//

import (
	"context"
	"math/rand"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"../labrpc"
	"github.com/sirupsen/logrus"
)

var logger *logrus.Logger

func init() {
	logger = logrus.New()

	file, err := os.OpenFile("logs/"+time.Now().Format(time.UnixDate)+".log", os.O_CREATE|os.O_WRONLY, 0666)
	if err == nil {
		logger.Out = file
	} else {
		logger.Info("Failed to log to file, using default stderr")
	}
	logger.SetLevel(logrus.ErrorLevel)
}

// import "bytes"
// import "../labgob"

//
// as each Raft peer becomes aware that successive log entries are
// committed, the peer should send an ApplyMsg to the service (or
// tester) on the same server, via the applyCh passed to Make(). set
// CommandValid to true to indicate that the ApplyMsg contains a newly
// committed log entry.
//
// in Lab 3 you'll want to send other kinds of messages (e.g.,
// snapshots) on the applyCh; at that point you can add fields to
// ApplyMsg, but set CommandValid to false for these other uses.
//
type ApplyMsg struct {
	CommandValid bool
	Command      interface{}
	CommandIndex int
}

const (
	FOLLOWER  int32 = 1
	CANDICATE int32 = 2
	LEADER    int32 = 3
)

//
// A Go object implementing a single Raft peer.
//
type Raft struct {
	mu        sync.Mutex // Lock to protect shared access to this peer's state
	voteLock  sync.Mutex
	peers     []*labrpc.ClientEnd // RPC end points of all peers
	persister *Persister          // Object to hold this peer's persisted state
	me        int                 // this peer's index into peers[]
	dead      int32               // set by Kill()

	// Your data here (2A, 2B, 2C).
	// Look at the paper's Figure 2 for a description of what
	// state a Raft server must maintain.

	// Persistent state
	CurrentTerm int32 // 当前的任期
	VoteFor     int   //candidateId that received vote in current term (or null if none)
	Logs        []Log

	// Volatile state
	CommitIndex int
	LastApplied int

	// Volatile state on leaders
	NextIndex  []int
	MatchIndex []int

	// 我自己添加的状态
	CurrentState           int32 // 当前的状态
	HeartBeatTicker        *time.Ticker
	HeartBeatTimeOutTicker *time.Ticker
	VoteTimeOutTicker      *time.Ticker
	VoteTime               int
	HeartBeatSend          int
	HeartBeatCheck         int
}

//func (rf *Raft) String() string {
//	b, err := json.Marshal(*rf)
//	if err != nil {
//		return fmt.Sprintf("%+v", *rf)
//	}
//	var out bytes.Buffer
//	err = json.Indent(&out, b, "", "    ")
//	if err != nil {
//		return fmt.Sprintf("%+v", *rf)
//	}
//	return out.String()
//}
//
//func (r *RequestVoteArgs) String() string {
//	b, err := json.Marshal(*r)
//	if err != nil {
//		return fmt.Sprintf("%+v", *r)
//	}
//	var out bytes.Buffer
//	err = json.Indent(&out, b, "", "    ")
//	if err != nil {
//		return fmt.Sprintf("%+v", *r)
//	}
//	return out.String()
//}
//
//func (r *RequestVoteReply) String() string {
//	b, err := json.Marshal(*r)
//	if err != nil {
//		return fmt.Sprintf("%+v", *r)
//	}
//	var out bytes.Buffer
//	err = json.Indent(&out, b, "", "    ")
//	if err != nil {
//		return fmt.Sprintf("%+v", *r)
//	}
//	return out.String()
//}

type Log struct {
	Term    int
	Command string
}

// GetState return currentTerm and whether this server
// believes it is the leader.
func (rf *Raft) GetState() (int, bool) {
	// Your code here (2A).
	state := atomic.LoadInt32(&rf.CurrentState)
	term := atomic.LoadInt32(&rf.CurrentTerm)
	return int(term), state == LEADER
}

//
// save Raft's persistent state to stable storage,
// where it can later be retrieved after a crash and restart.
// see paper's Figure 2 for a description of what should be persistent.
//
func (rf *Raft) persist() {
	// Your code here (2C).
	// Example:
	// w := new(bytes.Buffer)
	// e := labgob.NewEncoder(w)
	// e.Encode(rf.xxx)
	// e.Encode(rf.yyy)
	// data := w.Bytes()
	// rf.persister.SaveRaftState(data)
}

//
// restore previously persisted state.
//
func (rf *Raft) readPersist(data []byte) {
	if data == nil || len(data) < 1 { // bootstrap without any state?
		return
	}
	// Your code here (2C).
	// Example:
	// r := bytes.NewBuffer(data)
	// d := labgob.NewDecoder(r)
	// var xxx
	// var yyy
	// if d.Decode(&xxx) != nil ||
	//    d.Decode(&yyy) != nil {
	//   error...
	// } else {
	//   rf.xxx = xxx
	//   rf.yyy = yyy
	// }
}

// RequestVoteArgs
// example RequestVote RPC arguments structure.
// field names must start with capital letters!
//
type RequestVoteArgs struct {
	// Your data here (2A, 2B).
	// 候选人的任期号
	Term int
	// 请求选票的候选人 id
	CandidateId int
	// 候选人最后日志条目的索引值
	LastLogIndex int
	// 候选人最后日志条目的任期号
	LastLogTerm int
}

// RequestVoteReply
// example RequestVote RPC reply structure.
// field names must start with capital letters!
//
type RequestVoteReply struct {
	// Your data here (2A).
	// 当前任期号，以便候选人更新自己的任期号
	Term int
	// 是否同意这次选票
	VoteGranted bool
}

// RequestVote
// example RequestVote RPC handler.
//
func (rf *Raft) RequestVote(args *RequestVoteArgs, reply *RequestVoteReply) {
	rf.voteLock.Lock()
	defer rf.voteLock.Unlock()
	term := atomic.LoadInt32(&rf.CurrentTerm)
	lastLogIndex := len(rf.Logs)
	lastLogTerm := rf.Logs[len(rf.Logs)-1].Term

	if int(term) > args.Term {
		reply.VoteGranted = false
		reply.Term = int(term)
		return
	}
	// 收到的 rpc 里的 term 比自己的大，因此将状态转为跟随者
	atomic.StoreInt32(&rf.CurrentState, FOLLOWER)

	if rf.VoteFor == args.CandidateId || rf.VoteFor == -1 {
		if args.LastLogTerm > lastLogTerm || (args.LastLogTerm == lastLogTerm && args.LastLogIndex >= lastLogIndex) {
			reply.VoteGranted = true
			reply.Term = args.Term
			atomic.StoreInt32(&rf.CurrentTerm, int32(args.Term))
			logger.Trace(rf.me, " 收到了 ", args.CandidateId, " 的选举请求, 同意")

			// 收到选举请求后，需要抑制自己进行选举，否则可能导致不断地发起选举
			rf.VoteTimeOutTicker.Reset(time.Duration(rf.VoteTime) * time.Millisecond)
			return
		}
	}

	reply.VoteGranted = false
	reply.Term = args.Term
	logger.Trace(rf.me, " 收到了 ", args.CandidateId, " 的选举请求, 并且不同意")
}

//
// example code to send a RequestVote RPC to a server.
// server is the index of the target server in rf.peers[].
// expects RPC arguments in args.
// fills in *reply with RPC reply, so caller should
// pass &reply.
// the types of the args and reply passed to Call() must be
// the same as the types of the arguments declared in the
// handler function (including whether they are pointers).
//
// The labrpc package simulates a lossy network, in which servers
// may be unreachable, and in which requests and replies may be lost.
// Call() sends a request and waits for a reply. If a reply arrives
// within a timeout interval, Call() returns true; otherwise
// Call() returns false. Thus Call() may not return for a while.
// A false return can be caused by a dead server, a live server that
// can't be reached, a lost request, or a lost reply.
//
// Call() is guaranteed to return (perhaps after a delay) *except* if the
// handler function on the server side does not return.  Thus there
// is no need to implement your own timeouts around Call().
//
// look at the comments in ../labrpc/labrpc.go for more details.
//
// if you're having trouble gecztting RPC to work, check that you've
// capitalized all field names in structs passed over RPC, and
// that the caller passes the address of the reply struct with &, not
// the struct itself.
//
// func (rf *Raft) sendRequestVote(server int, args *RequestVoteArgs, reply *RequestVoteReply) bool {
// 	ok := rf.peers[server].Call("Raft.RequestVote", args, reply)
// 	return ok
// }

type AppendEntriesArgs struct {
	// leader's term
	Term         int
	LeaderId     int
	PrevLogIndex int
	PrevLogTerm  int
	Entries      []int
	LeaderCommit int
}
type AppendEntriesReply struct {
	Term    int
	Success bool
}

// AppendEntries 心跳通知、日志追加 RPC
func (rf *Raft) AppendEntries(args *AppendEntriesArgs, reply *AppendEntriesReply) {
	term := atomic.LoadInt32(&rf.CurrentTerm)
	logger.Trace(rf.me, " 收到 id ", args.LeaderId, "的心跳请求，其任期为 ", args.Term, " 自己的任期为 ", term)
	if args.Term < int(term) {
		// 自己的任期号比发来的心跳的大
		reply.Term = int(rf.CurrentTerm)
		reply.Success = false
		return
	}

	reply.Term = args.Term
	reply.Success = true
	atomic.StoreInt32(&rf.CurrentTerm, int32(args.Term))
	atomic.StoreInt32(&rf.CurrentState, FOLLOWER)
	// 收到心跳后，应该抑制其变为 candicater, 并且抑制其发起选举
	rf.HeartBeatTimeOutTicker.Reset(time.Duration(rf.HeartBeatCheck) * time.Millisecond)
	rf.VoteTimeOutTicker.Reset(time.Duration(rf.VoteTime) * time.Millisecond)
}

//
// the service using Raft (e.g. a k/v server) wants to start
// agreement on the next command to be appended to Raft's log. if this
// server isn't the leader, returns false. otherwise start the
// agreement and return immediately. there is no guarantee that this
// command will ever be committed to the Raft log, since the leader
// may fail or lose an election. even if the Raft instance has been killed,
// this function should return gracefully.
//
// the first return value is the index that the command will appear at
// if it's ever committed. the second return value is the current
// term. the third return value is true if this server believes it is
// the leader.
//
// 让 raft 把 command 添加到 log 中
func (rf *Raft) Start(command interface{}) (int, int, bool) {
	index := -1
	term := -1
	isLeader := true

	// Your code here (2B).

	return index, term, isLeader
}

//
// the tester doesn't halt goroutines created by Raft after each test,
// but it does call the Kill() method. your code can use killed() to
// check whether Kill() has been called. the use of atomic avoids the
// need for a lock.
//
// the issue is that long-running goroutines use memory and may chew
// up CPU time, perhaps causing later tests to fail and generating
// confusing debug output. any goroutine with a long-running loop
// should call killed() to check whether it should stop.
//
func (rf *Raft) Kill() {
	atomic.StoreInt32(&rf.dead, 1)
	atomic.StoreInt32(&rf.CurrentState, FOLLOWER)
	// Your code here, if desired.
}

func (rf *Raft) killed() bool {
	z := atomic.LoadInt32(&rf.dead)
	return z == 1
}

//
// the service or tester wants to create a Raft server. the ports
// of all the Raft servers (including this one) are in peers[]. this
// server's port is peers[me]. all the servers' peers[] arrays
// have the same order. persister is a place for this server to
// save its persistent state, and also initially holds the most
// recent saved state, if any. applyCh is a channel on which the
// tester or service expects Raft to send ApplyMsg messages.
// Make() must return quickly, so it should start goroutines
// for any long-running work.
//
// peers: 一组网络标识符
// me: 本端在 peers 中的下标
func Make(peers []*labrpc.ClientEnd, me int, persister *Persister, applyCh chan ApplyMsg) *Raft {
	rf := &Raft{}
	rf.peers = peers
	rf.persister = persister
	rf.me = me

	// Your initialization code here (2A, 2B, 2C).
	rf.CurrentTerm = 0
	rf.CurrentState = FOLLOWER
	rf.VoteFor = -1

	rf.HeartBeatCheck = 220
	rf.HeartBeatSend = 105
	rf.VoteTime = rand.Intn(150) + 150

	rf.VoteTimeOutTicker = time.NewTicker(time.Duration(rf.VoteTime))
	rf.HeartBeatTimeOutTicker = time.NewTicker(time.Duration(rf.HeartBeatCheck))
	rf.HeartBeatTicker = time.NewTicker(time.Duration(rf.HeartBeatSend))

	rf.Logs = append(rf.Logs, Log{
		Term:    0,
		Command: "Init",
	})

	// 启动心跳计时器，超时则发起选举，自己变成候选人状态
	ctx := context.Background()
	// follower 需要定时接收心跳，没收到心跳就变成 candicator
	go rf.CheckHeartBeatsClock(ctx)
	// leader 需要定时发送心跳
	go rf.SendHearBeatsClock(ctx)
	// candidate 身份需要发起一轮选举
	go rf.startVote(ctx)

	// initialize from state persisted before a crash
	rf.readPersist(persister.ReadRaftState())

	return rf
}
// startVote
// 发起一轮投票，实现方式为一个死循环，如果自己的状态为 candidate，并且计时器超时了，则发起投票
func (rf *Raft) startVote(ctx context.Context) {
	for {
		select {
		case <-rf.VoteTimeOutTicker.C:
			state := atomic.LoadInt32(&rf.CurrentState)
			if state != CANDICATE || rf.killed() {
				continue
			}
			// 重置心跳计时器，防止自己在选举的过程中超时，然后又发起一轮选举
			rf.HeartBeatTimeOutTicker.Reset(time.Duration(rf.HeartBeatCheck) * time.Millisecond)
			// 自增任期号
			atomic.AddInt32(&rf.CurrentTerm, 1)
			term := atomic.LoadInt32(&rf.CurrentTerm)
			logger.Debug("id ", rf.me, " 开始选举,自己的任期为 ", term)

			// 自己给自己投票
			cnt := 1
			rf.VoteFor = rf.me
			args := RequestVoteArgs{
				CandidateId:  rf.me,
				Term:         int(term),
				LastLogIndex: len(rf.Logs),
				LastLogTerm:  rf.Logs[len(rf.Logs)-1].Term,
			}
			// 在这里发起一次选举，向所有的 server 发送选举请求
			replies := make([]RequestVoteReply, len(rf.peers))
			for idx, server := range rf.peers {
				if idx != rf.me {
					idx := idx
					server := server
					go func() {
						logger.Trace(rf.me, " 向 ", idx, " 发送了选举请求")
						ok := server.Call("Raft.RequestVote", &args, &(replies[idx]))
						if !ok {
							logger.Warn(rf.me, "给 ", idx, " 发的选举没有得到回复")
						} else {
							logger.Trace(rf.me, " 收到的选举回复, CurrentTerm:", replies[idx].Term, ", Agree? ", replies[idx].VoteGranted)
						}
					}()
				}
			}
			time.Sleep(100 * time.Millisecond)

			for idx, v := range replies {
				if idx != rf.me {
					if v.VoteGranted {
						cnt++
					}
					if v.Term > int(term) {
						atomic.StoreInt32(&rf.CurrentTerm, int32(v.Term))
						// 这里可以直接变成 follower，因为就算所有的服务器都是 follower，他们也会因为心跳超时而重新发起一轮选举
						atomic.StoreInt32(&rf.CurrentState, FOLLOWER)
						logger.Info("id ", rf.me, "选举失败, 收到的任期号为 ", v.Term, ", 自己的任期号为", term, "收到的任期是 ", v.Term)
						goto CON
					}
				}
			}

			logger.Debug("id ", rf.me, "获得选票", cnt, "张，总共有", len(rf.peers), "人")
			// 获胜则变成 leader，没获胜则依旧是 candidate，继续选举
			if cnt > len(rf.peers)/2 {
				atomic.StoreInt32(&rf.CurrentState, LEADER)
			}
		CON:
			continue
		case <-ctx.Done():
			return
		}
	}

}

// CheckHeartBeatsClock
// 检查心跳计时器，如果一切正常的话，每次收到的心跳都会抑制这个计时器
// 如果没有收到心跳，则转变自己的状态为 candidate
func (rf *Raft) CheckHeartBeatsClock(ctx context.Context) {
	for {
		select {
		case <-rf.HeartBeatTimeOutTicker.C:
			state := atomic.LoadInt32(&rf.CurrentState)
			// 没收到心跳
			if state == FOLLOWER && !rf.killed() {
				logger.Debug(rf.me, " 没收到心跳QAQ")
				atomic.StoreInt32(&rf.CurrentState, CANDICATE)
				continue
			}
		case <-ctx.Done():
			return
		}
	}
}

// SendHearBeatsClock
// 定时发送心跳  master -> follower
// 如果返回的心跳 CurrentTerm 比自己大，则把自己的状态转变为 Follower，然后退出向所有服务器发送心跳的循环
func (rf *Raft) SendHearBeatsClock(ctx context.Context) {
	for {
		select {
		case <-rf.HeartBeatTicker.C:
			state := atomic.LoadInt32(&rf.CurrentState)
			if state != LEADER || rf.killed() {
				continue
			}
			logger.Debug("id ", rf.me, " 发送心跳")
			term := atomic.LoadInt32(&rf.CurrentTerm)
			args := AppendEntriesArgs{
				Term:     int(term),
				LeaderId: rf.me,
			}

			replies := make([]AppendEntriesReply, len(rf.peers))
			var noReplyNumber int32 = 0
			for idx, server := range rf.peers {
				idx := idx
				server := server
				if idx != rf.me {
					go func() {
						ok := server.Call("Raft.AppendEntries", &args, &(replies[idx]))
						if !ok {
							atomic.AddInt32(&noReplyNumber, 1)
							logger.Warn(rf.me, "给 ", idx, " 发的心跳没有得到回复")
							return
						}
						if replies[idx].Term > int(term) {
							// 收到的 CurrentTerm 比自己的大
							atomic.StoreInt32(&rf.CurrentTerm, int32(replies[idx].Term))
							atomic.StoreInt32(&rf.CurrentState, FOLLOWER)
							return
						}
					}()
				}
			}
			time.Sleep(100 * time.Millisecond)

			// 处理在 lab 2a中的特殊情况：出现分区后，自己要降级为 follower
			if int(atomic.LoadInt32(&noReplyNumber)) > len(rf.peers)/2 {
				logger.Error(rf.me, "发送的心跳有", noReplyNumber, "人都没有回复")
				atomic.StoreInt32(&rf.CurrentState, FOLLOWER)
			}
		case <-ctx.Done():
			return
		}
	}
}
