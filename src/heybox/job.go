package heybox

import (
	"fmt"
	"heybox-bot/logger"
	"sync"
	"time"
)

var (
	scheduledJobs   []*scheduledJob
	scheduledJobsMu sync.Mutex
)

type TimerCallback func(*TimerContext) error

type TimerContext struct {
	Name string
	task *scheduledJob
}

// SetInterval 修改当前定时器下一轮使用的执行间隔。
func (ctx *TimerContext) SetInterval(interval time.Duration) {
	if interval <= 0 {
		return
	}
	ctx.task.interval = interval
}

// Interval 返回当前定时器的执行间隔。
func (ctx *TimerContext) Interval() time.Duration {
	return ctx.task.interval
}

// StopTimer 标记当前定时器停止后续执行。
func (ctx *TimerContext) StopTimer() {
	ctx.task.stopped = true
}

type scheduledJob struct {
	name      string
	interval  time.Duration
	nextRun   time.Time
	callback  TimerCallback
	stopped   bool
	immediate bool
}

// AddTimer 添加首次延迟执行的定时器任务。
func AddTimer(name string, interval time.Duration, callback TimerCallback) error {
	return addTimer(name, interval, callback, false)
}

// AddImmediateTimer 添加启动后立即执行一次的定时器任务。
func AddImmediateTimer(name string, interval time.Duration, callback TimerCallback) error {
	return addTimer(name, interval, callback, true)
}

// addTimer 校验参数并注册定时器任务。
func addTimer(name string, interval time.Duration, callback TimerCallback, immediate bool) error {
	if name == "" {
		return fmt.Errorf("定时器名称为空")
	}
	if interval <= 0 {
		return fmt.Errorf("定时器间隔必须大于 0")
	}
	if callback == nil {
		return fmt.Errorf("定时器回调为空")
	}

	now := time.Now()

	scheduledJobsMu.Lock()
	defer scheduledJobsMu.Unlock()
	for _, job := range scheduledJobs {
		if job.name == name {
			return fmt.Errorf("定时器 %q 已存在", name)
		}
	}

	scheduledJobs = append(scheduledJobs, &scheduledJob{
		name:      name,
		interval:  interval,
		nextRun:   nextRunTime(now, interval, immediate),
		callback:  callback,
		immediate: immediate,
	})

	return nil
}

// nextRunTime 根据是否立即执行计算下一次运行时间。
func nextRunTime(now time.Time, interval time.Duration, immediate bool) time.Time {
	if immediate {
		return now
	}
	return now.Add(interval)
}

// resetScheduledJobs 在运行循环启动前重置所有定时器的下一次运行时间。
func resetScheduledJobs() {
	scheduledJobsMu.Lock()
	defer scheduledJobsMu.Unlock()

	now := time.Now()
	for _, job := range scheduledJobs {
		if job.stopped {
			continue
		}
		job.nextRun = nextRunTime(now, job.interval, job.immediate)
	}
}

// runLoop 持续驱动定时器任务直到收到停止信号。
func runLoop(stop <-chan struct{}, done chan<- struct{}) {
	defer close(done)

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-stop:
			logger.Info("机器人已停止")
			return
		case <-ticker.C:
			runDueJobs()
		}
	}
}

// runDueJobs 执行所有已经到期的定时器任务。
func runDueJobs() {
	now := time.Now()
	jobs := dueJobs(now)
	for _, job := range jobs {
		ctx := &TimerContext{
			Name: job.name,
			task: job,
		}
		if err := job.callback(ctx); err != nil {
			logger.Error("定时器 %q 回调执行失败: %v", job.name, err)
		}

		scheduledJobsMu.Lock()
		if !job.stopped {
			job.nextRun = time.Now().Add(job.interval)
		}
		scheduledJobsMu.Unlock()
	}
}

// dueJobs 返回指定时间点已经到期且未停止的任务列表。
func dueJobs(now time.Time) []*scheduledJob {
	scheduledJobsMu.Lock()
	defer scheduledJobsMu.Unlock()

	jobs := make([]*scheduledJob, 0, len(scheduledJobs))
	for _, job := range scheduledJobs {
		if job.stopped || now.Before(job.nextRun) {
			continue
		}
		jobs = append(jobs, job)
	}
	return jobs
}
