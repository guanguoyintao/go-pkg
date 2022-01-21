package main

import (
	ctx "context"
	"fmt"
	timewheel "main/time_wheel/time_wheel"
	"sync"
	"time"
)

type TaskTest struct {
	//R chan int
}

func MessageLogHandler(task *timewheel.TaskStatus) {
	fmt.Printf("task %s is completed, status is %t\n", task.Key, task.Ok)
}

var tw = timewheel.NewTimeWheel(ctx.Background(), 1*time.Second, 3600)

func (t *TaskTest) TaskExec(p interface{}) *timewheel.TaskStatus {

	//time.Sleep(2 * time.Second)
	fmt.Println("oooooooo")
	//select {
	//case i := <-t.R:
	//	fmt.Println(i)
	//}
	return nil

}

type task2 struct {
}

func (t *task2) TaskExec(p interface{}) *timewheel.TaskStatus {
	time.Sleep(5 * time.Second)
	fmt.Println("2222222")
	tw.AddTimer(10*time.Second, 0, "test", t, "")
	return nil
}

var wg = &sync.WaitGroup{}

func main() {
	wg.Add(2)
	//r := make(chan int, 10)
	t := &TaskTest{}
	t2 := &task2{}
	go func() {
		tw.Start()
		wg.Done()

	}()
	tw.AddTimer(0, 0, "ocean_engine", t, "")
	tw.AddTimer(0, 0, "test", t2, "")
	tw.SetTaskCallBack(MessageLogHandler)

	wg.Done()

	//for i := 0; i < 10; i++ {
	//	r <- i
	//}
	wg.Wait()
}
