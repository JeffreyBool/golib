package utils

import (
	"fmt"
	"sync"
	"context"
)

// 一个简易的协程池实现
type JobContextFunc func(ctx context.Context, str string) bool

type GContextPool struct {
	sync.Mutex

	// 任务队列
	JobChan chan string
	// 结果队列
	ResultChan chan bool
	// 协程池的大小
	Size int
	// 已经完成的任务量
	FinishCount int
	// 目标任务量
	TargetCount int
	// ResultChan 是否Close
	IsClose bool
	// 取消函数
	CancelFunc context.CancelFunc
	// Context
	Ctx context.Context
}

func NewGContextPool(size int) *GContextPool {
	pool := GContextPool{}
	pool.JobChan = make(chan string, SIZE)
	pool.ResultChan = make(chan bool, SIZE)
	pool.Size = size
	pool.IsClose = false
	pool.Ctx, pool.CancelFunc = context.WithCancel(context.Background())
	return &pool
}

func (p *GContextPool) ApplyAsync(f JobContextFunc, slice []string) <-chan bool {

	p.TargetCount = len(slice)
	// Producer
	go p.Produce(slice)
	// consumer
	for i := 0; i < p.Size; i++ {
		go p.Consume(f)
	}

	return p.ResultChan
}

func (p *GContextPool) Cancel() {
	fmt.Println("Context Cancel")
	p.CancelFunc()
}

func (p *GContextPool) Produce(slice []string) {
	for _, key := range slice {
		p.JobChan <- key
	}
	close(p.JobChan)
}

func (p *GContextPool) Consume(f JobContextFunc) {
	for job := range p.JobChan {
		defer func() {
			err := recover()
			if err != nil {
				fmt.Errorf("execute job error, %v", err)
				p.ResultChan <- false
				p.FinishOne()
			}
		}()

		select {
		case <-p.Ctx.Done():
			p.ResultChan <- false
		default:
			// 没有结束 ... 执行 ...
			p.ResultChan <- f(p.Ctx, job)
		}
		p.FinishOne()
	}
	p.TryClose()
}

// 记录完成了一个任务
func (p *GContextPool) FinishOne() {
	p.Lock()
	p.FinishCount++
	p.Unlock()
}

// 关闭结果Channel
func (p *GContextPool) TryClose() {
	p.Lock()
	if p.FinishCount == p.TargetCount && !p.IsClose {
		close(p.ResultChan)
		p.IsClose = true
	}
	p.Unlock()
}
