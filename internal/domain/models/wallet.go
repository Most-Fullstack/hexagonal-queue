package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// WalletStatement represents a wallet transaction statement
type WalletStatement struct {
	ID           primitive.ObjectID   `bson:"_id" json:"id,omitempty"`
	Datetime     time.Time            `bson:"datetime" json:"datetime"`
	Username     string               `bson:"username" json:"username"`
	ParentToken  string               `bson:"parent_token" json:"parent_token"` // token of the parent user who created this
	Amount       primitive.Decimal128 `bson:"amount" json:"amount"`
	BeforeCredit primitive.Decimal128 `bson:"before_credit" json:"before_credit"`
	AfterCredit  primitive.Decimal128 `bson:"after_credit" json:"after_credit"`
	Status       int8                 `bson:"status" json:"status"`
	IsCheck      int8                 `bson:"is_check" json:"is_check"`
	TypeName     string               `bson:"type_name" json:"type_name"`     // DEPOSIT, WITHDRAW, REFUND
	TypeAction   string               `bson:"type_action" json:"type_action"` // DEPOSIT, WITHDRAW
	Description  string               `bson:"description" json:"description"`
	Hash         string               `bson:"hash" json:"hash"`
	Channel      string               `bson:"channel" json:"channel"`
}

// AccountBalance represents user's current balance
type AccountBalance struct {
	ID          primitive.ObjectID   `bson:"_id" json:"id,omitempty"`
	Username    string               `bson:"username" json:"username"`
	ParentToken string               `bson:"parent_token" json:"parent_token"`
	Token       string               `bson:"token" json:"token"`
	Balance     primitive.Decimal128 `bson:"balance" json:"balance"`
	Status      int8                 `bson:"status" json:"status"`
	CreatedAt   time.Time            `bson:"created_at" json:"created_at"`
	UpdatedAt   time.Time            `bson:"updated_at" json:"updated_at"`
}

// TransactionRequest represents a wallet transaction request
type TransactionRequest struct {
	ParentToken string  `json:"parent_token" binding:"required"` // parent user token
	Token       string  `json:"token" binding:"required"`        // user token
	Username    string  `json:"username" binding:"required"`
	Action      string  `json:"action" binding:"required"`    // DEPOSIT, WITHDRAW, ADD_MEMBER
	TypeName    string  `json:"type_name" binding:"required"` // DEPOSIT, WITHDRAW, REFUND, etc.
	Amount      float64 `json:"amount"`
	Channel     string  `json:"channel" binding:"required"` // originating service
	Description string  `json:"description"`                // transaction description
	Lang        string  `json:"lang"`                       // response language
}

// TransactionResponse represents a wallet transaction response
type TransactionResponse struct {
	Status  int32       `json:"status"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

// DepositWithdrawResponse represents response for deposit/withdraw operations
type DepositWithdrawResponse struct {
	Username       string  `json:"username"`
	CurrentBalance float64 `json:"current_balance"` // current balance
	BeforeBalance  float64 `json:"before_balance"`  // balance before transaction
	AfterBalance   float64 `json:"after_balance"`   // balance after transaction
}

// BalanceResponse represents balance check response
type BalanceResponse struct {
	Username       string  `json:"username"`
	CurrentBalance float64 `json:"current_balance"` // current balance
}

// CreateMemberRequest represents member creation request
type CreateMemberRequest struct {
	DateTime time.Time `json:"date_time"`
	Username string    `json:"username"`
}




