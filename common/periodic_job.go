package common

import "time"

type PeriodicJob struct {
	duration       time.Duration
	ticker         *time.Ticker
	executionCount int64
	running        bool
	job            func()
}

func NewPeriodicJob(job func(), duration time.Duration) *PeriodicJob {
	return &PeriodicJob{
		duration:       duration,
		ticker:         time.NewTicker(duration),
		executionCount: 0,
		running:        false,
		job:            job,
	}
}

func (p *PeriodicJob) SetDuration(duration time.Duration) {
	p.duration = duration
	wasRunning := p.running
	p.Stop()
	p.ticker = time.NewTicker(duration)
	if wasRunning {
		p.Start()
	}
}

func (p *PeriodicJob) GetExecutionCount() int64 {
	return p.executionCount
}

func (p *PeriodicJob) ResetExecutionCount() {
	p.executionCount = 0
}

func (p *PeriodicJob) Start() {
	if p.running {
		return
	}

	p.running = true
	go func() {
		for range p.ticker.C {
			p.job()
			p.executionCount++
		}
	}()
}

func (p *PeriodicJob) StartImmediately() {
	if p.running {
		return
	}

	p.job()
	p.executionCount++
	p.Start()
}

func (p *PeriodicJob) Stop() {
	if !p.running {
		return
	}

	p.running = false
	p.ticker.Stop()
}

func (p *PeriodicJob) GetDuration() time.Duration {
	return p.duration
}
