# 使用下面的命令，将包管理切换成以前的模式
`go env -w GO111MODULE=off`

否则会因为go module的原因，无法运行
# LAB1
本次实验的任务是实现一个分布式的MapReduce。只需要修改`mr/master.go`,`mr/worker.go`,`mr/rpc.go`三个文件。