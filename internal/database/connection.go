package database

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

// Connection holds the MongoDB connection and configuration
type Connection struct {
	Client   *mongo.Client
	Database *mongo.Database
	Config   *DatabaseConfig
	mu       sync.RWMutex
}

var (
	instance *Connection
	once     sync.Once
)

// GetConnection returns a singleton MongoDB connection
// Creates a new connection if one doesn't exist, using environment variables for configuration
func GetConnection() (*Connection, error) {
	var err error
	once.Do(func() {
		config := GetDatabaseConfig()
		instance, err = newConnection(config)
	})
	return instance, err
}

// newConnection creates a new MongoDB connection
func newConnection(config *DatabaseConfig) (*Connection, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create client options with URI (URI contains all connection details including auth)
	clientOptions := options.Client().ApplyURI(config.URI)

	// Set application name for connection tracking
	if config.AppName != "" {
		clientOptions.SetAppName(config.AppName)
	}

	// Log connection attempt (with masked sensitive data)
	maskedConfig := config.MaskSensitiveData()
	log.Printf("Connecting to MongoDB database: %s (URI: %s)",
		maskedConfig.DatabaseName, maskedConfig.URI)

	// Connect to MongoDB
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	// Ping the database to verify connection
	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		return nil, fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	// Get database instance
	database := client.Database(config.DatabaseName)

	connection := &Connection{
		Client:   client,
		Database: database,
		Config:   config,
	}

	log.Printf("Successfully connected to MongoDB database: %s", config.DatabaseName)

	// Create indexes after successful connection
	if err := connection.createIndexes(ctx); err != nil {
		log.Printf("Warning: Failed to create database indexes: %v", err)
		// Don't fail the connection if index creation fails
		// The application can still function, just with reduced performance
	}

	return connection, nil
}

// Disconnect closes the MongoDB connection
func (c *Connection) Disconnect() error {
	if c.Client == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return c.Client.Disconnect(ctx)
}

// IsConnected checks if the MongoDB connection is active
func (c *Connection) IsConnected() bool {
	if c.Client == nil {
		return false
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	return c.Client.Ping(ctx, readpref.Primary()) == nil
}

// GetCollection returns a MongoDB collection
func (c *Connection) GetCollection(name string) *mongo.Collection {
	return c.Database.Collection(name)
}

// createIndexes creates database indexes for performance optimization
func (c *Connection) createIndexes(ctx context.Context) error {
	// Create indexes for generative-usages collection
	generativeUsagesCollection := c.GetCollection("generative-usages")

	// Index for timestamp-based queries
	timestampIndex := mongo.IndexModel{
		Keys:    bson.D{{Key: "created_at", Value: -1}}, // Descending order for recent-first queries
		Options: options.Index().SetName("created_at_desc").SetBackground(true),
	}

	// Index for vendor-based queries
	vendorIndex := mongo.IndexModel{
		Keys:    bson.D{{Key: "vendor", Value: 1}, {Key: "created_at", Value: -1}},
		Options: options.Index().SetName("vendor_created_at_desc").SetBackground(true),
	}

	// Index for request ID lookups
	requestIdIndex := mongo.IndexModel{
		Keys:    bson.D{{Key: "request_id", Value: 1}},
		Options: options.Index().SetName("request_id").SetBackground(true),
	}

	// Index for requested_at timestamp
	requestedAtIndex := mongo.IndexModel{
		Keys:    bson.D{{Key: "requested_at", Value: -1}},
		Options: options.Index().SetName("requested_at_desc").SetBackground(true),
	}

	// Index for status code queries
	statusCodeIndex := mongo.IndexModel{
		Keys:    bson.D{{Key: "status_code", Value: 1}, {Key: "created_at", Value: -1}},
		Options: options.Index().SetName("status_code_created_at_desc").SetBackground(true),
	}

	indexes := []mongo.IndexModel{timestampIndex, vendorIndex, requestIdIndex, requestedAtIndex, statusCodeIndex}

	_, err := generativeUsagesCollection.Indexes().CreateMany(ctx, indexes)
	if err != nil {
		return fmt.Errorf("failed to create generative-usages indexes: %w", err)
	}

	log.Printf("Successfully created database indexes for collection: generative-usages")
	return nil
}

// HealthCheck performs a health check on the MongoDB connection
func (c *Connection) HealthCheck() error {
	if c.Client == nil {
		return fmt.Errorf("MongoDB client is nil")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := c.Client.Ping(ctx, readpref.Primary()); err != nil {
		return fmt.Errorf("MongoDB ping failed: %w", err)
	}

	return nil
}
