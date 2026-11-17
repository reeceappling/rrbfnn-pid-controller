package timeutils

import (
	"time"
	"context"
)

var globalGetter = &DefaultTimeGetter{lastReadTimeField{lastReadTime: time.Now()}}

const timeGetterKey ctxString = "timeGetter"

type ctxString string

type TimeGetter interface {
	LastReadTime() time.Time
	Now() time.Time
}
type lastReadTimeField struct {
	lastReadTime time.Time
}

func (lrt lastReadTimeField) LastReadTime() time.Time {
	return lrt.lastReadTime
}

type DefaultTimeGetter struct {
	lastReadTimeField
}

func (tg *DefaultTimeGetter) Now() time.Time {
	return time.Now()
}

type CustomTimeGetter struct {
	currentTime time.Time
	lastReadTimeField
}

func (tg *CustomTimeGetter) Now() time.Time {
	return tg.currentTime
}
func (tg *CustomTimeGetter) SetTime(t time.Time) {
	tg.lastReadTime = tg.currentTime
	tg.currentTime = t
}
func (tg *CustomTimeGetter) MoveTimeForward(dt time.Duration) {
	tg.SetTime(tg.currentTime.Add(dt))
}

type PresetTimeGetter struct {
	lastReadTimeField
	next *timeNode
}

func NewPresetTimeGetter(lastReadTime *time.Time, preloadedFirstFutureTime time.Time, preloadedFutureTimes ...time.Time) *PresetTimeGetter {
	var lrt time.Time
	if lastReadTime != nil {
		lrt = *lastReadTime
	} else {
		lrt = preloadedFirstFutureTime.Add(-1 * time.Second) // TODO: ok?
	}
	out := &PresetTimeGetter{
		lastReadTimeField: lastReadTimeField{
			lastReadTime: lrt,
		},
		next: &timeNode{
			next: nil,
			time: preloadedFirstFutureTime,
		},
	}
	out.AddTimes(preloadedFutureTimes...)
	return out
}

type timeNode struct {
	next *timeNode
	time time.Time
}

func (tg *PresetTimeGetter) Now() time.Time {
	toUseAndUpdate := tg.next
	tg.lastReadTime, tg.next = toUseAndUpdate.time, toUseAndUpdate.next
	toUseAndUpdate.next = nil
	return tg.LastReadTime()
}
func (tg *PresetTimeGetter) AddTimes(toAdd ...time.Time) {
	var prev = tg.next
	for _, t := range toAdd {
		newNode := &timeNode{
			time: t,
		}
		if prev != nil {
			prev.next = newNode
		}
		prev = newNode
	}
}

type ChannelTimeGetter struct {
	lastReadTimeField
	next <-chan time.Time
}

func NewChannelTimeGetter(lrt time.Time, next chan time.Time) *ChannelTimeGetter {
	return &ChannelTimeGetter{
		lastReadTimeField: lastReadTimeField{lastReadTime: lrt},
		next:              next,
	}
}

func (tg *ChannelTimeGetter) Now() time.Time {
	newTime, ok := <-tg.next
	if !ok {
		panic("failed to pull from channel")
	}
	tg.lastReadTime = newTime
	return tg.LastReadTime()
}

func SetGetterOnContext(ctx context.Context, getter TimeGetter) context.Context {
	return context.WithValue(ctx, timeGetterKey, getter)
}

func getGetter(ctx context.Context) TimeGetter {
	ctxVal := ctx.Value(timeGetterKey)
	if ctxVal == nil {
		return globalGetter
	}
	tg, ok := ctxVal.(TimeGetter)
	if !ok {
		panic("invalid type of timeGetter, should never happen")
	}
	return tg
}

func GetTime(ctx context.Context) time.Time {
	return getGetter(ctx).Now()
}
func GetMostRecentTime(ctx context.Context) time.Time {
	return getGetter(ctx).LastReadTime()
}
func TimeSince(ctx context.Context, oldTime time.Time) time.Duration {
	tg := getGetter(ctx)
	return tg.Now().Sub(oldTime)
}
