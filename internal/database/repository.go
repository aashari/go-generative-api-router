package database

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Repository provides database operations for the application
type Repository struct {
	conn *Connection
	db   *mongo.Database
}

// NewRepository creates a new repository instance
func NewRepository() (*Repository, error) {
	conn, err := GetConnection()
	if err != nil {
		return nil, fmt.Errorf("failed to get database connection: %w", err)
	}

	return &Repository{
		conn: conn,
		db:   conn.Database,
	}, nil
}

// RequestLogRepository provides operations for request logs
type RequestLogRepository struct {
	collection *mongo.Collection
}

// GetRequestLogRepository returns a repository for request logs
func (r *Repository) GetRequestLogRepository() *RequestLogRepository {
	return &RequestLogRepository{
		collection: r.conn.GetCollection("request-logs"),
	}
}

// InsertRequestLog inserts a new request log
func (rlr *RequestLogRepository) InsertRequestLog(ctx context.Context, log *RequestLog) error {
	log.CreatedAt = time.Now()
	log.UpdatedAt = time.Now()

	_, err := rlr.collection.InsertOne(ctx, log)
	if err != nil {
		return fmt.Errorf("failed to insert request log: %w", err)
	}

	return nil
}

// GetRequestLogByID retrieves a request log by ID
func (rlr *RequestLogRepository) GetRequestLogByID(ctx context.Context, id string) (*RequestLog, error) {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, fmt.Errorf("invalid object ID: %w", err)
	}

	var log RequestLog
	err = rlr.collection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&log)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get request log: %w", err)
	}

	return &log, nil
}

// GetRequestLogByRequestID retrieves a request log by request ID
func (rlr *RequestLogRepository) GetRequestLogByRequestID(ctx context.Context, requestID string) (*RequestLog, error) {
	var log RequestLog
	err := rlr.collection.FindOne(ctx, bson.M{"request_id": requestID}).Decode(&log)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get request log by request ID: %w", err)
	}

	return &log, nil
}

// GetRecentRequestLogs retrieves recent request logs with pagination
func (rlr *RequestLogRepository) GetRecentRequestLogs(ctx context.Context, limit int64, offset int64) ([]*RequestLog, error) {
	opts := options.Find().
		SetSort(bson.D{{Key: "timestamp", Value: -1}}).
		SetLimit(limit).
		SetSkip(offset)

	cursor, err := rlr.collection.Find(ctx, bson.M{}, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to find request logs: %w", err)
	}
	defer cursor.Close(ctx)

	var logs []*RequestLog
	for cursor.Next(ctx) {
		var log RequestLog
		if err := cursor.Decode(&log); err != nil {
			return nil, fmt.Errorf("failed to decode request log: %w", err)
		}
		logs = append(logs, &log)
	}

	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("cursor error: %w", err)
	}

	return logs, nil
}

// GetRequestLogsByVendor retrieves request logs for a specific vendor
func (rlr *RequestLogRepository) GetRequestLogsByVendor(ctx context.Context, vendor string, limit int64) ([]*RequestLog, error) {
	opts := options.Find().
		SetSort(bson.D{{Key: "timestamp", Value: -1}}).
		SetLimit(limit)

	cursor, err := rlr.collection.Find(ctx, bson.M{"selected_vendor": vendor}, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to find request logs by vendor: %w", err)
	}
	defer cursor.Close(ctx)

	var logs []*RequestLog
	for cursor.Next(ctx) {
		var log RequestLog
		if err := cursor.Decode(&log); err != nil {
			return nil, fmt.Errorf("failed to decode request log: %w", err)
		}
		logs = append(logs, &log)
	}

	return logs, nil
}

// GetRequestLogsByTimeRange retrieves request logs within a time range
func (rlr *RequestLogRepository) GetRequestLogsByTimeRange(ctx context.Context, start, end time.Time, limit int64) ([]*RequestLog, error) {
	filter := bson.M{
		"timestamp": bson.M{
			"$gte": start,
			"$lte": end,
		},
	}

	opts := options.Find().
		SetSort(bson.D{{Key: "timestamp", Value: -1}}).
		SetLimit(limit)

	cursor, err := rlr.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to find request logs by time range: %w", err)
	}
	defer cursor.Close(ctx)

	var logs []*RequestLog
	for cursor.Next(ctx) {
		var log RequestLog
		if err := cursor.Decode(&log); err != nil {
			return nil, fmt.Errorf("failed to decode request log: %w", err)
		}
		logs = append(logs, &log)
	}

	return logs, nil
}

// VendorMetricsRepository provides operations for vendor metrics
type VendorMetricsRepository struct {
	collection *mongo.Collection
}

// GetVendorMetricsRepository returns a repository for vendor metrics
func (r *Repository) GetVendorMetricsRepository() *VendorMetricsRepository {
	return &VendorMetricsRepository{
		collection: r.conn.GetCollection("vendor_metrics"),
	}
}

// InsertVendorMetrics inserts new vendor metrics
func (vmr *VendorMetricsRepository) InsertVendorMetrics(ctx context.Context, metrics *VendorMetrics) error {
	metrics.CreatedAt = time.Now()
	metrics.UpdatedAt = time.Now()

	_, err := vmr.collection.InsertOne(ctx, metrics)
	if err != nil {
		return fmt.Errorf("failed to insert vendor metrics: %w", err)
	}

	return nil
}

// GetVendorMetricsByPeriod retrieves vendor metrics for a specific period
func (vmr *VendorMetricsRepository) GetVendorMetricsByPeriod(ctx context.Context, vendor, model string, periodType string, start, end time.Time) ([]*VendorMetrics, error) {
	filter := bson.M{
		"vendor":       vendor,
		"model":        model,
		"period_type":  periodType,
		"period_start": bson.M{"$gte": start},
		"period_end":   bson.M{"$lte": end},
	}

	opts := options.Find().SetSort(bson.D{{Key: "period_start", Value: -1}})

	cursor, err := vmr.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to find vendor metrics: %w", err)
	}
	defer cursor.Close(ctx)

	var metrics []*VendorMetrics
	for cursor.Next(ctx) {
		var metric VendorMetrics
		if err := cursor.Decode(&metric); err != nil {
			return nil, fmt.Errorf("failed to decode vendor metrics: %w", err)
		}
		metrics = append(metrics, &metric)
	}

	return metrics, nil
}

// SystemHealthRepository provides operations for system health
type SystemHealthRepository struct {
	collection *mongo.Collection
}

// GetSystemHealthRepository returns a repository for system health
func (r *Repository) GetSystemHealthRepository() *SystemHealthRepository {
	return &SystemHealthRepository{
		collection: r.conn.GetCollection("system_health"),
	}
}

// InsertSystemHealth inserts new system health data
func (shr *SystemHealthRepository) InsertSystemHealth(ctx context.Context, health *SystemHealth) error {
	health.CreatedAt = time.Now()

	_, err := shr.collection.InsertOne(ctx, health)
	if err != nil {
		return fmt.Errorf("failed to insert system health: %w", err)
	}

	return nil
}

// GetLatestSystemHealth retrieves the latest system health data
func (shr *SystemHealthRepository) GetLatestSystemHealth(ctx context.Context, environment string) (*SystemHealth, error) {
	filter := bson.M{"environment": environment}
	opts := options.FindOne().SetSort(bson.D{{Key: "timestamp", Value: -1}})

	var health SystemHealth
	err := shr.collection.FindOne(ctx, filter, opts).Decode(&health)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get latest system health: %w", err)
	}

	return &health, nil
}

// UserSessionRepository provides operations for user sessions
type UserSessionRepository struct {
	collection *mongo.Collection
}

// GetUserSessionRepository returns a repository for user sessions
func (r *Repository) GetUserSessionRepository() *UserSessionRepository {
	return &UserSessionRepository{
		collection: r.conn.GetCollection("user_sessions"),
	}
}

// UpsertUserSession creates or updates a user session
func (usr *UserSessionRepository) UpsertUserSession(ctx context.Context, session *UserSession) error {
	filter := bson.M{"session_id": session.SessionID}

	update := bson.M{
		"$set": bson.M{
			"user_agent":  session.UserAgent,
			"client_ip":   session.ClientIP,
			"environment": session.Environment,
			"last_seen":   time.Now(),
			"updated_at":  time.Now(),
		},
		"$inc": bson.M{
			"request_count": 1,
		},
		"$addToSet": bson.M{
			"models_used":  bson.M{"$each": session.ModelsUsed},
			"vendors_used": bson.M{"$each": session.VendorsUsed},
		},
		"$setOnInsert": bson.M{
			"first_seen": time.Now(),
			"created_at": time.Now(),
		},
	}

	opts := options.Update().SetUpsert(true)

	_, err := usr.collection.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		return fmt.Errorf("failed to upsert user session: %w", err)
	}

	return nil
}

// GetActiveUserSessions retrieves active user sessions within a time window
func (usr *UserSessionRepository) GetActiveUserSessions(ctx context.Context, since time.Time, limit int64) ([]*UserSession, error) {
	filter := bson.M{
		"last_seen": bson.M{"$gte": since},
	}

	opts := options.Find().
		SetSort(bson.D{{Key: "last_seen", Value: -1}}).
		SetLimit(limit)

	cursor, err := usr.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to find active user sessions: %w", err)
	}
	defer cursor.Close(ctx)

	var sessions []*UserSession
	for cursor.Next(ctx) {
		var session UserSession
		if err := cursor.Decode(&session); err != nil {
			return nil, fmt.Errorf("failed to decode user session: %w", err)
		}
		sessions = append(sessions, &session)
	}

	return sessions, nil
}

// GenerativeVendorLog operations - REMOVED: Database logging functionality has been removed
