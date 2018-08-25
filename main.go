package main

import (
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sparrc/go-ping"
	"log"
	"net/http"
)

var (
	// record every ping RTT latency
	pingRTTHistogram = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name: "ping_rtt_histogram_seconds",
		Help: "ping rtt latency distributions.",
		// not exactly a doubling, but covers a reasonable spread latencies
		Buckets: []float64{0.001, 0.0025, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1},
	},
		[]string{"host"},
	)
)

func endless_pings(h string) {
	pinger, err := ping.NewPinger(h)
	if err != nil {
		fmt.Printf("ERROR: %s\n", err.Error())
		return
	}

	pinger.OnRecv = func(pkt *ping.Packet) {
		fmt.Printf("%d bytes from %s: icmp_seq=%d time=%v\n",
			pkt.Nbytes, pkt.IPAddr, pkt.Seq, pkt.Rtt)
		pingRTTHistogram.WithLabelValues(h).Observe(pkt.Rtt.Seconds())
	}

	pinger.Run()
	// wait forever
	select {}
}

func root_handler(w http.ResponseWriter, r *http.Request) {
	title := "FastRobot golang constant pinger"

	from := ""
	if r.URL != nil {
		from = r.URL.String()
	}
	if from != "/favicon.ico" {
		log.Printf("title: %s\n", title)
	}

	fmt.Fprintf(w, "Hello from:  "+title+"\n")
}

func init() {
	// Register the summary and the histogram with Prometheus's default registry.
	prometheus.MustRegister(pingRTTHistogram)
}

func main() {
	go endless_pings("8.8.8.8")
	http.HandleFunc("/", root_handler)
	http.Handle("/metrics", promhttp.Handler())
	http.ListenAndServe(":8080", nil)
}
