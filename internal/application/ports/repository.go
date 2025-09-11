package ports

import (
	"context"
	"hexagonal-queue/internal/domain/models"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// WalletRepository defines the interface for wallet data persistence
type WalletRepository interface {
	// User operations
	GetUserByUsername(ctx context.Context, username, parentToken string) (*models.User, error)
	GetUserByToken(ctx context.Context, token string) (*models.User, error)
	CreateUser(ctx context.Context, user *models.User) error
	UpdateUser(ctx context.Context, user *models.User) error

	// Account balance operations
	GetAccountBalance(ctx context.Context, username, parentToken string) (*models.AccountBalance, error)
	GetAccountBalancesByParentToken(ctx context.Context, usernames []string, parentToken string) ([]*models.AccountBalance, error)
	CreateAccountBalance(ctx context.Context, balance *models.AccountBalance) error
	UpdateAccountBalance(ctx context.Context, username, parentToken string, newBalance primitive.Decimal128) error

	// Wallet statement operations
	GetLastWalletStatement(ctx context.Context, username, parentToken string) (*models.WalletStatement, error)
	CreateWalletStatement(ctx context.Context, statement *models.WalletStatement) (*models.WalletStatement, error)
	GetWalletStatements(ctx context.Context, username, parentToken string, limit int) ([]*models.WalletStatement, error)
	ClearOldStatements(ctx context.Context, username, parentToken string, keepCount int) error

	// Transaction operations (for ACID compliance)
	StartTransaction(ctx context.Context) (TransactionContext, error)
}

// TransactionContext defines transaction context interface
type TransactionContext interface {
	Commit(ctx context.Context) error
	Abort(ctx context.Context) error
	Context() context.Context
}

// CacheRepository defines the interface for caching operations
type CacheRepository interface {
	Get(ctx context.Context, key string) (interface{}, error)
	Set(ctx context.Context, key string, value interface{}, duration int) error
	Delete(ctx context.Context, key string) error
	Exists(ctx context.Context, key string) (bool, error)
}
