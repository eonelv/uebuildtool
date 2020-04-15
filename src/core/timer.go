package core

import "time"

type Timer struct {
	duration   int64
	updateTime int64
	ch         chan bool
}

func NewTimer() *Timer {
	return &Timer{}
}

func (this *Timer) GetChannel() chan bool {
	return this.ch
}

func (this *Timer) isActive() bool {
	return this.duration != 0
}

func (this *Timer) Stop() {
	this.duration = 0
	this.updateTime = 0
	if this.ch == nil {
		return
	}
	close(this.ch)
	this.ch = nil
}

func (this *Timer) Start(duration int64) {
	this.duration = duration
	this.updateTime = GetMilliSeconds()
	ch := make(chan bool, 1)
	this.ch = ch
	go func(timer *Timer) {
		for {
			if !timer.isActive() {
				LogInfo("timer is error ", duration)
				break
			}
			time.Sleep(100 * time.Millisecond)
			now := GetMilliSeconds()
			if now-this.updateTime >= duration {
				timer.updateTime = now
				ch <- true
			}
		}
	}(this)
}

func GetMilliSeconds() int64 {
	return time.Now().UnixNano() / 1e6
}
