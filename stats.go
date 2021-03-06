package main

import (
	"sync"
	"time"
)

type Statistics struct {
	filesToScan  int
	filesScanned int
	linesScanned int
	matchesFound int
	startTime    time.Time
	mux          sync.Mutex
}

func NewStatistics() *Statistics {
	return &Statistics{
		filesScanned: 0,
		linesScanned: 0,
		matchesFound: 0,
		startTime:    time.Now(),
	}
}

func (s *Statistics) IncrFilesToScan() {
	s.mux.Lock()
	s.filesToScan++
	s.mux.Unlock()
}

func (s *Statistics) IncrLineCount() {
	s.mux.Lock()
	s.linesScanned++
	s.mux.Unlock()
}

func (s *Statistics) IncrFileCount() {
	s.mux.Lock()
	s.filesScanned++
	s.mux.Unlock()
}

func (s *Statistics) IncrMatchCount() {
	s.mux.Lock()
	s.matchesFound++
	s.mux.Unlock()
}

func (s *Statistics) LineCount() int {
	s.mux.Lock()
	defer s.mux.Unlock()
	return s.linesScanned
}

func (s *Statistics) FilesToScanCount() int {
	s.mux.Lock()
	defer s.mux.Unlock()
	return s.filesToScan
}

func (s *Statistics) FileCount() int {
	s.mux.Lock()
	defer s.mux.Unlock()
	return s.filesScanned
}

func (s *Statistics) MatchCount() int {
	s.mux.Lock()
	defer s.mux.Unlock()
	return s.matchesFound
}

func (s *Statistics) ElapsedTime() time.Duration {
	return time.Now().Sub(s.startTime)
}
