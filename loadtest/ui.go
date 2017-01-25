// Copyright (c) 2016 Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package loadtest

import (
	"strconv"
	"strings"
	"time"

	"github.com/gizak/termui"
	"github.com/mattermost/mattermost-load-test/cmdlog"
)

func CreateLoadtestUI(stats *ClientTimingStats, logbuf *UIBuffer) {
	if err := termui.Init(); err != nil {
		cmdlog.Error("Unable to init termui: " + err.Error())
	}
	defer termui.Close()

	stopChan := make(chan bool)
	defer close(stopChan)

	timingBuffers := make(map[string]*UIBuffer)
	go sampleClientTimingStats(stopChan, time.Second, stats, timingBuffers)

	logText := termui.NewPar("")
	logText.Height = 12
	logText.BorderLabel = "Logs"
	logText.BorderFg = termui.ColorGreen
	logText.Border = true
	logText.Width = 100

	/*termui.Body.AddRows(
		termui.NewRow(
			termui.NewCol(0, 0, testText),
		),
	)

	termui.Body.Align()
	termui.Render(termui.Body)
	*/

	termui.Handle("/timer/1s", func(e termui.Event) {
		logText.Text = strings.Join(logbuf.GetBufString(), "\n")

		routesparks := make([]termui.Sparkline, 0, len(stats.RouteNames))
		for _, route := range stats.RouteNames {
			sl := termui.NewSparkline()
			if buf, ok := timingBuffers[route]; ok {
				sl.Data = buf.GetBufInt()
				sl.Title = route + " " + strconv.Itoa(sl.Data[len(sl.Data)-1]) + "ms"
			} else {
				sl.Title = route + " wait..."
			}
			sl.LineColor = termui.ColorGreen
			sl.Height = 8
			routesparks = append(routesparks, sl)
		}

		routesparksui := termui.NewSparklines(routesparks...)
		routesparksui.Height = 10*len(routesparks) + 4
		routesparksui.Width = 100
		routesparksui.Y = 12
		termui.Render(logText, routesparksui)
	})

	termui.Handle("/sys/kbd/q", func(termui.Event) {
		termui.StopLoop()
	})

	termui.Handle("/sys/kbd/C-c", func(termui.Event) {
		termui.StopLoop()
	})

	termui.Loop()

	// We are exiting now
	// Switch back to console logging
	cmdlog.SetConsoleLog()
}

func sampleClientTimingStats(stopChan chan bool, sampleRate time.Duration, stats *ClientTimingStats, buffers map[string]*UIBuffer) {
	timer := time.NewTicker(sampleRate)
	for {
		select {
		case <-timer.C:
			for _, path := range stats.RouteNames {
				if uiBuffer, ok := buffers[path]; ok {
					uiBuffer.Add(int(stats.Routes[path].DurationLastMinute.Rate()))
				} else {
					newBuffer := NewUIBuffer(20)
					newBuffer.Add(int(stats.Routes[path].DurationLastMinute.Rate()))
					buffers[path] = newBuffer
				}
			}
		case <-stopChan:
			return
		}
	}
}
