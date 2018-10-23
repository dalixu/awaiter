/*
Package awaiter 模拟C#的Task 不需要直接使用go func
*/
package awaiter

import (
	"fmt"
	"sync"
	"time"
)

//Awaiter Awaiter接口
type Awaiter interface {
	Await() (interface{}, error)
	Waiter() <-chan bool
	IsFinished() bool
	ContinueWith(do ContinueDo, arg ...interface{}) Awaiter
	Cancel() Awaiter
	CancelWaiter() <-chan bool
	IsCancellationRequested() bool
}

//Do Awaiter的调用函数类型
type Do func(aw Awaiter, arg ...interface{}) (interface{}, error)

//ContinueDo continue的调用函数类型
type ContinueDo func(aw Awaiter, prev Awaiter, arg ...interface{}) (interface{}, error)

//New 返回Awaiter
func New(do Do, arg ...interface{}) Awaiter {
	return newCommonAwaiter().async(do, arg...)
}

//InfiniteDuration 无限等待的时间
const (
	InfiniteDuration time.Duration = 1<<63 - 1
)

//Delay 延时处理 支持撤销 内部使用的time.After d为InfiniteDuration 则无限等待
func Delay(d time.Duration) Awaiter {
	var tc *time.Timer
	if d != InfiniteDuration {
		tc = time.NewTimer(d)
	}
	return newCommonAwaiter().async(func(aw Awaiter, arg ...interface{}) (interface{}, error) {
		if tc != nil {
			select {
			case <-arg[0].(*time.Timer).C:
			case <-aw.CancelWaiter():
				arg[0].(*time.Timer).Stop() //及时回收Timer
			}
		} else {
			select {
			case <-aw.CancelWaiter():
			}
		}

		return nil, nil
	}, tc)
}

//CommonAwaiter 通用的Awaiter封装goroutine以便能够简单的返回值
//New(...).Await()
type CommonAwaiter struct {
	finishedChan    chan bool
	reason          error       //结束时返回的错误码
	result          interface{} //结束时返回的RoutineDo结果
	cancelChan      chan bool
	continueMutex   *sync.Mutex
	continueAwaiter *CommonAwaiter
	continueDo      Do
	continueArg     []interface{}
}

func newCommonAwaiter() *CommonAwaiter {
	return &CommonAwaiter{
		finishedChan:    make(chan bool),
		reason:          nil,
		result:          nil,
		cancelChan:      make(chan bool),
		continueMutex:   &sync.Mutex{},
		continueAwaiter: nil,
		continueDo:      nil,
		continueArg:     nil,
	}
}

//Await 等待Awaiter返回
func (aw *CommonAwaiter) Await() (interface{}, error) {
	<-aw.finishedChan
	return aw.result, aw.reason
}

//Waiter 返回管道 方便在select中等待
//use: case <- Waiter() not case v := <-Waiter()
//因为运行结束后直接Close了 并没有写入值 这里把Chan当作纯粹的bool
func (aw *CommonAwaiter) Waiter() <-chan bool {
	return aw.finishedChan
}

//IsFinished RoutineDo 是否结束
func (aw *CommonAwaiter) IsFinished() bool {
	select {
	case <-aw.finishedChan:
		return true
	default:
		return false
	}
}

//ContinueWith 方便写pipeline 注意 do运行时参数传递时会把之前的Awaiter放到arg前 一起传给do
func (aw *CommonAwaiter) ContinueWith(do ContinueDo, arg ...interface{}) Awaiter {
	aw.continueMutex.Lock()
	defer aw.continueMutex.Unlock()
	//已经注册过continueWith函数 禁止再注册
	if aw.continueAwaiter != nil {
		return nil
	}
	aw.continueAwaiter = newCommonAwaiter()
	param := make([]interface{}, 0, len(arg)+1)
	param = append(append(param, aw), arg...)
	aw.continueDo = func(aw Awaiter, arg ...interface{}) (interface{}, error) {
		return do(aw, arg[0].(Awaiter), arg[1:]...)
	}
	aw.continueArg = param
	if aw.IsFinished() {
		aw.continueAwaiter.async(aw.continueDo, aw.continueArg...)
	}
	return aw.continueAwaiter
}

//Cancel 要求AwaiterDo尽快cancel退出 但并不保证RoutineDo以Cancel退出
//只能调用一次
func (aw *CommonAwaiter) Cancel() Awaiter {
	close(aw.cancelChan)
	return aw
}

//CancelWaiter 返回撤销管道 方便在select中使用
func (aw *CommonAwaiter) CancelWaiter() <-chan bool {
	return aw.cancelChan
}

//IsCancellationRequested AwaiterDo应该不时的调用这个函数以便知道自己是否需要及时退出 如果返回true应该尽快退出
func (aw *CommonAwaiter) IsCancellationRequested() bool {
	select {
	case <-aw.cancelChan: //close 或者有值的话 这里会直接返回
		return true
	default:
		return false
	}
}

func (aw *CommonAwaiter) async(do Do, arg ...interface{}) *CommonAwaiter {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				if e, ok := r.(error); ok {
					aw.setResult(nil, e)
				} else {
					aw.setResult(nil, fmt.Errorf("Awaiter:Unknown Error:%v", r))
				}
			}
			close(aw.finishedChan)
			aw.continueMutex.Lock()
			if aw.continueAwaiter != nil {
				aw.continueAwaiter.async(aw.continueDo, aw.continueArg...)
			}
			aw.continueMutex.Unlock()
		}()
		aw.setResult(do(aw, arg...))
	}()
	return aw
}

func (aw *CommonAwaiter) setResult(result interface{}, err error) {
	aw.reason = err
	aw.result = result
}

//CollectionAwaiter Awaiter的集合用于处理多个Awaiter
type CollectionAwaiter struct {
	*CommonAwaiter
	children []Awaiter
}

//Cancel 覆盖CommonAwaiter Cancel
func (ca *CollectionAwaiter) Cancel() Awaiter {
	close(ca.cancelChan)
	for _, c := range ca.children {
		c.Cancel()
	}

	return ca
}

//WhenAny 有1个完成则返回完成的Awaiter
func WhenAny(aw ...Awaiter) Awaiter {
	parent := &CollectionAwaiter{CommonAwaiter: newCommonAwaiter(), children: aw}
	parent.async(func(aw Awaiter, arg ...interface{}) (interface{}, error) {
		aws := arg[0].([]Awaiter)
		if len(aws) == 0 {
			return nil, nil
		}
		// for {
		// 	for _, a := range aws {
		// 		if a.IsFinished() {
		// 			return a, nil
		// 		}
		// 	}
		// 	runtime.Gosched()//容易产生CPU占用率过高
		// }
		chans := make(chan Awaiter, len(aws))
		for _, a := range aws {
			current := a
			go func() {
				current.Await()
				chans <- current
			}()
		}
		select {
		case a := <-chans:
			return a, nil
		}
	}, parent.children)

	return parent
}

//WhenAll 等待所有的完成并返回[]Awaiter
func WhenAll(aw ...Awaiter) Awaiter {
	parent := &CollectionAwaiter{CommonAwaiter: newCommonAwaiter(), children: aw}
	parent.async(func(aw Awaiter, arg ...interface{}) (interface{}, error) {
		aws := arg[0].([]Awaiter)
		for _, a := range aws {
			a.Await()
		}
		return aws, nil
	}, parent.children)
	return parent
}
