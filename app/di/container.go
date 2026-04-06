package di

import (
	"log/slog"

	"github.com/gofastadev/gofasta/app/graphql/resolvers"
	"github.com/gofastadev/gofasta/pkg/auth"
	"github.com/gofastadev/gofasta/pkg/cache"
	"github.com/gofastadev/gofasta/pkg/encryption"
	"github.com/gofastadev/gofasta/pkg/i18n"
	"github.com/gofastadev/gofasta/pkg/mailer"
	"github.com/gofastadev/gofasta/pkg/notify"
	repoInterfaces "github.com/gofastadev/gofasta/app/repositories/interfaces"
	"github.com/gofastadev/gofasta/app/rest/controllers"
	svcInterfaces "github.com/gofastadev/gofasta/app/services/interfaces"
	"github.com/gofastadev/gofasta/app/validators"
	"github.com/gofastadev/gofasta/configs"
	"github.com/gofastadev/gofasta/pkg/queue"
	"github.com/gofastadev/gofasta/pkg/session"
	"github.com/gofastadev/gofasta/pkg/storage"
	"github.com/gofastadev/gofasta/pkg/websocket"
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
