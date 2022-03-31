package main

import (
	"flag"
	"net/http"
	"os"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

func main() {
	var minRepoCount int
	var maxRepoCount int
	var maxNetConns int
	var workerCount int

	flag.IntVar(&minRepoCount, "min-repo-count", 5, "minimum number of reopsitories to query")
	flag.IntVar(&maxRepoCount, "max-repo-count", 25, "maximum number of repositories to query")
	flag.IntVar(&maxNetConns, "max-net-conns", 10, "max number of concurrent network requests")
	flag.IntVar(&workerCount, "worker-count", 3, "number of goroutines, processing query resulsts")

	// Setup logging.
	l := &log.Logger{
		Out:   os.Stdout,
		Level: log.DebugLevel,
		Formatter: &log.TextFormatter{
			TimestampFormat: "2006-01-02T15:04:05.999999999Z07:00",
			FullTimestamp:   true,
		},
	}

	// Setup service.
	f := NewGitlabGraphQLClient(l, http.DefaultClient)
	s := NewService(l, f, ",", maxNetConns, workerCount)

	// Send concurrent reqeusts to service.
	wg := sync.WaitGroup{}
	for i := minRepoCount; i <= maxRepoCount; i++ {
		repoCount := i
		go func() {
			wg.Add(1)
			defer wg.Done()

			l.Infof("Requesting summary of %d repos\n", repoCount)
			summary, err := s.CreateRepositorySummary(repoCount)
			if err != nil {
				log.Errorln(err)
			}
			l.Infof("Received summary of %d repos: %v\n", repoCount, summary)
		}()
	}

	wg.Wait()

	// Wait for goroutines to send requests.
	time.Sleep(5 * time.Millisecond)
	// Check, that deadlock doesn't happen.
	s.Stop()
}
