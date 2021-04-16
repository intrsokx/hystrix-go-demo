package main

import (
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
	"os"
	"sync"
)

func init() {
	log.SetFormatter(&log.TextFormatter{
		TimestampFormat: "2006-01-02 15:04:05",
		FullTimestamp:   true,
	})
	log.SetReportCaller(true)
	log.SetLevel(log.DebugLevel)
	log.SetOutput(os.Stdout)
	//log.SetBufferPool()
}

func main() {
	wg := sync.WaitGroup{}

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			resp, err := http.Get("http://localhost:8080/test")
			if err != nil {
				log.Error("http get err:", err)
			}
			html, err := ioutil.ReadAll(resp.Body)
			log.Info("html:", string(html))
		}()

	}
	wg.Wait()
}
