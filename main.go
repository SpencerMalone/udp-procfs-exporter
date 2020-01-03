package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	targetProcName  string
	targetPID       string
	udpBufferQueued = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "statsd_exporter_udp_buffer_queued",
			Help: "The number of queued UDP messages in the linux buffer.",
		},
		[]string{"protocol"},
	)
	udpBufferDropped = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "statsd_exporter_udp_buffer_dropped",
			Help: "The number of dropped UDP messages in the linux buffer",
		},
		[]string{"protocol"},
	)
)

func init() {
	prometheus.MustRegister(udpBufferQueued)
	prometheus.MustRegister(udpBufferDropped)
}

func main() {
	if runtime.GOOS != "linux" {
		log.Fatalln("ProcFS is only supported on linux!")
	}
	args := os.Args
	if len(args) != 2 {
		log.Fatalln("Usage: udp-procfs-exporter <processname>")
	}

	findPIDByName(args[1])
	if targetPID == "" {
		log.Fatalln("Unable to find proc with the name: " + targetProcName)
	}
	fmt.Println("UDP Procfs Exporter started, watching PID " + targetPID)
	go serveHTTP(":8125", "/metrics")
	watchUDPBuffers(0, 0)
}

func serveHTTP(listenAddress, metricsEndpoint string) {
	//lint:ignore SA1019 prometheus.Handler() is deprecated.
	http.Handle(metricsEndpoint, promhttp.Handler())
	log.Fatal(http.ListenAndServe(listenAddress, nil))
}

func findPIDByName(procName string) {
	targetProcName = procName
	err := filepath.Walk("/proc", walkProcFSStatus)
	if err != nil {
		if err == io.EOF {
			// Not an error, just a signal when we are done
			err = nil
		} else {
			log.Fatal(err)
		}
	}
}

func walkProcFSStatus(path string, info os.FileInfo, err error) error {
	if err != nil {
		// All of these are garbage that I can find, we just need to skip any known error'd files
		return nil
	}

	if strings.Contains(path, "/status") && strings.Contains(path, "/proc/") && strings.Count(path, "/") == 3 {
		pid, err := strconv.Atoi(path[6:strings.LastIndex(path, "/")])
		if err != nil {
			return err
		}

		f, err := ioutil.ReadFile(path)
		if err != nil {
			return err
		}

		// First line of procfs status files looks like..
		// Name:	<proc name>
		name := string(f[6:bytes.IndexByte(f, '\n')])

		if name == targetProcName {
			targetPID = strconv.Itoa(pid)
		}
	}

	return nil
}

func watchUDPBuffers(lastDropped int, lastDropped6 int) {
	for {
		queuedUDP, droppedUDP := parseProcfsNetFile("/proc/" + targetPID + "/net/udp")
		label := "udp"

		udpBufferQueued.WithLabelValues(label).Set(float64(queuedUDP))

		diff := droppedUDP - lastDropped
		if diff < 0 {
			fmt.Println("Dropped count went negative! Abandoning UDP buffer parsing")
			diff = 0
			droppedUDP = lastDropped
		}
		udpBufferDropped.WithLabelValues(label).Add(float64(diff))

		queuedUDP6, droppedUDP6 := parseProcfsNetFile("/proc/" + targetPID + "/net/udp6")
		label = "udp6"

		udpBufferQueued.WithLabelValues(label).Set(float64(queuedUDP6))

		diff = droppedUDP6 - lastDropped6
		if diff < 0 {
			fmt.Println("Dropped count went negative! Abandoning UDP buffer parsing")
			diff = 0
			droppedUDP6 = lastDropped6
		}
		udpBufferDropped.WithLabelValues(label).Add(float64(diff))

		time.Sleep(10 * time.Second)
		lastDropped = droppedUDP
		lastDropped6 = droppedUDP6
	}
}

func parseProcfsNetFile(filename string) (int, int) {
	f, err := os.Open(filename)
	if err != nil {
		return 0, 0
	}
	defer f.Close()

	queued := 0
	dropped := 0
	s := bufio.NewScanner(f)
	for n := 0; s.Scan(); n++ {
		// Skip the header lines.
		if n < 1 {
			continue
		}

		fields := strings.Fields(s.Text())

		queuedLine, err := strconv.ParseInt(strings.Split(fields[4], ":")[1], 16, 32)
		queued = queued + int(queuedLine)
		if err != nil {
			fmt.Println("Unable to parse queued UDP buffers:", err)
			return 0, 0
		}

		droppedLine, err := strconv.Atoi(fields[12])
		dropped = dropped + droppedLine
		if err != nil {
			fmt.Println("Unable to parse dropped UDP buffers:", err)
			return 0, 0
		}
	}

	return queued, dropped
}
