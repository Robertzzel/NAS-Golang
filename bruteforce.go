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

func (guard *BruteForceGuard) IsBruteForceAttempt(ip net.Addr) bool {
	guard.cleanOldAttempts()
	currentAttempt := guard.getAttemptByIp(ip)

	if currentAttempt == nil {
		currentAttempt = &BruteForceAttempt{Count: 1, LastTime: time.Now(), Ip: ip}
		guard.attempts = append(guard.attempts, currentAttempt)
	} else {
		currentAttempt.Count++
	}

	return currentAttempt.Count > 100
}

func (guard *BruteForceGuard) cleanOldAttempts() {
	guard.attempts = Filter(guard.attempts, func(attempt *BruteForceAttempt) bool {
		return attempt.LastTime.After(time.Now().Add(-time.Minute))
	})
}

func (guard *BruteForceGuard) getAttemptByIp(ip net.Addr) *BruteForceAttempt {
	return FirstOr(guard.attempts, func(attempt *BruteForceAttempt) bool {
		return attempt.Ip.String() == ip.String()
	}, nil)
}
