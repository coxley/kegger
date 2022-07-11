package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/golang/glog"
	"github.com/gorilla/websocket"
	"periph.io/x/conn/v3/gpio"
	"periph.io/x/conn/v3/gpio/gpioreg"
	"periph.io/x/host/v3"
)

var (
	STATS        *Stats = nil
	frontend            = flag.String("frontend", "./www/build/", "directory to serve www from")
	addr                = flag.String("addr", ":80", "socket to listen for www connections on (including websocket)")
	pulsesPerGal        = flag.Int("ppg", 4994, "how many pulses of your flow meter == 1 gallon?")
	taps                = map[int]string{}
)

func main() {
	flag.Func("tap", "tap number to the associated GPIO, eg: 1:GPIO2. (can be repeated)", func(s string) error {
		parts := strings.SplitN(s, ":", 2)
		if len(parts) != 2 {
			return fmt.Errorf("unable to parse %q as int:gpioname", s)
		}

		id, err := strconv.Atoi(parts[0])
		if err != nil {
			return fmt.Errorf("unable to parse %q as int:gpioname: %v", s, err)
		}
		taps[id] = parts[1]
		return nil
	})
	flag.Set("logtostderr", "true")
	flag.Parse()

	if len(taps) == 0 {
		glog.Fatal("must configure one or more taps with the '-tap' flag")
	}

	stats, err := loadStats()
	if err != nil {
		glog.Fatal(err)
	}
	STATS = stats

	if _, err := host.Init(); err != nil {
		glog.Fatalf("failed loading the host drivers: %v", err)
	}

	glog.Info("Welcome to kegs by coxley")
	glog.Info("-------------------------")
	glog.Info("")
	glog.Info("Configured taps:")
	for tap, pinDesc := range taps {
		glog.Infof("%d -> %s", tap, pinDesc)
	}
	glog.Info("")

	for tap, pinDesc := range taps {
		p := gpioreg.ByName(pinDesc)
		if p == nil {
			glog.Fatalf("could not find %q by name", pinDesc)
		}

		if err := p.In(gpio.PullUp, gpio.FallingEdge); err != nil {
			glog.Fatal(err)
		}
		go measure(tap, p)
	}

	glog.Info("")
	go startWWW()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	done := make(chan bool, 1)
	go func() {
		sig := <-sigs
		glog.Warningf("shutting down from signal: %v", sig)
		done <- true
	}()

	<-done
}

var statsmu = sync.Mutex{}

type Stats struct {
	TotalPulses int              `json:"total_pulses"`
	TotalOunces float64          `json:"total_ounces"`
	Records     map[int][]Record `json:"records"`
	watchers    map[int]func([]byte)
}

type Record struct {
	Timestamp int64   `json:"timestamp"`
	Tap       int     `json:"tap"`
	Pulses    int     `json:"pulses"`
	Ounces    float64 `json:"ounces"`
}

func loadStats() (*Stats, error) {
	buf, err := ioutil.ReadFile("stats")
	if err != nil {
		return nil, err
	}

	if len(buf) == 0 {
		// TODO: Handle dynamic configured taps
		s := &Stats{Records: map[int][]Record{}, watchers: map[int]func([]byte){}}
		return s, nil
	}

	s := &Stats{}
	err = json.Unmarshal(buf, s)
	if err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Stats) Save() error {
	statsmu.Lock()
	defer statsmu.Unlock()
	buf, err := s.JSON()
	if err != nil {
		return err
	}

	err = os.WriteFile("stats", buf, 0600)
	if err != nil {
		return err
	}

	for _, cb := range s.watchers {
		go cb(buf)
	}

	return nil
}

func (s *Stats) JSON() ([]byte, error) {
	return json.Marshal(s)
}

func (s *Stats) AddRecord(tap int, pulses int, ounces float64) {
	s.Records[tap] = append(s.Records[tap], Record{
		Timestamp: time.Now().Unix(),
		Tap:       tap,
		Pulses:    pulses,
		Ounces:    ounces,
	})
	s.TotalPulses += pulses
	s.TotalOunces += ounces
	err := s.Save()
	if err != nil {
		glog.Errorf("flushing stats to disk failed: %v", err)
	}
}

type PourState int

const (
	PourIdle PourState = iota
	PourActive
)

type Pour struct {
	Timestamp int64   `json:"timestamp"`
	Tap       int     `json:"tap"`
	Active    bool    `json:"active"`
	Pulses    int     `json:"pulses"`
	Ounces    float64 `json:"ounces"`
}

var pourWatchers = map[int]func([]byte){}

func pourSubscribe(cb func([]byte)) int {
	if pourWatchers == nil {
		pourWatchers = map[int]func([]byte){}
	}
	id := len(pourWatchers) + 1
	pourWatchers[id] = cb
	return id
}

func pourUnsubscribe(id int) {
	delete(pourWatchers, id)
}

func pourUpdate(tap int, pulses int, active bool) {
	p := &Pour{
		Timestamp: time.Now().Unix(),
		Tap:       tap,
		Active:    active,
		Pulses:    pulses,
		Ounces:    pulsesToOz(pulses),
	}
	b, err := json.Marshal(p)
	if err != nil {
		glog.Error(err)
		return
	}
	for _, cb := range pourWatchers {
		go cb(b)
	}
}

func measure(tap int, p gpio.PinIO) {
	glog.Infof("Tap %d configured, running, and reading from %q", tap, p)
	var pulses int
	var state = PourIdle
	for {
		edge := p.WaitForEdge(time.Second * 4)
		gotEdge, timedOut := edge, !edge
		switch {
		case timedOut && state == PourIdle:
			continue
		case timedOut && state == PourActive:
			// End pour
			state = PourIdle
			if pulses < 10 {
				pulses = 0
				continue
			}
			ounces := pulsesToOz(pulses)
			STATS.AddRecord(tap, pulses, ounces)
			pourUpdate(tap, pulses, false)
			glog.Infof("[Tap %d] %.2f oz pour finished (pulses=%d)", tap, ounces, pulses)
			pulses = 0
		case gotEdge && state == PourIdle:
			glog.Infof("[Tap %d] Pour started", tap)
			state = PourActive
			pulses++
		case gotEdge && state == PourActive:
			pulses++
			// Delay notifying the web UI until a pour is likely vs. CO2 discharge
			if pulses == 10 {
				pourUpdate(tap, pulses, true)
			}
			if pulses%5 == 0 {
				pourUpdate(tap, pulses, true)
			}
		}
	}
}

func pulsesToOz(p int) float64 {
	ozPerGal := 128
	return float64(ozPerGal) * (float64(p) / float64(*pulsesPerGal))
}

// Subscribe callback to run every time Stats are updated
//
// Returns an ID to defer Unsubscribe with
func (s *Stats) Subscribe(cb func([]byte)) int {
	if s.watchers == nil {
		s.watchers = map[int]func([]byte){}
	}
	id := len(s.watchers) + 1
	s.watchers[id] = cb
	return id
}

func (s *Stats) Unsubscribe(id int) {
	if s.watchers == nil {
		s.watchers = map[int]func([]byte){}
	}
	delete(s.watchers, id)
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func statsEndpoint(w http.ResponseWriter, r *http.Request) {
	upgrader.CheckOrigin = func(r *http.Request) bool { return true }

	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
	}

	glog.Infof("Client Connected: addr=%s, agent=%s", r.RemoteAddr, r.UserAgent())
	if err != nil {
		log.Println(err)
	}

	// Send the stats we have when a client connects
	b, _ := STATS.JSON()
	if err := ws.WriteMessage(websocket.TextMessage, b); err != nil {
		glog.Warning(err)
		return
	}

	// Subscribe for future stats changes
	mu := sync.Mutex{}
	id := STATS.Subscribe(func(b []byte) {
		mu.Lock()
		defer mu.Unlock()
		if err := ws.WriteMessage(websocket.TextMessage, b); err != nil {
			glog.Warning(err)
			return
		}
	})
	defer STATS.Unsubscribe(id)
	for {
		time.Sleep(time.Second * 2)
	}
}

func pourEndpoint(w http.ResponseWriter, r *http.Request) {
	upgrader.CheckOrigin = func(r *http.Request) bool { return true }

	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
	}

	glog.Infof("Client Connected: addr=%s, agent=%s", r.RemoteAddr, r.UserAgent())
	if err != nil {
		log.Println(err)
	}

	mu := sync.Mutex{}
	id := pourSubscribe(func(b []byte) {
		mu.Lock()
		defer mu.Unlock()
		if err := ws.WriteMessage(websocket.TextMessage, b); err != nil {
			glog.Warning(err)
			return
		}
	})
	defer pourUnsubscribe(id)
	for {
		time.Sleep(time.Second * 2)
	}
}

func startWWW() {
	http.Handle("/", http.FileServer(http.Dir(*frontend)))
	http.HandleFunc("/stats", statsEndpoint)
	http.HandleFunc("/pours", pourEndpoint)
	glog.Infof("Starting web server")
	glog.Error(http.ListenAndServe(*addr, nil))
}
