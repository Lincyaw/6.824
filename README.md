# 使用下面的命令，将包管理切换成以前的模式
`go env -w GO111MODULE=off`

否则会因为go module的原因，无法运行
# LAB1

## step1
本次实验的任务是实现一个分布式的MapReduce。只需要修改`mr/master.go`,`mr/worker.go`,`mr/rpc.go`三个文件。

根据提示，首先需要修改`workder.go`中的`Worker*()`函数来发送一个RPC给master，让master给点活干；
master回应一个还没开始做的 map 任务, 不至于让worker闲的没事干。

接下来修改worker，让他能够读文件，然后调用写好的 map 函数来干活。

### 看代码结构
上面提到的几个文件的目录结构为

```
main/mrworker.go
main/mrmaster.go

mr/master.go
mr/rpc.go
mr/worker.go

mrapps/wc.go
```

其中，`mrapps`文件夹中实现的是在不同的业务场景下的不同的`map`以及`reduce`函数。
这些都将以插件(plugin)的形式在运行时使用。

`main`文件夹的几个文件主要是调用了`mr`文件夹中我们实现的几个函数。他也提供了基本的工具函数，比如如何加载一个插件。

在我的第二个commit里，能够运行一个示例的程序。可以看一下他的函数调用流程：

MakeMaster -> server, 从而开始监听特定的端口号。在master中，一直循环这个任务，直到master.Done();

Worker这边呢，则发送了一个示例的请求给master。因此我们需要修改worker发送的请求，然后在master里处理对应的请求。