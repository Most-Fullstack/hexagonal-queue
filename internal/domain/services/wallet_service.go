package services

import (
	"errors"
	"strings"
	"time"

	"hexagonal-queue/internal/domain/models"
	"hexagonal-queue/pkg/utils"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

var (
	ErrInsufficientFunds     = errors.New("insufficient funds")
	ErrInvalidAmount         = errors.New("invalid amount")
	ErrUserNotFound          = errors.New("user not found")
	ErrBalanceMismatch       = errors.New("balance mismatch detected")
	ErrNegativeBalance       = errors.New("balance cannot be negative")
	ErrUnauthorizedOperation = errors.New("unauthorized operation")
)

// WalletService handles wallet business logic
type WalletService struct{}

// NewWalletService creates a new wallet service
func NewWalletService() *WalletService {
	return &WalletService{}
}

// ValidateDepositRequest validates a deposit request
func (ws *WalletService) ValidateDepositRequest(req models.TransactionRequest) error {
	if req.Amount <= 0 {
		return ErrInvalidAmount
	}
	if req.Username == "" {
		return errors.New("username is required")
	}
	if req.ParentToken == "" {
		return errors.New("parent token is required")
	}
	return nil
}

// ValidateWithdrawRequest validates a withdraw request
func (ws *WalletService) ValidateWithdrawRequest(req models.TransactionRequest, currentBalance float64) error {
	if req.Amount <= 0 {
		return ErrInvalidAmount
	}
	if req.Username == "" {
		return errors.New("username is required")
	}
	if req.ParentToken == "" {
		return errors.New("parent token is required")
	}
	if currentBalance < req.Amount {
		return ErrInsufficientFunds
	}
	return nil
}

// CreateDepositStatement creates a wallet statement for deposit
func (ws *WalletService) CreateDepositStatement(req models.TransactionRequest,
	currentBalance primitive.Decimal128, user models.User) (models.WalletStatement, error) {

	amountDecimal, err := utils.ConvertFloatToDecimal128(req.Amount)
	if err != nil {
		return models.WalletStatement{}, err
	}

	newBalance, err := utils.SumDecimal128(currentBalance, amountDecimal)
	if err != nil {
		return models.WalletStatement{}, err
	}

	return models.WalletStatement{
		ID:           primitive.NewObjectID(),
		Datetime:     time.Now().UTC(),
		Username:     req.Username,
		ParentToken:  user.ParentToken,
		Amount:       amountDecimal,
		BeforeCredit: currentBalance,
		AfterCredit:  newBalance,
		Status:       1,
		IsCheck:      1,
		TypeName:     strings.ToUpper(req.TypeName),
		TypeAction:   "DEPOSIT",
		Description:  req.Description,
		Channel:      req.Channel,
	}, nil
}

// CreateWithdrawStatement creates a wallet statement for withdraw
func (ws *WalletService) CreateWithdrawStatement(req models.TransactionRequest,
	currentBalance primitive.Decimal128, user models.User) (models.WalletStatement, error) {

	// Convert amount to negative for withdrawal
	amountDecimal, err := utils.ConvertFloatToDecimal128(-req.Amount)
	if err != nil {
		return models.WalletStatement{}, err
	}

	newBalance, err := utils.SumDecimal128(currentBalance, amountDecimal)
	if err != nil {
		return models.WalletStatement{}, err
	}

	// Check for negative balance
	newBalanceFloat, err := utils.ConvertDecimal128ToFloat(newBalance)
	if err != nil {
		return models.WalletStatement{}, err
	}

	if newBalanceFloat < 0 {
		return models.WalletStatement{}, ErrNegativeBalance
	}

	return models.WalletStatement{
		ID:           primitive.NewObjectID(),
		Datetime:     time.Now().UTC(),
		Username:     req.Username,
		ParentToken:  user.ParentToken,
		Amount:       amountDecimal,
		BeforeCredit: currentBalance,
		AfterCredit:  newBalance,
		Status:       1,
		IsCheck:      1,
		TypeName:     strings.ToUpper(req.TypeName),
		TypeAction:   "WITHDRAW",
		Description:  req.Description,
		Channel:      req.Channel,
	}, nil
}

// CreateParentWithdrawStatement creates a parent withdraw statement for deposit operations
func (ws *WalletService) CreateParentWithdrawStatement(req models.TransactionRequest,
	parentBalance primitive.Decimal128, parent models.User) (models.WalletStatement, error) {

	// Convert amount to negative for parent withdrawal
	amountDecimal, err := utils.ConvertFloatToDecimal128(-req.Amount)
	if err != nil {
		return models.WalletStatement{}, err
	}

	newBalance, err := utils.SumDecimal128(parentBalance, amountDecimal)
	if err != nil {
		return models.WalletStatement{}, err
	}

	// Check for negative balance
	newBalanceFloat, err := utils.ConvertDecimal128ToFloat(newBalance)
	if err != nil {
		return models.WalletStatement{}, err
	}

	if newBalanceFloat < 0 {
		return models.WalletStatement{}, ErrNegativeBalance
	}

	return models.WalletStatement{
		ID:           primitive.NewObjectID(),
		Datetime:     time.Now().UTC(),
		Username:     parent.Username,
		ParentToken:  parent.ParentToken,
		Amount:       amountDecimal,
		BeforeCredit: parentBalance,
		AfterCredit:  newBalance,
		Status:       1,
		IsCheck:      1,
		TypeName:     strings.ToUpper(req.TypeName),
		TypeAction:   "WITHDRAW",
		Description:  req.Description,
		Channel:      req.Channel,
	}, nil
}

// ValidateBalanceConsistency validates that account balance matches last wallet statement
func (ws *WalletService) ValidateBalanceConsistency(accountBalance primitive.Decimal128,
	lastStatementBalance primitive.Decimal128) error {

	isEqual, err := utils.CompareDecimal128(accountBalance, lastStatementBalance)
	if err != nil {
		return err
	}

	if !isEqual {
		return ErrBalanceMismatch
	}

	return nil
}

// CalculateNewBalance calculates new balance after transaction
func (ws *WalletService) CalculateNewBalance(currentBalance primitive.Decimal128,
	amount float64, isDeposit bool) (primitive.Decimal128, error) {

	var amountDecimal primitive.Decimal128
	var err error

	if isDeposit {
		amountDecimal, err = utils.ConvertFloatToDecimal128(amount)
	} else {
		amountDecimal, err = utils.ConvertFloatToDecimal128(-amount)
	}

	if err != nil {
		return primitive.Decimal128{}, err
	}

	newBalance, err := utils.SumDecimal128(currentBalance, amountDecimal)
	if err != nil {
		return primitive.Decimal128{}, err
	}

	// For withdrawals, check for negative balance
	if !isDeposit {
		newBalanceFloat, err := utils.ConvertDecimal128ToFloat(newBalance)
		if err != nil {
			return primitive.Decimal128{}, err
		}

		if newBalanceFloat < 0 {
			return primitive.Decimal128{}, ErrNegativeBalance
		}
	}

	return newBalance, nil
}

// IsParentWithdrawRequired checks if parent withdrawal is required for the operation
func (ws *WalletService) IsParentWithdrawRequired(parent models.User) bool {
	return !strings.EqualFold(parent.TypeMember, "super")
}
