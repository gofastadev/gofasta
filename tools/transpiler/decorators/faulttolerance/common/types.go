// Package common provides shared types and utilities for fault tolerance decorators
package common

import (
	"time"
)

// ActorState represents the state of an actor
type ActorState int

const (
	ActorCreated ActorState = iota
	ActorStarted
	ActorStopped
	ActorFailed
)

// SupervisorStrategy defines the restart strategy for supervised actors
type SupervisorStrategy int

const (
	OneForOne SupervisorStrategy = iota
	OneForAll
	RestForOne
)

// Message represents a message in the actor system
type Message struct {
	ID        string
	Payload   interface{}
	Sender    string
	Timestamp time.Time
}

// ActorRef represents a reference to an actor
type ActorRef struct {
	Path     string
	Address  string
	Name     string
	System   string
}

// SupervisorConfig holds configuration for a supervisor
type SupervisorConfig struct {
	Strategy      SupervisorStrategy
	MaxRetries    int
	RetryInterval time.Duration
	Name          string
}

// ActorConfig holds configuration for an actor
type ActorConfig struct {
	ID           string
	Supervised   bool
	Supervisor   string
	MailboxSize  int
	PoolSize     int
	MaxMessages  int
	Timeout      time.Duration
}

// String returns string representation of ActorState
func (s ActorState) String() string {
	switch s {
	case ActorCreated:
		return "created"
	case ActorStarted:
		return "started"
	case ActorStopped:
		return "stopped"
	case ActorFailed:
		return "failed"
	default:
		return "unknown"
	}
}

// String returns string representation of SupervisorStrategy
func (s SupervisorStrategy) String() string {
	switch s {
	case OneForOne:
		return "OneForOne"
	case OneForAll:
		return "OneForAll"
	case RestForOne:
		return "RestForOne"
	default:
		return "OneForOne"
	}
}

// ActorInterface defines the common interface for all actors
type ActorInterface interface {
	SendMessage(msg Message) error
	Stop()
	GetStats() map[string]interface{}
}

// SupervisorInterface defines the common interface for supervisors  
type SupervisorInterface interface {
	AddChild(name string)
	RemoveChild(name string)
	GetStats() map[string]interface{}
}

// ActorRefInterface defines the common interface for actor references
type ActorRefInterface interface {
	SendMessage(msg Message) error
	HealthCheck() map[string]bool
	GetStats() map[string]interface{}
}