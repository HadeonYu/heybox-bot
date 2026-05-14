package heybox

import (
	"fmt"
	"heybox-bot/logger"
	"time"
)

var scheduledJobs []*scheduledJob

type TimerCallback func(*TimerContext) error

type TimerContext struct {
	Name string
	task *scheduledJob
}

func (ctx *TimerContext) SetInterval(interval time.Duration) {
	if interval <= 0 {
		return
	}
	ctx.task.interval = interval
}

func (ctx *TimerContext) Interval() time.Duration {
	return ctx.task.interval
}

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

func AddTimer(name string, interval time.Duration, callback TimerCallback) error {
	return addTimer(name, interval, callback, false)
}

func AddImmediateTimer(name string, interval time.Duration, callback TimerCallback) error {
	return addTimer(name, interval, callback, true)
}

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

	runMu.Lock()
	defer runMu.Unlock()

	if stopCh != nil {
		return fmt.Errorf("机器人运行后不能添加定时器")
	}

	for _, job := range scheduledJobs {
		if job.name == name {
			return fmt.Errorf("定时器 %q 已存在", name)
		}
	}

	scheduledJobs = append(scheduledJobs, &scheduledJob{
		name:      name,
		interval:  interval,
		callback:  callback,
		immediate: immediate,
	})

	return nil
}

func cloneScheduledJobs() []*scheduledJob {
	jobs := make([]*scheduledJob, 0, len(scheduledJobs))
	now := time.Now()
	for _, job := range scheduledJobs {
		clone := *job
		if clone.immediate {
			clone.nextRun = now
		} else {
			clone.nextRun = now.Add(clone.interval)
		}
		jobs = append(jobs, &clone)
	}
	return jobs
}

func runLoop(stop <-chan struct{}, done chan<- struct{}, jobs []*scheduledJob) {
	defer close(done)

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-stop:
			logger.Info("机器人已停止")
			return
		case <-ticker.C:
			runDueJobs(jobs)
		}
	}
}

func runDueJobs(jobs []*scheduledJob) {
	now := time.Now()
	for _, job := range jobs {
		if job.stopped || now.Before(job.nextRun) {
			continue
		}

		ctx := &TimerContext{
			Name: job.name,
			task: job,
		}
		if err := job.callback(ctx); err != nil {
			logger.Error("定时器 %q 回调执行失败: %v", job.name, err)
		}

		if !job.stopped {
			job.nextRun = time.Now().Add(job.interval)
		}
	}
}
