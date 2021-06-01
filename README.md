# 实验记录

- [lab1 MapReduce](doc/Lab1.md)

- [lab2a Raft](doc/Lab2a.md)

## 环境相关的问题

### 使用下面的命令，将包管理切换成以前的模式

`go env -w GO111MODULE=off`

否则会因为go module的原因，无法运行

### 使用 logrus 格式化输出

下载
```go
go get -v github.com/sirupsen/logrus
```

会遇到`cannot find package "golang.org/x/sys/unix`的问题，自己手动下一个
```sh
mkdir -p $GOPATH/src/golang.org/x
cd $GOPATH/src/golang.org/x
git clone https://github.com/golang/sys.git
```

### 用法

Logrus 有七个日志级别：Trace、Debug、Info、Warning、Error、Fatal 和 Panic。

```go
// 导入包
log "github.com/sirupsen/logrus"

// 在初始化函数中添加限制等级，这里只会输出 warning 以上的 log
log.SetLevel(log.WarnLevel)

// 一些例子
log.Trace("Something very low level.")
log.Debug("Useful debugging information.")
log.Info("Something noteworthy happened!")
log.Warn("You should probably take a look at this.")
log.Error("Something failed but I'm not quitting.")
// Calls os.Exit(1) after logging
log.Fatal("Bye.")
// Calls panic() after logging
log.Panic("I'm bailing.")
```