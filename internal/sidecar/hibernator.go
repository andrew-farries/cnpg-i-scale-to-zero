package sidecar

import "context"

// Hibernator handles hibernation for a cluster.
// Implementations may also handle related tasks like suspending backups.
type Hibernator interface {
	// Hibernate triggers hibernation for the specified cluster.
	// Returns nil if hibernation was successful or skipped (e.g., already hibernated).
	// Returns an error if hibernation failed and should be retried.
	Hibernate(ctx context.Context, namespace, name string) error
}
