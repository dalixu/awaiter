package awaiter

import (
	"errors"
	"testing"
	"time"
)

func TestAwaiterPanic(t *testing.T) {
	v, err := New(func(aw Awaiter, arg ...interface{}) (interface{}, error) {
		panic(errors.New("test panic"))
	}).Await()
	if err == nil || v != nil {
		t.Error("awaiter panic fail")
	} else {
		t.Log(err)
	}

	v, err = New(func(aw Awaiter, arg ...interface{}) (interface{}, error) {
		panic(100)
	}).Await()
	if err == nil || v != nil {
		t.Error("awaiter panic fail")
	} else {
		t.Log(err)
	}
}

func TestAwaiterArg(t *testing.T) {
	//测试aw参数传递是否正确
	aw := New(func(aw Awaiter, arg ...interface{}) (interface{}, error) {
		if len(arg) == 3 && arg[0].(int) == 1 && arg[1].(int) == 2 && arg[2].(int) == 3 {
			return true, nil
		}
		return false, errors.New("awaiter arg fail")
	}, 1, 2, 3)

	v, err := aw.Await()
	if v.(bool) == true && err == nil {
		t.Log("awater arg success 1")
	} else {
		t.Error(err)
	}
	//不传递参数
	aw = New(func(aw Awaiter, arg ...interface{}) (interface{}, error) {
		if len(arg) == 0 {
			return true, nil
		}
		return false, errors.New("awaiter  send param fail")
	})
	v, err = aw.Await()
	if v.(bool) == true && err == nil {
		t.Log("awater arg success 2")
	} else {
		t.Error(err)
	}
	//传递nil参数
	aw = New(func(aw Awaiter, arg ...interface{}) (interface{}, error) {
		if len(arg) == 1 && arg[0] == nil {
			return true, nil
		}
		return false, errors.New("awaiter  send param fail")
	}, nil)
	v, err = aw.Await()
	if v.(bool) == true && err == nil {
		t.Log("awater arg success 3")
	} else {
		t.Error(err)
	}
}

func TestAwaiterCancel(t *testing.T) {
	v, err := New(func(aw Awaiter, arg ...interface{}) (interface{}, error) {
		count := 0
		for {
			if aw.IsCancellationRequested() {
				//多次调用NeedCancel不会引起panic
				aw.IsCancellationRequested()
				aw.IsCancellationRequested()
				return nil, errors.New("it's cancel error")
			}
			time.Sleep(1 * time.Second)
			count++
			if count >= 5 {
				return nil, errors.New("test cancel fail")
			}
		}
	}).Cancel().Await()

	if err == nil || v != nil {
		t.Error("awaiter cancel fail")
	} else {
		t.Log(err)
	}
}

func TestAwaiterCancelWaiter(t *testing.T) {
	v, err := New(func(aw Awaiter, arg ...interface{}) (interface{}, error) {
	loop:
		for {
			select {
			case <-aw.CancelWaiter():
				break loop
			}
		}
		return nil, errors.New("it's cancel error")
	}).Cancel().Await()

	if err == nil || v != nil {
		t.Error("awaiter cancel fail")
	} else {
		t.Log(err)
	}
}

func TestAwaiterResult(t *testing.T) {
	v, err := New(func(aw Awaiter, arg ...interface{}) (interface{}, error) {
		return arg[0].(string), nil
	}, "1").Cancel().Await()

	if err != nil || v == nil {
		t.Error("awaiter result fail", err, v)
	} else {
		t.Log(v, err)
	}
}

func TestAwaiterIsFinished(t *testing.T) {
	c := make(chan bool, 1)
	aw := New(func(aw Awaiter, arg ...interface{}) (interface{}, error) {
		select {
		case <-c:
		}
		return true, nil
	})
	if aw.IsFinished() {
		t.Error("awaiter finished fail 0")
	}
	if aw.IsFinished() {
		t.Error("awaiter finished fail 1")
	}
	c <- true
	aw.Await()
	if !aw.IsFinished() {
		t.Error("awaiter finished fail 2")
	}
	if !aw.IsFinished() {
		t.Error("awaiter finished fail 3")
	}
	v, err := aw.Await()
	if err != nil || v == nil {
		t.Error("awaiter result fail")
	} else {
		t.Log(v, err)
	}
}

func TestAwaiterContinueArg(t *testing.T) {
	//测试awcontinueWith参数传递是否正确
	aw := New(func(aw Awaiter, arg ...interface{}) (interface{}, error) {
		return nil, nil
	}).ContinueWith(func(aw Awaiter, prev Awaiter, arg ...interface{}) (interface{}, error) {
		if len(arg) == 3 && prev.IsFinished() &&
			arg[0].(int) == 1 && arg[1].(int) == 2 && arg[2].(int) == 3 {
			return true, nil
		}

		return false, errors.New("awaiter continuewith send param fail")
	}, 1, 2, 3)
	v, err := aw.Await()
	if v.(bool) == true && err == nil {
		t.Log("awater continue arg success 1")
	} else {
		t.Error(err)
	}
	//不传递参数
	aw = New(func(aw Awaiter, arg ...interface{}) (interface{}, error) {
		return nil, nil
	}).ContinueWith(func(aw Awaiter, prev Awaiter, arg ...interface{}) (interface{}, error) {
		if len(arg) == 0 && prev.IsFinished() {
			return true, nil
		}
		return false, errors.New("awaiter continuewith send param fail")
	})
	v, err = aw.Await()
	if v.(bool) == true && err == nil {
		t.Log("awater continue arg success 2")
	} else {
		t.Error(err)
	}
	//传递nil参数
	aw = New(func(aw Awaiter, arg ...interface{}) (interface{}, error) {
		return nil, nil
	}).ContinueWith(func(aw Awaiter, prev Awaiter, arg ...interface{}) (interface{}, error) {
		if len(arg) == 1 && prev.IsFinished() &&
			arg[0] == nil {
			return true, nil
		}
		return false, errors.New("awaiter continuewith send param fail")
	}, nil)
	v, err = aw.Await()
	if v.(bool) == true && err == nil {
		t.Log("awater continue arg success 3")
	} else {
		t.Error(err)
	}
}

//测试aw已经完成后 能否执行continueWith
func TestAwaiterContinueWithIfFinished(t *testing.T) {

	aw := New(func(aw Awaiter, arg ...interface{}) (interface{}, error) {
		return nil, nil
	})
	aw.Await()
	c := make(chan bool)
	aw.ContinueWith(func(aw Awaiter, prev Awaiter, arg ...interface{}) (interface{}, error) {
		c <- true
		return nil, nil
	})
	select {
	case <-c:
		t.Log("awaiter continue success")
	case <-time.After(2 * time.Second):
		t.Error("awaiter continue fail")
	}
}

//测试aw未完成后 能否执行continueWith
func TestAwaiterContinueWithIfRunning(t *testing.T) {
	c := make(chan bool)
	New(func(aw Awaiter, arg ...interface{}) (interface{}, error) {
		<-c
		return nil, nil
	}).ContinueWith(func(aw Awaiter, prev Awaiter, arg ...interface{}) (interface{}, error) {
		c <- false
		return nil, nil
	})
	c <- true
	//aw.Await()
	select {
	case v := <-c:
		if v == false {
			t.Log("awaiter continue success")
		} else {
			t.Error("awaiter continue fail")
		}
	case <-time.After(2 * time.Second):
		t.Error("awaiter continue fail")
	}
}

func TestAwaiterWhenAny(t *testing.T) {
	aw1 := New(func(aw Awaiter, arg ...interface{}) (interface{}, error) {
		return nil, nil
	})
	aw2 := New(func(aw Awaiter, arg ...interface{}) (interface{}, error) {
		select {
		case <-time.After(10 * time.Second):
			return nil, errors.New("TestAwaiterWhenAny: timeout")
		case <-aw.CancelWaiter():
			return nil, nil
		}
	})
	any := WhenAny(aw1, aw2)
	v, e := any.Await()
	if e != nil {
		t.Error(e)
	} else {
		if v.(Awaiter) != aw1 {
			t.Error(e)
		}
	}
}

func TestAwaiterWhenAnyWithCancel(t *testing.T) {
	aw1 := New(func(aw Awaiter, arg ...interface{}) (interface{}, error) {
		select {
		case <-time.After(10 * time.Second):
			return nil, errors.New("TestAwaiterWhenAnyWithCancel: timeout")
		case <-aw.CancelWaiter():
			return nil, nil
		}
	})
	aw2 := New(func(aw Awaiter, arg ...interface{}) (interface{}, error) {
		select {
		case <-time.After(10 * time.Second):
			return nil, errors.New("TestAwaiterWhenAnyWithCancel: timeout")
		case <-aw.CancelWaiter():
			return nil, nil
		}
	})
	any := WhenAny(aw1, aw2)
	v, e := any.Cancel().Await() //v可能是aw1也可能aw2
	if e != nil {
		t.Error(e)
	} else {
		if v.(Awaiter) != aw1 && v.(Awaiter) != aw2 {
			t.Error(e)
		}
		if !v.(Awaiter).IsCancellationRequested() {
			t.Error(e)
		}
	}
}

func TestAwaiterWhenAll(t *testing.T) {
	aw1 := New(func(aw Awaiter, arg ...interface{}) (interface{}, error) {
		return nil, nil
	})
	aw2 := New(func(aw Awaiter, arg ...interface{}) (interface{}, error) {
		select {
		case <-time.After(1 * time.Second):
			return nil, errors.New("TestAwaiterWhenAll: timeout")
		case <-aw.CancelWaiter():
			return nil, nil
		}
	})
	all := WhenAll(aw1, aw2)
	v, e := all.Cancel().Await() //v可能是aw1也可能aw2
	if e != nil {
		t.Error(e)
	} else {
		array := v.([]Awaiter)
		if len(array) != 2 || array[0] != aw1 || array[1] != aw2 {
			t.Error("TestAwaiterWhenAll fail")
		}
	}
}

func TestAwaiterWhenAllWithCancel(t *testing.T) {
	aw1 := New(func(aw Awaiter, arg ...interface{}) (interface{}, error) {
		select {
		case <-time.After(10 * time.Second):
			return nil, errors.New("TestAwaiterWhenAllWithCancel: timeout")
		case <-aw.CancelWaiter():
			return nil, nil
		}
	})
	aw2 := New(func(aw Awaiter, arg ...interface{}) (interface{}, error) {
		select {
		case <-time.After(10 * time.Second):
			return nil, errors.New("TestAwaiterWhenAllWithCancel: timeout")
		case <-aw.CancelWaiter():
			return nil, nil
		}
	})
	all := WhenAll(aw1, aw2)
	v, e := all.Cancel().Await() //v可能是aw1也可能aw2
	if e != nil {
		t.Error(e)
	} else {
		array := v.([]Awaiter)
		if len(array) != 2 || array[0] != aw1 || array[1] != aw2 || !aw1.IsCancellationRequested() || !aw2.IsCancellationRequested() {
			t.Error("TestAwaiterWhenAllWithCancel fail")
		}
	}
}

func TestAwaiterDelay(t *testing.T) {
	now := time.Now()
	Delay(1 * time.Second).Await()
	if (time.Now().Sub(now)) < time.Second {
		t.Error("TestAwaiterDelay fail")
	}
}

func TestAwaiterDelayCancel(t *testing.T) {
	now := time.Now()
	aw := Delay(2 * time.Second)
	select {
	case <-time.After(500 * time.Millisecond):
		aw.Cancel().Await()
	}
	if (time.Now().Sub(now)) >= time.Second {
		t.Error("TestAwaiterDelayCancel fail")
	}
}
