package main

import (
	"encoding/json"
	"errors"
	"strings"
	"sync"

	log "github.com/sirupsen/logrus"
)

// RepositoryData holds information from about repositroy, received from Gitlab API.
type RepositoryData struct {
	Name       string `json:"name"`
	ForksCount int    `json:"forksCount"`
}

// RepositoryDataFetcher fetches data via Gitlab API.
type RepositoryDataFetcher interface {
	FetchRepositoryData(repoCount int) ([]RepositoryData, error)
}

// RepositorySummary holds result of RepositoryData processing.
type RepositorySummary struct {
	JoinedNames string `json:"joinedNames"`
	TotalForks  int    `json:"totalForks"`
}

func (summary RepositorySummary) String() string {
	jsonData, _ := json.Marshal(summary)
	return string(jsonData)
}

type summaryTask struct {
	data   []RepositoryData
	result chan RepositorySummary
}

type Service struct {
	log log.FieldLogger

	// Configuration.
	repoFetcher RepositoryDataFetcher
	repoNameSep string

	// Resource management.
	netConns chan struct{}
	sumTasks chan summaryTask

	// Graceful termination.
	wg     sync.WaitGroup
	stopCh chan struct{}
}

func NewService(log log.FieldLogger, repoFetcher RepositoryDataFetcher,
	repoNameSep string, maxNetConns int, workerCount int) *Service {

	s := &Service{
		log: log,

		repoFetcher: repoFetcher,
		repoNameSep: repoNameSep,

		netConns: make(chan struct{}, maxNetConns),
		sumTasks: make(chan summaryTask),

		wg:     sync.WaitGroup{},
		stopCh: make(chan struct{}),
	}

	// Limit number of network-bounded and cpu-bounded goroutines.
	for i := 0; i < maxNetConns; i++ {
		s.netConns <- struct{}{}
	}
	for i := 0; i < workerCount; i++ {
		go s.repoistorySummaryWorker()
	}

	return s
}

func (s *Service) CreateRepositorySummary(repoCount int) (RepositorySummary, error) {
	// Check for termination.
	select {
	case <-s.stopCh:
		return RepositorySummary{}, errors.New("service terminated")
	default:
	}

	// Fetch repository data and put it in processing queue.
	data, err := s.fetchRepositoryData(repoCount)
	if err != nil {
		s.log.Errorln(err)
		return RepositorySummary{}, err
	}
	task := summaryTask{
		data:   data,
		result: make(chan RepositorySummary),
	}
	s.sumTasks <- task

	// Wait to processing to finish.
	select {
	case result := <-task.result:
		return result, nil
	case <-s.stopCh:
		return RepositorySummary{}, errors.New("service terminated")
	}
}

func (s *Service) Stop() {
	s.log.Infoln("Terminating service")
	close(s.stopCh)
	s.wg.Wait()
}

func (s *Service) fetchRepositoryData(repoCount int) ([]RepositoryData, error) {
	// Accuire network connection and release it back, when done.
	conn := <-s.netConns
	defer func() {
		s.netConns <- conn
	}()

	data, err := s.repoFetcher.FetchRepositoryData(repoCount)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func (s *Service) repoistorySummaryWorker() {
	s.wg.Add(1)
	defer s.wg.Done()

	for {
		// Get task from processing queue.
		var task summaryTask
		select {
		case task = <-s.sumTasks:
		case <-s.stopCh:
			return
		}

		s.log.Debugf("Start creating summary of %d repos\n", len(task.data))

		// Create summary and send it back.
		repoNames := make([]string, len(task.data))
		totalForks := 0
		for i, data := range task.data {
			repoNames[i] = data.Name
			totalForks += data.ForksCount
		}

		s.log.Debugf("Creating repositroy summary of %d repos finished\n", len(task.data))

		task.result <- RepositorySummary{
			JoinedNames: strings.Join(repoNames, s.repoNameSep),
			TotalForks:  totalForks,
		}
	}
}
