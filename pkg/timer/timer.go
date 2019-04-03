package timer

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/lentil1016/descheduler/pkg/config"
)

var outOfTime bool
var outOfTimeMutex sync.Mutex
var duration time.Duration
var pushEvent func()

func InitTimer(pushEventHandle func()) error {
	// Set the function which will be called when timer starts.
	pushEvent = pushEventHandle

	// Disabled trigger first.
	// Is ok to Set `outOfTime` without Lock here because worker are not running yet.
	// There is no race condition during this method.
	outOfTime = true
	conf := config.GetConfig()
	if conf.Triggers.Mode == "time" {
		var err error
		duration, err = time.ParseDuration(conf.Triggers.Time.For)
		if err != nil {
			return err
		}
	} else if conf.Triggers.Mode == "event" {
		// In event mode, descheduler is triggered not by timer but by event.
		// This will allow event to be triggered.
		outOfTime = false
	} else {
		// Unexpected value check
		return errors.New("Please check config file. Can't recognize spec.triggers.mode with value " + conf.Triggers.Mode + ", either set it to [event] or [time]")
	}
	return nil
}

func RunTimer() {
	conf := config.GetConfig()
	if conf.Triggers.Mode == "time" {
		hour, min, _ := conf.Triggers.Time.From.Clock()
		go runTimerAt(hour, min)
	}

}

func runTimerAt(hour int, min int) {
	for {
		// If now is the time that user configured in spec.triggers.time.from, start a timer.
		if curHour, curMin, _ := time.Now().Clock(); curHour == hour && curMin == min {
			timer := time.NewTimer(duration)
			fmt.Printf("Timer started at %v:%v, last for %v\n", hour, min, duration.String())
			{
				outOfTimeMutex.Lock()
				outOfTime = false
				outOfTimeMutex.Unlock()
			}
			pushEvent()
			// wait util timer stopped
			<-timer.C
			{
				outOfTimeMutex.Lock()
				outOfTime = true
				outOfTimeMutex.Unlock()
			}
			fmt.Println("Timer stopped")
		} else {
			time.Sleep(20 * time.Second)
		}
	}
}

func IsOutOfTime() bool {
	return outOfTime
}
