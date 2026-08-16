package auth

import (
	"fmt"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/casbin/casbin/v2/persist"
	rediswatcher "github.com/casbin/redis-watcher/v2"
	"github.com/gofastadev/gofasta/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// newSQLiteDB returns an isolated in-memory database. The shared cache and a
// per-test name let two connections see the same data without two tests seeing
// each other's.
func newSQLiteDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	sqlDB, err := db.DB()
	require.NoError(t, err)
	// An in-memory SQLite database lives as long as a connection to it does.
	// Let the pool close one and the schema disappears mid-test.
	sqlDB.SetMaxIdleConns(1)
	t.Cleanup(func() { _ = sqlDB.Close() })
	return db
}

func adapterFor(t *testing.T, db *gorm.DB) persist.Adapter {
	t.Helper()
	adapter, err := NewGormAdapter(db)
	require.NoError(t, err)
	return adapter
}

func newSQLiteAdapter(t *testing.T) persist.Adapter {
	t.Helper()
	return adapterFor(t, newSQLiteDB(t))
}

func TestNewGormAdapter_RequiresADatabase(t *testing.T) {
	_, err := NewGormAdapter(nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "db is required")
}

// ---------- the Redis watcher ----------

func securedRedis(t *testing.T, password string) *miniredis.Miniredis {
	t.Helper()
	server := miniredis.RunT(t)
	if password != "" {
		server.RequireAuth(password)
	}
	return server
}

// This is the failure the WatcherConfig shape exists to prevent, and it was
// found in production code: the watcher was built without a password against a
// Redis with `requirepass` set. Redis answers NOAUTH, the watcher constructor
// fails, and a caller that logs the error and continues runs forever with
// policy invalidation silently switched off — every other subsystem still works,
// so nothing looks wrong.
func TestNewRedisWatcher_WithoutThePasswordFailsLoudly(t *testing.T) {
	server := securedRedis(t, "s3cr3t")

	_, err := NewRedisWatcher(WatcherConfig{Addr: server.Addr()})

	require.Error(t, err, "a watcher with no password must not appear to succeed")
	assert.Contains(t, err.Error(), "NOAUTH")
}

func TestNewRedisWatcher_WithThePasswordConnects(t *testing.T) {
	server := securedRedis(t, "s3cr3t")

	watcher, err := NewRedisWatcher(WatcherConfig{Addr: server.Addr(), Password: "s3cr3t"})

	require.NoError(t, err)
	require.NotNil(t, watcher)
	t.Cleanup(func() { watcher.Close() })
}

func TestNewRedisWatcher_NeedsAnAddress(t *testing.T) {
	_, err := NewRedisWatcher(WatcherConfig{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Addr")
}

func TestNewRedisWatcher_DefaultsTheChannel(t *testing.T) {
	// Replicas watching different channels never hear each other, so the
	// default has to be shared rather than per-caller.
	server := securedRedis(t, "")

	watcher, err := NewRedisWatcher(WatcherConfig{Addr: server.Addr()})
	require.NoError(t, err)
	t.Cleanup(func() { watcher.Close() })

	assert.Equal(t, "casbin/policy_updated", DefaultPolicyChannel)
}

func TestWatcherConfigFromRedis_CarriesThePassword(t *testing.T) {
	// Deriving the watcher's connection from the application's Redis config is
	// what stops the two from drifting — above all, it stops the password from
	// being left out.
	got := WatcherConfigFromRedis(config.RedisConfig{
		Host:     "redis.internal",
		Port:     "6379",
		Password: "s3cr3t",
		DB:       3,
	}, "")

	assert.Equal(t, "redis.internal:6379", got.Addr)
	assert.Equal(t, "s3cr3t", got.Password)
	assert.Equal(t, 3, got.DB)
	assert.Equal(t, "", got.Channel, "empty channel means NewRedisWatcher applies the default")
	assert.True(t, got.IgnoreSelf, "a process does not need to hear its own change")
}

func TestWatcherConfigFromRedis_DefaultsThePort(t *testing.T) {
	got := WatcherConfigFromRedis(config.RedisConfig{Host: "redis.internal"}, "custom")
	assert.Equal(t, "redis.internal:6379", got.Addr)
	assert.Equal(t, "custom", got.Channel)
}

// ---------- watcher + enforcer, wired together ----------

func TestNewRBACServiceWithAdapter_AttachesTheWatcher(t *testing.T) {
	server := securedRedis(t, "s3cr3t")
	watcher, err := NewRedisWatcher(WatcherConfig{
		Addr: server.Addr(), Password: "s3cr3t", IgnoreSelf: true,
	})
	require.NoError(t, err)
	t.Cleanup(func() { watcher.Close() })

	svc, err := NewRBACServiceWithAdapter(writeDomainModel(t), newSQLiteAdapter(t), WithWatcher(watcher))
	require.NoError(t, err)
	require.NotNil(t, svc)

	// The enforcer still decides correctly with a watcher attached.
	_, err = svc.AddRoleForUserInDomain("alice", "admin", "inst-a")
	require.NoError(t, err)
	_, err = svc.AddPolicyInDomain("admin", "inst-a", "/courses/*", "create", EffectAllow)
	require.NoError(t, err)

	allowed, err := svc.EnforceInDomain("alice", "inst-a", "/courses/intro", "create")
	require.NoError(t, err)
	assert.True(t, allowed)
}

func TestNewGormAdapter_ReportsAnUnusableDatabase(t *testing.T) {
	// The adapter creates its casbin_rule table on the way in, so a database
	// that is already closed fails here rather than at the first authorization
	// check — which is where it would otherwise show up, in a request.
	db := newSQLiteDB(t)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	require.NoError(t, sqlDB.Close())

	adapter, err := NewGormAdapter(db)

	require.Error(t, err)
	assert.Nil(t, adapter)
	assert.Contains(t, err.Error(), "gorm adapter init")
}

// ---------- the cluster branch ----------

// noopWatcher is a watcher that attaches and does nothing, standing in for one
// built against a real Redis cluster.
type noopWatcher struct{}

func (noopWatcher) SetUpdateCallback(func(string)) error { return nil }
func (noopWatcher) Update() error                        { return nil }
func (noopWatcher) Close()                               {}

// stubClusterWatcher replaces the cluster constructor for the duration of a
// test and records what it was asked for.
func stubClusterWatcher(t *testing.T, watcher persist.Watcher, err error) *struct {
	addrs   string
	options rediswatcher.WatcherOptions
} {
	t.Helper()

	got := &struct {
		addrs   string
		options rediswatcher.WatcherOptions
	}{}

	original := newClusterWatcher
	newClusterWatcher = func(addrs string, options rediswatcher.WatcherOptions) (persist.Watcher, error) {
		got.addrs = addrs
		got.options = options
		return watcher, err
	}
	t.Cleanup(func() { newClusterWatcher = original })

	return got
}

func TestNewRedisWatcher_AddrsSelectsTheClusterClient(t *testing.T) {
	// Addrs is what makes the choice explicit. Reaching for the cluster client
	// to talk to a standalone Redis — or the reverse — means the password is
	// read from a field the other client never looks at, and the watcher comes
	// up unauthenticated against a Redis that requires a password.
	got := stubClusterWatcher(t, noopWatcher{}, nil)

	watcher, err := NewRedisWatcher(WatcherConfig{
		Addrs:      []string{"node-a:6379", "node-b:6379", "node-c:6379"},
		Password:   "s3cr3t",
		Channel:    "policy",
		IgnoreSelf: true,
	})

	require.NoError(t, err)
	require.NotNil(t, watcher)

	assert.Equal(t, "node-a:6379,node-b:6379,node-c:6379", got.addrs,
		"every node has to be passed, or the client only knows part of the cluster")
	assert.Equal(t, "s3cr3t", got.options.ClusterOptions.Password,
		"the password must go to the cluster options, which is where the cluster client reads it")
	assert.Equal(t, "policy", got.options.Channel)
	assert.True(t, got.options.IgnoreSelf)
}

func TestNewRedisWatcher_ClusterDefaultsTheChannel(t *testing.T) {
	// Replicas watching different channels never hear each other, and the
	// cluster branch has to default it the same way the standalone one does.
	got := stubClusterWatcher(t, noopWatcher{}, nil)

	_, err := NewRedisWatcher(WatcherConfig{Addrs: []string{"node-a:6379"}})

	require.NoError(t, err)
	assert.Equal(t, DefaultPolicyChannel, got.options.Channel)
}

func TestNewRedisWatcher_ClusterFailureIsReported(t *testing.T) {
	stubClusterWatcher(t, nil, fmt.Errorf("NOAUTH Authentication required"))

	watcher, err := NewRedisWatcher(WatcherConfig{Addrs: []string{"node-a:6379"}})

	require.Error(t, err)
	assert.Nil(t, watcher)
	assert.Contains(t, err.Error(), "redis cluster watcher")
	assert.Contains(t, err.Error(), "NOAUTH",
		"a caller that logs this and carries on must at least be able to read why")
}

func TestNewRedisWatcher_AddrsWinsOverAddr(t *testing.T) {
	// Both set is a misconfiguration, and the documented rule is that Addrs
	// selects cluster mode and Addr is ignored. Silently preferring Addr would
	// dial one node of a cluster and appear to work.
	got := stubClusterWatcher(t, noopWatcher{}, nil)

	_, err := NewRedisWatcher(WatcherConfig{
		Addr:  "standalone:6379",
		Addrs: []string{"node-a:6379"},
	})

	require.NoError(t, err)
	assert.Equal(t, "node-a:6379", got.addrs)
}
