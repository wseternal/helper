package stats

import (
	"fmt"
	"time"
)

type BuiltinTerm uint64

const (
	TermCustom             = 0
	TermSecond BuiltinTerm = 1 << iota
	TermMinue
	TermHour
	TermDay
	TermWeek
)

const (
	// checkThreshHold in unit of second, if duration of Term beyond the threshold,
	// use timer to trigger the reset, otherwise, checking the elapsed each time when
	// the counter is changed
	checkThreshHold  = time.Duration(10)
	channelBufferNum = 10
)

// Statistic is collected according to corresponding period.
// A Counter object may have multiple Statistic objects.
// when duration less than checkThreshHold, it is checked
// every time when a counter message arrives.
type Statistic struct {
	// total: how many times the corresponding counter event occurred
	// during the specific period.
	// Min: the minimal total in history;
	// Max: the maximal total in history.
	Total, Min, Max int64

	// Statistic may have extra value attached, e.g.: when
	// evaluate the network delay value of each requests
	ValueTotal, ValueAvg, ValueAvgMin, ValueAvgMax int64

	// Duration Statistic period, in unit of second
	Duration time.Duration
	TSStart  int64
}

// countMessage message to modify the counter
type countMessage struct {
	Count int64
	Value int64
	TS    int64
}

type Counter struct {
	Name  string
	Terms map[time.Duration]*Statistic

	StartTime int64
	msg       chan *countMessage
	reset     chan *Statistic
}

func (c *Statistic) Add(total, value int64) {
	c.Total += total
	c.ValueTotal += value
}

func (c *Statistic) Reset() {
	if c.Total == 0 {
		return
	}
	if c.Total > c.Max {
		c.Max = c.Total
	}
	if c.Min == 0 || c.Min > c.Total {
		c.Min = c.Total
	}

	c.ValueAvg = c.ValueTotal / c.Total
	if c.ValueAvg > c.ValueAvgMax {
		c.ValueAvgMax = c.ValueAvg
	}
	if c.ValueAvgMin == 0 || c.ValueAvgMin > c.ValueAvg {
		c.ValueAvgMin = c.ValueAvg
	}
	c.Total = 0
	c.ValueTotal = 0
}

func (counter *Counter) counterMessageHandler() {
	len := len(counter.Terms)
	counter.msg = make(chan *countMessage, channelBufferNum*len)
	counter.reset = make(chan *Statistic, len)

	counter.StartTime = time.Now().Unix()
	for _, c := range counter.Terms {
		if c.Duration > checkThreshHold {
			count := c
			time.AfterFunc(count.Duration*time.Second, func() {
				counter.reset <- count
			})
		}
		c.TSStart = counter.StartTime
	}
	var elapsed time.Duration
	for {
		select {
		case msg := <-counter.msg:
			for k, c := range counter.Terms {
				if k <= checkThreshHold {
					elapsed = time.Duration(msg.TS - c.TSStart)
					if elapsed > c.Duration {
						c.Reset()
						c.TSStart = msg.TS
					} else {
						c.Total += msg.Count
						c.ValueTotal += msg.Value
					}
				} else {
					c.Total += msg.Count
					c.ValueTotal += msg.Value
				}
			}
		case c := <-counter.reset:
			c.Reset()
			time.AfterFunc(c.Duration*time.Second, func() {
				counter.reset <- c
			})
		}
	}
}

func (counter *Counter) Start() {
	go counter.counterMessageHandler()
}

func (counter *Counter) Add(count, value int64) {
	now := time.Now().Unix()
	counter.msg <- &countMessage{
		Count: count,
		TS:    now,
		Value: value,
	}
}

func (counter *Counter) Inc() {
	counter.Add(1, 0)
}

func (counter *Counter) AddTerm(dur time.Duration) (err error) {
	if dur < time.Second {
		return fmt.Errorf("do not support dur <%s> that is less than 1 second", dur.String())
	}
	if counter.StartTime != 0 {
		return fmt.Errorf("counter is already started at %s, can not add new term", time.Unix(counter.StartTime, 0).String())
	}
	key := dur / time.Second
	if _, found := counter.Terms[key]; found {
		return nil
	}
	counter.Terms[key] = &Statistic{
		Duration: key,
	}
	return nil
}

// NewCounter create a counter object with given name and period
// the counter's statistics are collected using terms, e.g.: you can
// get stats of the last second, minute, our, day, etc.
func NewCounter(name string, terms BuiltinTerm) (counter *Counter) {
	counter = &Counter{
		Name:  name,
		Terms: make(map[time.Duration]*Statistic, 1),
	}
	var key time.Duration
	if (terms & TermSecond) > 0 {
		key = time.Second / time.Second
		counter.Terms[key] = &Statistic{
			Duration: key,
		}
	}
	if (terms & TermMinue) > 0 {
		key = time.Minute / time.Second
		counter.Terms[key] = &Statistic{
			Duration: key,
		}
	}
	if (terms & TermHour) > 0 {
		key = time.Hour / time.Second
		counter.Terms[key] = &Statistic{
			Duration: key,
		}
	}
	if (terms & TermDay) > 0 {
		key = time.Hour * 24 / time.Second
		counter.Terms[key] = &Statistic{
			Duration: key,
		}
	}
	if (terms & TermWeek) > 0 {
		key = time.Hour * 24 * 7 / time.Second
		counter.Terms[key] = &Statistic{
			Duration: key,
		}
	}
	return counter
}
