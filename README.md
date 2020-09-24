# common
自用公共库

## eventloop
包含 :<br>
ticker: 每秒能分发100万左右，包含AfterFunc 和 Tick ，简单无脑，适合大部分适用场景<br>
eventloop : 单线程的时间轮询，基于lockfree_queue,每秒投递函数处理能力在1000万级<br>
lockfree_queue: 原子操作队列，拷贝的第三方库的代码

## logger
基于zap改的日志库，按 *天* 分文件写日志

## timer

基于time.ticker 的 millsecond 精度的时间库

## network

自己封装的tcp库。30K 连接只需 700M 内存，多了没测试，用不了那么多

## ulimit 

修改linux 下面 rlimit 限制。改为了999999
