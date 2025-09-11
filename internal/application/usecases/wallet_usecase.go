package usecases

import (
	"context"
	"fmt"
	"log"
	"time"

	"hexagonal-queue/internal/application/ports"
	"hexagonal-queue/internal/domain/models"
	"hexagonal-queue/internal/domain/services"
	"hexagonal-queue/pkg/utils"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Ports interface aliases for easier use in use cases
type (
	WalletRepository = ports.WalletRepository
	QueuePort        = ports.QueuePort
	CacheRepository  = ports.CacheRepository
)

// WalletUseCase handles wallet-related business operations
type WalletUseCase struct {
	walletRepo    WalletRepository
	queuePort     QueuePort
	cacheRepo     CacheRepository
	walletService *services.WalletService
}

// NewWalletUseCase creates a new wallet use case
func NewWalletUseCase(walletRepo WalletRepository, queuePort QueuePort) *WalletUseCase {
	return &WalletUseCase{
		walletRepo:    walletRepo,
		queuePort:     queuePort,
		walletService: services.NewWalletService(),
	}
}

// SetCacheRepository sets the cache repository (optional dependency)
func (uc *WalletUseCase) SetCacheRepository(cacheRepo CacheRepository) {
	uc.cacheRepo = cacheRepo
}

// HandleDeposit processes deposit transactions
func (uc *WalletUseCase) HandleDeposit(ctx context.Context, request models.TransactionRequest) models.TransactionResponse {
	// log.Printf("Processing deposit: %s -> %s, amount: %.2f", request.ParentToken, request.Username, request.Amount)

	// Validate request
	if err := uc.walletService.ValidateDepositRequest(request); err != nil {
		return models.TransactionResponse{
			Status:  400,
			Message: err.Error(),
		}
	}

	// Start database transaction
	txCtx, err := uc.walletRepo.StartTransaction(ctx)
	if err != nil {
		log.Printf("Failed to start transaction: %v", err)
		return models.TransactionResponse{
			Status:  500,
			Message: "Failed to start database transaction",
		}
	}

	response := uc.processDeposit(txCtx.Context(), txCtx, request)

	if response.Status == 200 {
		if err := txCtx.Commit(ctx); err != nil {
			log.Printf("Failed to commit transaction: %v", err)
			return models.TransactionResponse{
				Status:  500,
				Message: "Failed to commit transaction",
			}
		}
	} else {
		txCtx.Abort(ctx)
	}

	return response
}

// HandleWithdraw processes withdraw transactions
func (uc *WalletUseCase) HandleWithdraw(ctx context.Context, request models.TransactionRequest) models.TransactionResponse {
	// log.Printf("Processing withdraw: %s -> %s, amount: %.2f", request.ParentToken, request.Username, request.Amount)

	// Get current balance first to validate
	accountBalance, err := uc.walletRepo.GetAccountBalance(ctx, request.Username, request.ParentToken)
	if err != nil {
		return models.TransactionResponse{
			Status:  404,
			Message: "User not found",
		}
	}

	currentBalance, err := utils.ConvertDecimal128ToFloat(accountBalance.Balance)
	if err != nil {
		return models.TransactionResponse{
			Status:  500,
			Message: "Failed to process balance",
		}
	}

	// Validate request
	if err := uc.walletService.ValidateWithdrawRequest(request, currentBalance); err != nil {
		return models.TransactionResponse{
			Status:  400,
			Message: err.Error(),
		}
	}

	// Start database transaction
	txCtx, err := uc.walletRepo.StartTransaction(ctx)
	if err != nil {
		log.Printf("Failed to start transaction: %v", err)
		return models.TransactionResponse{
			Status:  500,
			Message: "Failed to start database transaction",
		}
	}

	response := uc.processWithdraw(txCtx.Context(), txCtx, request)

	if response.Status == 200 {
		if err := txCtx.Commit(ctx); err != nil {
			log.Printf("Failed to commit transaction: %v", err)
			return models.TransactionResponse{
				Status:  500,
				Message: "Failed to commit transaction",
			}
		}
	} else {
		txCtx.Abort(ctx)
	}

	return response
}

// HandleCreateMember creates a new member
func (uc *WalletUseCase) HandleCreateMember(ctx context.Context, request models.TransactionRequest) models.TransactionResponse {
	log.Printf("Creating member: %s", request.Username)

	// Validate username
	if err := utils.ValidateUsername(request.Username); err != nil {
		return models.TransactionResponse{
			Status:  400,
			Message: err.Error(),
		}
	}

	// Check if user already exists
	existingUser, err := uc.walletRepo.GetUserByUsername(ctx, request.Username, request.ParentToken)
	if err == nil && existingUser != nil {
		return models.TransactionResponse{
			Status:  409,
			Message: "User already exists",
		}
	}

	parentUser, err := uc.walletRepo.GetUserByToken(ctx, request.ParentToken)
	if err != nil {
		return models.TransactionResponse{
			Status:  500,
			Message: "Failed to get parent user",
		}
	}

	if parentUser == nil {
		return models.TransactionResponse{
			Status:  404,
			Message: "Parent user not found",
		}
	}

	uuid := utils.GenerateUUID()

	// Create new user
	user := &models.User{
		ID:          primitive.NewObjectID(),
		Username:    request.Username,
		ParentToken: parentUser.Token,
		Token:       uuid,
		TypeMember:  "member", // default type
		Status:      1,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	if err := uc.walletRepo.CreateUser(ctx, user); err != nil {
		log.Printf("Failed to create user: %v", err)
		return models.TransactionResponse{
			Status:  500,
			Message: "Failed to create user",
		}
	}

	// Create initial account balance
	balance := &models.AccountBalance{
		ID:          primitive.NewObjectID(),
		Username:    request.Username,
		ParentToken: request.ParentToken,
		Token:       request.Token,
		Balance:     primitive.NewDecimal128(0, 0), // Zero balance
		Status:      1,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	if err := uc.walletRepo.CreateAccountBalance(ctx, balance); err != nil {
		log.Printf("Failed to create account balance: %v", err)
		return models.TransactionResponse{
			Status:  500,
			Message: "Failed to create account balance",
		}
	}

	return models.TransactionResponse{
		Status:  200,
		Message: "Member created successfully",
		Data: models.CreateMemberRequest{
			DateTime: time.Now().UTC(),
			Username: request.Username,
		},
	}
}

// HandleBalanceCheck retrieves user balance
func (uc *WalletUseCase) HandleBalanceCheck(ctx context.Context, username, parentToken string) models.TransactionResponse {
	accountBalance, err := uc.getAccountBalanceWithCache(ctx, username, parentToken)
	if err != nil {
		return models.TransactionResponse{
			Status:  404,
			Message: "User not found",
		}
	}

	balance, err := utils.ConvertDecimal128ToFloat(accountBalance.Balance)
	if err != nil {
		return models.TransactionResponse{
			Status:  500,
			Message: "Failed to process balance",
		}
	}

	return models.TransactionResponse{
		Status:  200,
		Message: "Success",
		Data: models.BalanceResponse{
			Username:       username,
			CurrentBalance: balance,
		},
	}
}

// GetMultipleBalances retrieves balances for multiple users
func (uc *WalletUseCase) GetMultipleBalances(ctx context.Context, usernames []string, parentToken string) models.TransactionResponse {
	balances, err := uc.walletRepo.GetAccountBalancesByParentToken(ctx, usernames, parentToken)
	if err != nil {
		return models.TransactionResponse{
			Status:  500,
			Message: "Failed to retrieve balances",
		}
	}

	var result []models.BalanceResponse
	for _, balance := range balances {
		balanceFloat, err := utils.ConvertDecimal128ToFloat(balance.Balance)
		if err != nil {
			log.Printf("Failed to convert balance for user %s: %v", balance.Username, err)
			balanceFloat = 0
		}

		result = append(result, models.BalanceResponse{
			Username:       balance.Username,
			CurrentBalance: balanceFloat,
		})
	}

	return models.TransactionResponse{
		Status:  200,
		Message: "Success",
		Data:    result,
	}
}

// processDeposit handles the deposit logic within a transaction
func (uc *WalletUseCase) processDeposit(ctx context.Context, txCtx ports.TransactionContext, request models.TransactionRequest) models.TransactionResponse {
	// Get user data
	user, err := uc.walletRepo.GetUserByUsername(ctx, request.Username, request.ParentToken)
	if err != nil {

		fmt.Printf("Error getting user by username: %v\n", err)
		return models.TransactionResponse{
			Status:  404,
			Message: "User not found",
		}
	}

	// Get current account balance
	accountBalance, err := uc.walletRepo.GetAccountBalance(ctx, request.Username, request.ParentToken)
	if err != nil {
		return models.TransactionResponse{
			Status:  404,
			Message: "Account balance not found",
		}
	}

	// Get last wallet statement
	lastStatement, err := uc.walletRepo.GetLastWalletStatement(ctx, request.Username, request.ParentToken)
	if err != nil {
		return models.TransactionResponse{
			Status:  500,
			Message: "Failed to get wallet statement",
		}
	}

	// Validate balance consistency
	if err := uc.walletService.ValidateBalanceConsistency(accountBalance.Balance, lastStatement.AfterCredit); err != nil {
		return models.TransactionResponse{
			Status:  500,
			Message: "Balance inconsistency detected",
		}
	}

	// Create deposit statement
	statement, err := uc.walletService.CreateDepositStatement(request, accountBalance.Balance, *user)
	if err != nil {
		return models.TransactionResponse{
			Status:  500,
			Message: fmt.Sprintf("Failed to create deposit statement: %v", err),
		}
	}

	// Save wallet statement
	_, err = uc.walletRepo.CreateWalletStatement(ctx, &statement)
	if err != nil {
		return models.TransactionResponse{
			Status:  500,
			Message: "Failed to save wallet statement",
		}
	}

	// Update account balance
	if err := uc.walletRepo.UpdateAccountBalance(ctx, request.Username, request.ParentToken, statement.AfterCredit); err != nil {
		return models.TransactionResponse{
			Status:  500,
			Message: "Failed to update account balance",
		}
	}

	beforeBalance, _ := utils.ConvertDecimal128ToFloat(statement.BeforeCredit)
	afterBalance, _ := utils.ConvertDecimal128ToFloat(statement.AfterCredit)

	return models.TransactionResponse{
		Status:  200,
		Message: "Deposit successful",
		Data: models.DepositWithdrawResponse{
			Username:       request.Username,
			CurrentBalance: afterBalance,
			BeforeBalance:  beforeBalance,
			AfterBalance:   afterBalance,
		},
	}
}

// processWithdraw handles the withdraw logic within a transaction
func (uc *WalletUseCase) processWithdraw(ctx context.Context, txCtx ports.TransactionContext, request models.TransactionRequest) models.TransactionResponse {
	// Get user data
	user, err := uc.walletRepo.GetUserByUsername(ctx, request.Username, request.ParentToken)
	if err != nil {
		return models.TransactionResponse{
			Status:  404,
			Message: "User not found",
		}
	}

	// Get current account balance
	accountBalance, err := uc.walletRepo.GetAccountBalance(ctx, request.Username, request.ParentToken)
	if err != nil {
		return models.TransactionResponse{
			Status:  404,
			Message: "Account balance not found",
		}
	}

	// Get last wallet statement
	lastStatement, err := uc.walletRepo.GetLastWalletStatement(ctx, request.Username, request.ParentToken)
	if err != nil {
		return models.TransactionResponse{
			Status:  500,
			Message: "Failed to get wallet statement",
		}
	}

	// Validate balance consistency
	if err := uc.walletService.ValidateBalanceConsistency(accountBalance.Balance, lastStatement.AfterCredit); err != nil {
		return models.TransactionResponse{
			Status:  500,
			Message: "Balance inconsistency detected",
		}
	}

	// Create withdraw statement
	statement, err := uc.walletService.CreateWithdrawStatement(request, accountBalance.Balance, *user)
	if err != nil {
		return models.TransactionResponse{
			Status:  400,
			Message: fmt.Sprintf("Failed to create withdraw statement: %v", err),
		}
	}

	// Save wallet statement
	_, err = uc.walletRepo.CreateWalletStatement(ctx, &statement)
	if err != nil {
		return models.TransactionResponse{
			Status:  500,
			Message: "Failed to save wallet statement",
		}
	}

	// Update account balance
	if err := uc.walletRepo.UpdateAccountBalance(ctx, request.Username, request.ParentToken, statement.AfterCredit); err != nil {
		return models.TransactionResponse{
			Status:  500,
			Message: "Failed to update account balance",
		}
	}

	beforeBalance, _ := utils.ConvertDecimal128ToFloat(statement.BeforeCredit)
	afterBalance, _ := utils.ConvertDecimal128ToFloat(statement.AfterCredit)

	return models.TransactionResponse{
		Status:  200,
		Message: "Withdraw successful",
		Data: models.DepositWithdrawResponse{
			Username:       request.Username,
			CurrentBalance: afterBalance,
			BeforeBalance:  beforeBalance,
			AfterBalance:   afterBalance,
		},
	}
}

// getAccountBalanceWithCache retrieves account balance with caching if available
func (uc *WalletUseCase) getAccountBalanceWithCache(ctx context.Context, username, parentToken string) (*models.AccountBalance, error) {
	// Try cache first if available
	if uc.cacheRepo != nil {
		cacheKey := fmt.Sprintf("balance:%s:%s", parentToken, username)
		if cached, err := uc.cacheRepo.Get(ctx, cacheKey); err == nil && cached != nil {
			if balance, ok := cached.(*models.AccountBalance); ok {
				return balance, nil
			}
		}
	}

	// Get from database
	balance, err := uc.walletRepo.GetAccountBalance(ctx, username, parentToken)
	if err != nil {
		return nil, err
	}

	// Cache the result if cache is available
	if uc.cacheRepo != nil {
		cacheKey := fmt.Sprintf("balance:%s:%s", parentToken, username)
		uc.cacheRepo.Set(ctx, cacheKey, balance, 300) // Cache for 5 minutes
	}

	return balance, nil
}
