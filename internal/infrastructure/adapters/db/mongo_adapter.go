package db

import (
	"context"
	"fmt"
	"log"
	"time"

	"hexagonal-queue/internal/application/ports"
	"hexagonal-queue/internal/domain/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// MongoAdapter implements the WalletRepository interface
type MongoAdapter struct {
	client   *mongo.Client
	database *mongo.Database
	config   MongoConfig
}

// MongoConfig holds MongoDB configuration
type MongoConfig struct {
	URI                  string
	Database             string
	UsersCollection      string
	BalancesCollection   string
	StatementsCollection string
	ConnectionTimeout    time.Duration
	QueryTimeout         time.Duration
	MaxStatementHistory  int
}

// MongoTransactionContext implements the TransactionContext interface
type MongoTransactionContext struct {
	session mongo.Session
	ctx     context.Context
}

// NewMongoAdapter creates a new MongoDB adapter
func NewMongoAdapter(uri, database string) (*MongoAdapter, error) {
	config := MongoConfig{
		URI:                  uri,
		Database:             database,
		UsersCollection:      "users",
		BalancesCollection:   "account_balances",
		StatementsCollection: "wallet_statements",
		ConnectionTimeout:    10 * time.Second,
		QueryTimeout:         30 * time.Second,
		MaxStatementHistory:  1000, // Keep last 1000 statements per user
	}

	// Create MongoDB client
	ctx, cancel := context.WithTimeout(context.Background(), config.ConnectionTimeout)
	defer cancel()

	clientOptions := options.Client().ApplyURI(config.URI)
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	// Test the connection
	if err := client.Ping(ctx, nil); err != nil {
		return nil, fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	db := client.Database(config.Database)

	adapter := &MongoAdapter{
		client:   client,
		database: db,
		config:   config,
	}

	// Create indexes
	if err := adapter.createIndexes(); err != nil {
		log.Printf("Warning: failed to create indexes: %v", err)
	}

	return adapter, nil
}

// createIndexes creates necessary indexes for optimal performance
func (m *MongoAdapter) createIndexes() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Users collection indexes
	usersCollection := m.database.Collection(m.config.UsersCollection)
	usersIndexes := []mongo.IndexModel{
		{Keys: bson.D{{Key: "username", Value: 1}, {Key: "parent_token", Value: 1}}, Options: options.Index().SetUnique(true)},
		{Keys: bson.D{{Key: "token", Value: 1}}, Options: options.Index().SetUnique(true)},
		{Keys: bson.D{{Key: "parent_token", Value: 1}}},
	}

	if _, err := usersCollection.Indexes().CreateMany(ctx, usersIndexes); err != nil {
		return fmt.Errorf("failed to create users indexes: %w", err)
	}

	// Account balances collection indexes
	balancesCollection := m.database.Collection(m.config.BalancesCollection)
	balancesIndexes := []mongo.IndexModel{
		{Keys: bson.D{{Key: "username", Value: 1}, {Key: "parent_token", Value: 1}}, Options: options.Index().SetUnique(true)},
		{Keys: bson.D{{Key: "parent_token", Value: 1}}},
		{Keys: bson.D{{Key: "updated_at", Value: -1}}},
	}

	if _, err := balancesCollection.Indexes().CreateMany(ctx, balancesIndexes); err != nil {
		return fmt.Errorf("failed to create balances indexes: %w", err)
	}

	// Wallet statements collection indexes
	statementsCollection := m.database.Collection(m.config.StatementsCollection)
	statementsIndexes := []mongo.IndexModel{
		{Keys: bson.D{{Key: "username", Value: 1}, {Key: "parent_token", Value: 1}, {Key: "datetime", Value: -1}}},
		{Keys: bson.D{{Key: "parent_token", Value: 1}, {Key: "datetime", Value: -1}}},
		{Keys: bson.D{{Key: "datetime", Value: -1}}},
		{Keys: bson.D{{Key: "channel", Value: 1}, {Key: "datetime", Value: -1}}},
	}

	if _, err := statementsCollection.Indexes().CreateMany(ctx, statementsIndexes); err != nil {
		return fmt.Errorf("failed to create statements indexes: %w", err)
	}

	return nil
}

// User operations

// GetUserByUsername retrieves a user by username and parent token
func (m *MongoAdapter) GetUserByUsername(ctx context.Context, username, parentToken string) (*models.User, error) {
	collection := m.database.Collection(m.config.UsersCollection)

	filter := bson.M{
		"username":     username,
		"parent_token": parentToken,
		"status":       1,
	}

	var user models.User
	err := collection.FindOne(ctx, filter).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get user by username: %w", err)
	}

	return &user, nil
}

// GetUserByToken retrieves a user by token
func (m *MongoAdapter) GetUserByToken(ctx context.Context, token string) (*models.User, error) {
	collection := m.database.Collection(m.config.UsersCollection)

	filter := bson.M{
		"token":  token,
		"status": 1,
	}

	var user models.User
	err := collection.FindOne(ctx, filter).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get user by token: %w", err)
	}

	return &user, nil
}

// CreateUser creates a new user
func (m *MongoAdapter) CreateUser(ctx context.Context, user *models.User) error {
	collection := m.database.Collection(m.config.UsersCollection)

	if user.ID.IsZero() {
		user.ID = primitive.NewObjectID()
	}

	user.CreatedAt = time.Now().UTC()
	user.UpdatedAt = time.Now().UTC()

	_, err := collection.InsertOne(ctx, user)
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}

// UpdateUser updates an existing user
func (m *MongoAdapter) UpdateUser(ctx context.Context, user *models.User) error {
	collection := m.database.Collection(m.config.UsersCollection)

	user.UpdatedAt = time.Now().UTC()

	filter := bson.M{"_id": user.ID}
	update := bson.M{"$set": user}

	result, err := collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("user not found")
	}

	return nil
}

// Account balance operations

// GetAccountBalance retrieves account balance by username and parent token
func (m *MongoAdapter) GetAccountBalance(ctx context.Context, username, parentToken string) (*models.AccountBalance, error) {
	collection := m.database.Collection(m.config.BalancesCollection)

	filter := bson.M{
		"username":     username,
		"parent_token": parentToken,
		"status":       1,
	}

	var balance models.AccountBalance
	err := collection.FindOne(ctx, filter).Decode(&balance)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("account balance not found")
		}
		return nil, fmt.Errorf("failed to get account balance: %w", err)
	}

	return &balance, nil
}

// GetAccountBalancesByParentToken retrieves multiple account balances
func (m *MongoAdapter) GetAccountBalancesByParentToken(ctx context.Context, usernames []string, parentToken string) ([]*models.AccountBalance, error) {
	collection := m.database.Collection(m.config.BalancesCollection)

	filter := bson.M{
		"username":     bson.M{"$in": usernames},
		"parent_token": parentToken,
		"status":       1,
	}

	cursor, err := collection.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to get account balances: %w", err)
	}
	defer cursor.Close(ctx)

	var balances []*models.AccountBalance
	for cursor.Next(ctx) {
		var balance models.AccountBalance
		if err := cursor.Decode(&balance); err != nil {
			return nil, fmt.Errorf("failed to decode balance: %w", err)
		}
		balances = append(balances, &balance)
	}

	return balances, nil
}

// CreateAccountBalance creates a new account balance
func (m *MongoAdapter) CreateAccountBalance(ctx context.Context, balance *models.AccountBalance) error {
	collection := m.database.Collection(m.config.BalancesCollection)

	if balance.ID.IsZero() {
		balance.ID = primitive.NewObjectID()
	}

	balance.CreatedAt = time.Now().UTC()
	balance.UpdatedAt = time.Now().UTC()

	_, err := collection.InsertOne(ctx, balance)
	if err != nil {
		return fmt.Errorf("failed to create account balance: %w", err)
	}

	return nil
}

// UpdateAccountBalance updates the account balance
func (m *MongoAdapter) UpdateAccountBalance(ctx context.Context, username, parentToken string, newBalance primitive.Decimal128) error {
	collection := m.database.Collection(m.config.BalancesCollection)

	filter := bson.M{
		"username":     username,
		"parent_token": parentToken,
	}

	update := bson.M{
		"$set": bson.M{
			"balance":    newBalance,
			"updated_at": time.Now().UTC(),
		},
	}

	result, err := collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update account balance: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("account balance not found")
	}

	return nil
}

// Wallet statement operations

// GetLastWalletStatement retrieves the most recent wallet statement
func (m *MongoAdapter) GetLastWalletStatement(ctx context.Context, username, parentToken string) (*models.WalletStatement, error) {
	collection := m.database.Collection(m.config.StatementsCollection)

	filter := bson.M{
		"username":     username,
		"parent_token": parentToken,
	}

	opts := options.FindOne().SetSort(bson.D{{Key: "datetime", Value: -1}})

	var statement models.WalletStatement
	err := collection.FindOne(ctx, filter, opts).Decode(&statement)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			// If no statement exists, create a zero balance statement
			return &models.WalletStatement{
				AfterCredit: primitive.NewDecimal128(0, 0),
			}, nil
		}
		return nil, fmt.Errorf("failed to get last wallet statement: %w", err)
	}

	return &statement, nil
}

// CreateWalletStatement creates a new wallet statement
func (m *MongoAdapter) CreateWalletStatement(ctx context.Context, statement *models.WalletStatement) (*models.WalletStatement, error) {
	collection := m.database.Collection(m.config.StatementsCollection)

	if statement.ID.IsZero() {
		statement.ID = primitive.NewObjectID()
	}

	if statement.Datetime.IsZero() {
		statement.Datetime = time.Now().UTC()
	}

	_, err := collection.InsertOne(ctx, statement)
	if err != nil {
		return nil, fmt.Errorf("failed to create wallet statement: %w", err)
	}

	return statement, nil
}

// GetWalletStatements retrieves wallet statements with limit
func (m *MongoAdapter) GetWalletStatements(ctx context.Context, username, parentToken string, limit int) ([]*models.WalletStatement, error) {
	collection := m.database.Collection(m.config.StatementsCollection)

	filter := bson.M{
		"username":     username,
		"parent_token": parentToken,
	}

	opts := options.Find().
		SetSort(bson.D{{Key: "datetime", Value: -1}}).
		SetLimit(int64(limit))

	cursor, err := collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to get wallet statements: %w", err)
	}
	defer cursor.Close(ctx)

	var statements []*models.WalletStatement
	for cursor.Next(ctx) {
		var statement models.WalletStatement
		if err := cursor.Decode(&statement); err != nil {
			return nil, fmt.Errorf("failed to decode statement: %w", err)
		}
		statements = append(statements, &statement)
	}

	return statements, nil
}

// ClearOldStatements removes old wallet statements to maintain history limit
func (m *MongoAdapter) ClearOldStatements(ctx context.Context, username, parentToken string, keepCount int) error {
	collection := m.database.Collection(m.config.StatementsCollection)

	// Find statements to keep (most recent ones)
	filter := bson.M{
		"username":     username,
		"parent_token": parentToken,
	}

	opts := options.Find().
		SetSort(bson.D{{Key: "datetime", Value: -1}}).
		SetLimit(int64(keepCount)).
		SetProjection(bson.M{"_id": 1})

	cursor, err := collection.Find(ctx, filter, opts)
	if err != nil {
		return fmt.Errorf("failed to find statements to keep: %w", err)
	}
	defer cursor.Close(ctx)

	var keepIDs []primitive.ObjectID
	for cursor.Next(ctx) {
		var doc struct {
			ID primitive.ObjectID `bson:"_id"`
		}
		if err := cursor.Decode(&doc); err != nil {
			continue
		}
		keepIDs = append(keepIDs, doc.ID)
	}

	if len(keepIDs) == 0 {
		return nil // No statements to process
	}

	// Delete old statements
	deleteFilter := bson.M{
		"username":     username,
		"parent_token": parentToken,
		"_id":          bson.M{"$nin": keepIDs},
	}

	result, err := collection.DeleteMany(ctx, deleteFilter)
	if err != nil {
		return fmt.Errorf("failed to delete old statements: %w", err)
	}

	log.Printf("Cleared %d old statements for user %s", result.DeletedCount, username)
	return nil
}

// Transaction operations

// StartTransaction starts a MongoDB transaction
func (m *MongoAdapter) StartTransaction(ctx context.Context) (ports.TransactionContext, error) {
	session, err := m.client.StartSession()
	if err != nil {
		return nil, fmt.Errorf("failed to start session: %w", err)
	}

	if err := session.StartTransaction(); err != nil {
		session.EndSession(ctx)
		return nil, fmt.Errorf("failed to start transaction: %w", err)
	}

	sessionCtx := mongo.NewSessionContext(ctx, session)

	return &MongoTransactionContext{
		session: session,
		ctx:     sessionCtx,
	}, nil
}

// Commit commits the transaction
func (mtc *MongoTransactionContext) Commit(ctx context.Context) error {
	defer mtc.session.EndSession(ctx)
	return mtc.session.CommitTransaction(ctx)
}

// Abort aborts the transaction
func (mtc *MongoTransactionContext) Abort(ctx context.Context) error {
	defer mtc.session.EndSession(ctx)
	return mtc.session.AbortTransaction(ctx)
}

// Context returns the session context
func (mtc *MongoTransactionContext) Context() context.Context {
	return mtc.ctx
}

// Close closes the MongoDB connection
func (m *MongoAdapter) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	return m.client.Disconnect(ctx)
}

// HealthCheck checks the MongoDB connection health
func (m *MongoAdapter) HealthCheck(ctx context.Context) error {
	return m.client.Ping(ctx, nil)
}
