package main

import (
	"net"
	"time"
)

type BruteForceAttempt struct {
	Count    uint
	LastTime time.Time
	Ip       net.Addr
}

type BruteForceGuard struct {
	attempts []*BruteForceAttempt
}

func NewBruteForceGuard() BruteForceGuard {
	return BruteForceGuard{make([]*BruteForceAttempt, 0)}
}

func (guard *BruteForceGuard) CheckBruteForceAttempt(ip net.Addr) bool {
	guard.attempts = Filter(guard.attempts, func(attempt *BruteForceAttempt) bool {
		return attempt.LastTime.After(time.Now().Add(-time.Minute))
	}) // cleaning old attempts

	currentAttempt := FirstOr(guard.attempts, func(attempt *BruteForceAttempt) bool {
		return attempt.Ip.String() == attempt.Ip.String()
	}, nil)

	if currentAttempt == nil {
		currentAttempt = &BruteForceAttempt{Count: 1, LastTime: time.Now(), Ip: ip}
		guard.attempts = append(guard.attempts, currentAttempt)
	} else {
		currentAttempt.Count++
	}

	return currentAttempt.Count > 100
}
