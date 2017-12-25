package awaiter

import (
	"fmt"
	"time"
)

//Looper 使用awaiter实现一个类似定时器的功能 定时执行DO
type Looper struct {
	awaiter Awaiter
}

//LooperDo Looper的定时执行函数
type LooperDo func(lp *Looper)

//NewLooper 返回一个Looper
func NewLooper(dt time.Duration, do LooperDo) *Looper {
	looper := &Looper{
		awaiter: nil,
	}
	looper.awaiter = New(func(aw Awaiter, arg ...interface{}) (interface{}, error) {
	loop:
		for {
			select {
			case <-time.After(dt):
				looper.invoke(do)
			case <-aw.CancelWaiter():
				break loop
			}
		}
		return nil, nil
	})
	return looper
}

func (lp *Looper) invoke(do LooperDo) (e error) {
	defer func() {
		if err := recover(); err != nil {
			e = fmt.Errorf("%+v", err)
		}
	}()
	do(lp)
	return nil
}

//Exit 停止定时器
func (lp *Looper) Exit() error {
	_, err := lp.awaiter.Cancel().Await()
	return err
}

//IsCancellationRequested 是否取消
func (lp *Looper) IsCancellationRequested() bool {
	return lp.awaiter.IsCancellationRequested()
}
