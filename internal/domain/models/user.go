package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// User represents a system user
type User struct {
	ID          primitive.ObjectID `bson:"_id" json:"id,omitempty"`
	Username    string             `bson:"username" json:"username"`
	ParentToken string             `bson:"parent_token" json:"parent_token"`
	Token       string             `bson:"token" json:"token"`
	TypeMember  string             `bson:"type_member" json:"type_member"` // super, agent, member
	Status      int8               `bson:"status" json:"status"`
	CreatedAt   time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt   time.Time          `bson:"updated_at" json:"updated_at"`
}

// ValidationResult represents user validation result
type ValidationResult struct {
	Valid bool   `json:"valid"`
	User  User   `json:"user"`
	Error string `json:"error,omitempty"`
}
