package timewheel

import (
	"container/list"
	ctx "context"
	"fmt"
	"time"
)

type Job interface {
	TaskExec(interface{}) *TaskStatus
}

type TaskCallBack func(*TaskStatus)

type TaskStatus struct {
	Data interface{}
	Key  string
	Ok   bool
}

type TimeWheel struct {
	time              int64 // 时间轮当前时间
	ctx               ctx.Context
	runing            bool
	cycleInterval     time.Duration // 任务周期间隔
	interval          time.Duration
	ticker            *time.Ticker
	slots             [][]*list.List // 槽
	taskCounter       int
	taskMap           map[string]int
	timer             map[interface{}]int
	taskCallBack      TaskCallBack     // 时间轮任务完成回调函数
	taskStatusChannel chan *TaskStatus // 时间轮任务执行状态channel
	currentPos        int              // 当前指针指向哪一个槽
	slotNum           int              // 槽数量
	addTaskChannel    chan *Task       // 新增任务channel
	removeTaskChannel chan string      // 删除任务channel
	stopChannel       chan bool        // 停止定时器channel
}

// Task 延时任务
type Task struct {
	ctx           ctx.Context
	delay         time.Duration // 延迟时间
	cycleInterval time.Duration // 时间轮需要转动几圈
	circle        int           // 时间轮需要转动几圈
	job           Job           // 定时器执行任务函数
	key           string        // 定时器唯一标识, 用于删除定时器
	data          interface{}   // 任务函数参数
	offset        int           // 任务在链表数组中偏移量
}

// NewTimeWheel 创建时间轮
func NewTimeWheel(context ctx.Context, interval time.Duration, slotNum int) *TimeWheel {
	if interval <= 0 || slotNum <= 0 {
		return nil
	}
	tw := &TimeWheel{
		ctx:               context,
		interval:          interval,
		runing:            false,
		slots:             make([][]*list.List, slotNum),
		timer:             make(map[interface{}]int),
		currentPos:        0,
		slotNum:           slotNum,
		addTaskChannel:    make(chan *Task),
		removeTaskChannel: make(chan string),
		stopChannel:       make(chan bool),
		taskMap:           make(map[string]int),
		taskCounter:       -1,
		taskCallBack: func(status *TaskStatus) {

		},
	}

	tw.initSlots()

	return tw
}

func (tw *TimeWheel) SetTaskCallBack(taskCallBack TaskCallBack) {
	tw.taskCallBack = taskCallBack
}

func (tw *TimeWheel) initSlots() {
	for i := 0; i < tw.slotNum; i++ {
		tw.slots[i] = []*list.List{}
	}
}

func (tw *TimeWheel) appendTaskSlots() {
	for i := 0; i < tw.slotNum; i++ {
		tw.slots[i] = append(tw.slots[i], list.New())
	}
}

// Start 启动时间轮
func (tw *TimeWheel) Start() {
	fmt.Println("time wheel started")
	tw.ticker = time.NewTicker(tw.interval)
	tw.start()
}

// Stop 停止时间轮
func (tw *TimeWheel) Stop() {
	tw.stopChannel <- true
}

// AddTimer 添加定时器 key为定时器唯一标识
func (tw *TimeWheel) AddTimer(
	delay time.Duration,
	cycleInterval time.Duration,
	key string,
	job Job,
	data interface{},
) {
	if delay < 0 {
		return
	}
	// 通过反射判断job类型继承自哪一个结构体，并动态赋值上下文等值
	_, ok := tw.taskMap[key]
	if ok {
		tw.addTaskChannel <- &Task{
			delay:         delay,
			key:           key,
			data:          data,
			cycleInterval: cycleInterval,
			job:           job,
			ctx:           tw.ctx,
			offset:        tw.taskMap[key],
		}

	} else {
		tw.taskCounter += 1
		tw.taskMap[key] = tw.taskCounter
		tw.appendTaskSlots()

		tw.addTaskChannel <- &Task{
			delay:         delay,
			key:           key,
			data:          data,
			cycleInterval: cycleInterval,
			job:           job,
			ctx:           tw.ctx,
			offset:        tw.taskCounter,
		}
	}
}

// RemoveTimer 删除定时器 key为添加定时器时传递的定时器唯一标识
func (tw *TimeWheel) RemoveTimer(key string) {
	if key == "" {
		return
	}
	tw.removeTaskChannel <- key
}

func (tw *TimeWheel) start() {
	for {
		select {
		case <-tw.ticker.C:
			tw.time += 1
			//fmt.Printf("%s wheel %d\n", tw.interval, tw.time)
			tw.runing = true
			tw.tickHandler()
			tw.runing = false
		case task := <-tw.addTaskChannel:
			tw.addTask(task)
		case taskStatus := <-tw.taskStatusChannel:
			tw.taskCallBack(taskStatus)
		case key := <-tw.removeTaskChannel:
			tw.removeTask(key)
		case <-tw.stopChannel:
			tw.ticker.Stop()
			return
		}

	}
}

func (tw *TimeWheel) tickHandler() {
	fmt.Println(tw.currentPos)
	allTaskList := tw.slots[tw.currentPos]
	for _, l := range allTaskList {
		tw.scanAndRunTask(l)
	}
	if tw.currentPos == tw.slotNum-1 {
		tw.currentPos = 0
	} else {
		tw.currentPos++
	}

}

// 扫描链表中过期定时器, 并执行回调函数
func (tw *TimeWheel) scanAndRunTask(l *list.List) {
	for e := l.Front(); e != nil; {
		task := e.Value.(*Task)
		if task.circle > 0 {
			task.circle--
			e = e.Next()
			continue
		}
		go func(interface{}) {
			taskStatus := task.job.TaskExec(task.data)
			if taskStatus == nil {
				taskStatus = &TaskStatus{
					Data: task.data,
				}
			}
			taskStatus.Key = task.key
			tw.taskStatusChannel <- taskStatus
		}(task.data)

		next := e.Next()
		l.Remove(e)
		if task.key != "" {
			delete(tw.timer, task.key)
		}
		if task.cycleInterval > 0 {
			task.delay = task.cycleInterval
			tw.addTask(task)
		}

		e = next
	}
}

// 新增任务到链表中
func (tw *TimeWheel) addTask(task *Task) {
	pos, circle := tw.getPositionAndCircle(task.delay)
	task.circle = circle

	tw.slots[pos][tw.taskMap[task.key]].PushBack(task)

	if task.key != "" {
		tw.timer[task.key] = pos
	}
}

// 获取定时器在槽中的位置, 时间轮需要转动的圈数
func (tw *TimeWheel) getPositionAndCircle(d time.Duration) (pos int, circle int) {
	delaySeconds := int(d.Seconds())
	if delaySeconds == 0 && !tw.runing {
		pos = tw.currentPos
		circle = 0

		return
	} else if delaySeconds == 0 && tw.runing {
		pos = tw.currentPos
		circle = 0

		return
	}
	intervalSeconds := int(tw.interval.Seconds())
	circle = delaySeconds / intervalSeconds / tw.slotNum
	pos = (tw.currentPos + delaySeconds/intervalSeconds) % tw.slotNum

	return
}

// 从链表中删除任务
func (tw *TimeWheel) removeTask(key string) {
	// 获取定时器所在的槽
	position, ok := tw.timer[key]
	if !ok {
		return
	}
	// 获取槽指向的链表
	l := tw.slots[position][tw.taskMap[key]]
	for e := l.Front(); e != nil; {
		task := e.Value.(*Task)
		if task.key == key {
			delete(tw.timer, task.key)
			l.Remove(e)
		}

		e = e.Next()
	}
}
