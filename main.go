package main

import (
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sparrc/go-ping"
	"log"
	"net/http"
)

import (
	_ "net/http/pprof"
	"time"
)

var (
	// record every ping RTT latency
	pingRTTHistogram = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name: "ping_rtt_histogram_seconds",
		Help: "ping rtt latency distribution.",
		// not exactly a doubling, but covers a reasonable spread latencies
		Buckets: []float64{0.001, 0.0025, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1},
	},
		[]string{"host"},
	)
	pingSent = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "ping_packets_sent",
		Help: "packets sent by pinger",
	},
		[]string{"host"},
	)
	pingReceived = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "ping_packets_srec",
		Help: "packets received by pinger",
	},
		[]string{"host"},
	)
)

func endless_pings(h string) {
	defer fmt.Println("ended ping")

	pinger, err := ping.NewPinger(h)
	if err != nil {
		fmt.Printf("ERROR: %s\n", err.Error())
		return
	}

	pinger.OnRecv = func(pkt *ping.Packet) {
		// fmt.Printf("%d bytes from %s: icmp_seq=%d time=%v\n",
		//	pkt.Nbytes, pkt.IPAddr, pkt.Seq, pkt.Rtt)
		// Observe the rtt as seconds into the histogram
		pingRTTHistogram.WithLabelValues(pkt.IPAddr.String()).Observe(pkt.Rtt.Seconds())
		pingSent.WithLabelValues(pkt.IPAddr.String()).Set(float64(pinger.PacketsSent))
		pingReceived.WithLabelValues(pkt.IPAddr.String()).Set(float64(pinger.PacketsRecv))
		//
		time.Sleep(1 * time.Millisecond)
	}

	pinger.Run()
	// wait forever
	select {}
}

func init() {
	// Register the summary and the histogram with Prometheus's default registry.
	prometheus.MustRegister(pingRTTHistogram)
	prometheus.MustRegister(pingReceived)
	prometheus.MustRegister(pingSent)
}

func main() {
	// Handle SIGINT and SIGTERM.
	//ch := make(chan os.Signal)
	//signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	//fmt.Println("received", <-ch)

	defer fmt.Println("ended main")
	go endless_pings("8.8.8.8")
	// go endless_pings("1.1.1.1")

	// http.HandleFunc("/", root_handler)
	http.Handle("/metrics", promhttp.Handler())
	log.Fatal(http.ListenAndServe(":8080", nil))

	select {}
}
