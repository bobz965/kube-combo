package controller

import "errors"

// SyncState is the result of calling synchronization callbacks.
type SyncState int

const (
	// The update was processed successfully.
	SyncStateSuccess SyncState = iota
	// The update caused a transient error, the k8s client should
	// retry later.
	SyncStateError
	// The update was accepted, but requires reprocessing all watched
	// services.
	SyncStateReprocessAll
	// The update caused a non transient error, the k8s client should
	// just report and giveup.
	SyncStateErrorNoRetry
)

var errRetry = errors.New("event handling failed, retrying")
