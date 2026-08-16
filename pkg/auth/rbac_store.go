package auth

import (
	"fmt"
	"net"
	"strings"

	"github.com/casbin/casbin/v2/persist"
	gormadapter "github.com/casbin/gorm-adapter/v3"
	rediswatcher "github.com/casbin/redis-watcher/v2"
	"github.com/gofastadev/gofasta/pkg/config"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// DefaultPolicyChannel is the Redis pub/sub channel policy-change
// notifications are published on. Every replica sharing a policy must watch the
// same channel; replicas on different channels never hear each other.
const DefaultPolicyChannel = "casbin/policy_updated"

// The watcher constructors, named so a test can substitute them. Both dial
// Redis on the way in, so the cluster branch is otherwise only reachable with a
// real Redis cluster on the machine running the tests — and a branch that
// selects the wrong client is exactly the mistake [NewRedisWatcher]'s signature
// exists to prevent.
var (
	newStandaloneWatcher = rediswatcher.NewWatcher
	newClusterWatcher    = rediswatcher.NewWatcherWithCluster
)

// NewGormAdapter stores Casbin policy in the given database, in a `casbin_rule`
// table the adapter creates if absent.
//
// Reusing the application's *gorm.DB rather than opening a connection means the
// policy participates in the same pool and the same migrations, and one fewer
// credential is configured.
func NewGormAdapter(db *gorm.DB) (persist.Adapter, error) {
	if db == nil {
		return nil, fmt.Errorf("rbac: db is required")
	}
	adapter, err := gormadapter.NewAdapterByDB(db)
	if err != nil {
		return nil, fmt.Errorf("rbac: gorm adapter init: %w", err)
	}
	return adapter, nil
}

// WatcherConfig configures the Redis-backed Casbin watcher.
type WatcherConfig struct {
	// Addr is "host:port" for a standalone Redis. For a cluster, list every
	// node in Addrs instead.
	Addr string
	// Addrs, when non-empty, selects the cluster client and Addr is ignored.
	Addrs []string
	// Password authenticates to Redis. Omitting it against a Redis with
	// `requirepass` set fails the watcher, not the application.
	Password string
	// DB selects the Redis logical database. Ignored in cluster mode, which
	// has only database 0.
	DB int
	// Channel defaults to DefaultPolicyChannel.
	Channel string
	// IgnoreSelf drops notifications this process published. Almost always
	// what you want: the process that changed the policy already has it.
	IgnoreSelf bool
}

// NewRedisWatcher builds a Casbin watcher that broadcasts policy changes over
// Redis pub/sub, so every replica reloads when any one of them makes a change.
//
// Pass it to [WithWatcher]. Without a watcher, a policy edit reaches only the
// replica that handled it, and a revoked permission remains usable on the
// others until they restart.
//
// Two mistakes this signature exists to prevent, both of which fail at startup
// with a clear error rather than degrading quietly:
//
//   - Omitting Password against a Redis with `requirepass` set. The watcher's
//     Ping returns "NOAUTH Authentication required", and code that logs the
//     error and carries on ends up with policy invalidation permanently off
//     while everything else works.
//   - Reaching for the cluster client to talk to a standalone Redis. They take
//     their password from different places, so a password set for one is
//     invisible to the other. Addrs selects cluster mode explicitly; Addr does
//     not.
func NewRedisWatcher(cfg WatcherConfig) (persist.Watcher, error) {
	channel := cfg.Channel
	if channel == "" {
		channel = DefaultPolicyChannel
	}

	if len(cfg.Addrs) > 0 {
		watcher, err := newClusterWatcher(strings.Join(cfg.Addrs, ","), rediswatcher.WatcherOptions{
			ClusterOptions: redis.ClusterOptions{Password: cfg.Password},
			Channel:        channel,
			IgnoreSelf:     cfg.IgnoreSelf,
		})
		if err != nil {
			return nil, fmt.Errorf("rbac: redis cluster watcher on %v: %w", cfg.Addrs, err)
		}
		return watcher, nil
	}

	if cfg.Addr == "" {
		return nil, fmt.Errorf("rbac: watcher needs Addr (standalone) or Addrs (cluster)")
	}

	watcher, err := newStandaloneWatcher(cfg.Addr, rediswatcher.WatcherOptions{
		Options:    redis.Options{Password: cfg.Password, DB: cfg.DB},
		Channel:    channel,
		IgnoreSelf: cfg.IgnoreSelf,
	})
	if err != nil {
		return nil, fmt.Errorf("rbac: redis watcher on %s: %w", cfg.Addr, err)
	}
	return watcher, nil
}

// WatcherConfigFromRedis derives a WatcherConfig from the application's Redis
// configuration, so the watcher cannot drift from the Redis everything else
// talks to — in particular it cannot end up without the password.
func WatcherConfigFromRedis(cfg config.RedisConfig, channel string) WatcherConfig {
	port := cfg.Port
	if port == "" {
		port = "6379"
	}
	return WatcherConfig{
		Addr:       net.JoinHostPort(cfg.Host, port),
		Password:   cfg.Password,
		DB:         cfg.DB,
		Channel:    channel,
		IgnoreSelf: true,
	}
}
