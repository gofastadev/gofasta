package notify

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&DBNotification{}))
	return db
}

func TestDBNotification_TableName(t *testing.T) {
	n := DBNotification{}
	assert.Equal(t, "notifications", n.TableName())
}

func TestNewDatabaseChannel(t *testing.T) {
	db := setupTestDB(t)
	ch := NewDatabaseChannel(db)
	require.NotNil(t, ch)
	assert.NotNil(t, ch.db)
}

func TestDatabaseChannel_Channel(t *testing.T) {
	ch := NewDatabaseChannel(nil)
	assert.Equal(t, ChannelDatabase, ch.Channel())
}

func TestDatabaseChannel_Send_Success(t *testing.T) {
	db := setupTestDB(t)
	ch := NewDatabaseChannel(db)

	recipient := Recipient{
		ID:    "user-123",
		Email: "test@example.com",
		Name:  "Test User",
	}
	notification := Notification{
		Subject: "Welcome",
		Body:    "Welcome to the platform",
		Data:    map[string]any{"key": "value"},
	}

	err := ch.Send(context.Background(), recipient, notification)
	require.NoError(t, err)

	var records []DBNotification
	require.NoError(t, db.Find(&records).Error)
	require.Len(t, records, 1)

	record := records[0]
	assert.Equal(t, "user-123", record.UserID)
	assert.Equal(t, "Welcome", record.Subject)
	assert.Equal(t, "Welcome to the platform", record.Body)
	assert.Contains(t, record.Data, `"key":"value"`)
	assert.NotEmpty(t, record.ID)
	assert.False(t, record.CreatedAt.IsZero())
	assert.Nil(t, record.ReadAt)
}

func TestDatabaseChannel_Send_MultipleNotifications(t *testing.T) {
	db := setupTestDB(t)
	ch := NewDatabaseChannel(db)

	recipient := Recipient{ID: "user-456"}

	for i := 0; i < 3; i++ {
		err := ch.Send(context.Background(), recipient, Notification{
			Subject: "Notification",
			Body:    "Body",
		})
		require.NoError(t, err)
	}

	var count int64
	require.NoError(t, db.Model(&DBNotification{}).Count(&count).Error)
	assert.Equal(t, int64(3), count)
}

func TestDatabaseChannel_Send_NilData(t *testing.T) {
	db := setupTestDB(t)
	ch := NewDatabaseChannel(db)

	err := ch.Send(context.Background(), Recipient{ID: "user-789"}, Notification{
		Subject: "No Data",
		Body:    "Body without data",
		Data:    nil,
	})
	require.NoError(t, err)

	var record DBNotification
	require.NoError(t, db.First(&record).Error)
	assert.Equal(t, "null", record.Data)
}
