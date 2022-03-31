package main

import (
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"testing"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

// RepositroyDataFetcherMock emulates Gitlab API with random network delay.
type RepositroyDataFetcherMock struct {
	delayMax int
}

func (mock *RepositroyDataFetcherMock) FetchRepositoryData(repoCount int) ([]RepositoryData, error) {
	// Generate network dealy.
	delay := rand.Intn(mock.delayMax + 1)
	time.Sleep(time.Duration(delay) * time.Millisecond)

	// Generate dummy data.
	data := make([]RepositoryData, repoCount)
	for i := 0; i < repoCount; i++ {
		data[i] = RepositoryData{
			Name:       fmt.Sprint(i),
			ForksCount: i,
		}
	}
	return data, nil
}

// expectedRepositorySummary returns expected results of proceccsing of RepositoryData.
func (mock *RepositroyDataFetcherMock) expectedRepositorySummary(repoCount int, repoNameSep string) RepositorySummary {
	data, _ := mock.FetchRepositoryData(repoCount)

	repoNames := make([]string, repoCount)
	totalForks := 0
	for i, data := range data {
		repoNames[i] = data.Name
		totalForks += data.ForksCount
	}

	return RepositorySummary{
		JoinedNames: strings.Join(repoNames, repoNameSep),
		TotalForks:  totalForks,
	}
}

var dataFetcherMock = RepositroyDataFetcherMock{
	delayMax: 200,
}

func TestService(t *testing.T) {
	// Service configuration.
	maxNetConns := 50
	workerCount := 4
	requestCount := maxNetConns * 2
	s := NewService(&log.Logger{}, &dataFetcherMock, ",", maxNetConns, workerCount)

	// Send concurrent reqeusts to service.
	wg := sync.WaitGroup{}
	for i := 0; i < requestCount; i++ {
		repoCount := i

		go func() {
			wg.Add(1)
			defer wg.Done()

			summary, err := s.CreateRepositorySummary(repoCount)
			assert.Nil(t, err)
			assert.Equal(t, dataFetcherMock.expectedRepositorySummary(repoCount, ","), summary)
		}()
	}
	wg.Wait()

	// Wait for goroutines to send requests.
	time.Sleep(5 * time.Millisecond)
	// Check, that deadlock doesn't happen.
	s.Stop()
}
