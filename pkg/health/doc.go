// Package health provides a health check controller with three endpoints:
// /health (basic liveness), /health/live (process alive), and /health/ready
// (checks database and cache connectivity).
package health
