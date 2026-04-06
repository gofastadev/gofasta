package di

import (
	"log/slog"

	"github.com/healtronlabs/gofasta/app/graphql/resolvers"
	"github.com/healtronlabs/gofasta/pkg/auth"
	"github.com/healtronlabs/gofasta/pkg/cache"
	"github.com/healtronlabs/gofasta/pkg/encryption"
	"github.com/healtronlabs/gofasta/pkg/i18n"
	"github.com/healtronlabs/gofasta/pkg/mailer"
	"github.com/healtronlabs/gofasta/pkg/notify"
	repoInterfaces "github.com/healtronlabs/gofasta/app/repositories/interfaces"
	"github.com/healtronlabs/gofasta/app/rest/controllers"
	svcInterfaces "github.com/healtronlabs/gofasta/app/services/interfaces"
	"github.com/healtronlabs/gofasta/app/validators"
	"github.com/healtronlabs/gofasta/configs"
	"github.com/healtronlabs/gofasta/pkg/queue"
	"github.com/healtronlabs/gofasta/pkg/session"
	"github.com/healtronlabs/gofasta/pkg/storage"
	"github.com/healtronlabs/gofasta/pkg/websocket"
	"gorm.io/gorm"
)

// ServiceContainer holds all application dependencies, assembled by Wire.
type ServiceContainer struct {
	Config         *configs.AppConfig
	DB             *gorm.DB
	Logger         *slog.Logger
	Validator      *validators.AppValidator
	EmailSender    mailer.EmailSender
	CacheService   cache.CacheService
	JWTService     *auth.JWTService
	RBACService    *auth.RBACService
	StorageService storage.StorageService
	QueueService   queue.QueueService
	WebSocketHub   *websocket.Hub
	Notifier       *notify.Notifier
	I18nService    *i18n.I18nService
	Encrypter      *encryption.Encrypter
	SessionStore   *session.Store
	UserRepo       repoInterfaces.UserRepositoryInterface
	UserService    svcInterfaces.UserServiceInterface
	UserController *controllers.UserController
	Resolver       *resolvers.Resolver
}
