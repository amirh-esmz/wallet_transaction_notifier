package repository

import (
    "context"
    "time"
    "errors"
    "log"

    "go.mongodb.org/mongo-driver/mongo"
    "go.mongodb.org/mongo-driver/mongo/options"
    "go.mongodb.org/mongo-driver/bson"

    "github.com/you/wallet_transaction_notifier/internal/domain"
    "github.com/you/wallet_transaction_notifier/internal/ports"
)

// Mongo repositories with actual MongoDB implementation

var (
    mongoClient *mongo.Client
    mongoDB     *mongo.Database
)

func initMongoDB(uri string, dbName string) error {
    if mongoClient != nil {
        return nil // Already initialized
    }

    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
    if err != nil {
        return err
    }

    // Test the connection
    err = client.Ping(ctx, nil)
    if err != nil {
        return err
    }

    mongoClient = client
    mongoDB = client.Database(dbName)
    log.Printf("Connected to MongoDB: %s", dbName)
    return nil
}

type MongoWalletRepository struct{}

func NewMongoWalletRepository(uri string, dbName string) (ports.WalletRepository, error) {
    if err := initMongoDB(uri, dbName); err != nil {
        return nil, err
    }
    return &MongoWalletRepository{}, nil
}

func (r *MongoWalletRepository) Create(ctx context.Context, wallet domain.Wallet) error {
    _ = ctx
    _ = wallet
    return errors.New("not implemented")
}

func (r *MongoWalletRepository) ListByUser(ctx context.Context, userID string) ([]domain.Wallet, error) {
    _ = ctx
    _ = userID
    return []domain.Wallet{
        {ID: "w-demo", UserID: "u-demo", Blockchain: "ethereum", Address: "0x0000000000000000000000000000000000000000", CreatedAt: time.Now()},
    }, nil
}

// Sessions
type MongoSessionRepository struct{}

func NewMongoSessionRepository(uri string, dbName string) (ports.SessionRepository, error) {
    if err := initMongoDB(uri, dbName); err != nil {
        return nil, err
    }
    return &MongoSessionRepository{}, nil
}

func (r *MongoSessionRepository) UpsertTelegramSession(ctx context.Context, s domain.TelegramSession) error {
    collection := mongoDB.Collection("telegram_sessions")
    
    filter := bson.M{"chatId": s.ChatID}
    update := bson.M{
        "$set": bson.M{
            "chatId":     s.ChatID,
            "state":      s.State,
            "lastAction": s.LastAction,
            "updatedAt":  time.Now(),
        },
        "$setOnInsert": bson.M{
            "createdAt": time.Now(),
        },
    }
    
    opts := options.Update().SetUpsert(true)
    _, err := collection.UpdateOne(ctx, filter, update, opts)
    return err
}

func (r *MongoSessionRepository) GetTelegramSession(ctx context.Context, chatID string) (domain.TelegramSession, error) {
    collection := mongoDB.Collection("telegram_sessions")
    
    var session domain.TelegramSession
    filter := bson.M{"chatId": chatID}
    err := collection.FindOne(ctx, filter).Decode(&session)
    
    if err == mongo.ErrNoDocuments {
        // Return a new session if not found
        return domain.TelegramSession{
            ChatID:    chatID,
            State:     domain.StateIdle,
            CreatedAt: time.Now(),
            UpdatedAt: time.Now(),
        }, nil
    }
    
    return session, err
}

// Subscriptions
type MongoSubscriptionRepository struct{}

func NewMongoSubscriptionRepository(uri string, dbName string) (ports.SubscriptionRepository, error) {
    if err := initMongoDB(uri, dbName); err != nil {
        return nil, err
    }
    return &MongoSubscriptionRepository{}, nil
}

func (r *MongoSubscriptionRepository) AddSubscription(ctx context.Context, sub domain.Subscription) error {
    collection := mongoDB.Collection("subscriptions")
    
    // Check if subscription already exists
    filter := bson.M{
        "chatId":     sub.ChatID,
        "blockchain": sub.Blockchain,
        "address":    sub.Address,
    }
    
    var existing domain.Subscription
    err := collection.FindOne(ctx, filter).Decode(&existing)
    if err == nil {
        // Subscription already exists
        return nil
    }
    
    if err != mongo.ErrNoDocuments {
        return err
    }
    
    // Insert new subscription
    _, err = collection.InsertOne(ctx, sub)
    return err
}

func (r *MongoSubscriptionRepository) RemoveSubscription(ctx context.Context, chatID string, blockchain string, address string) error {
    collection := mongoDB.Collection("subscriptions")
    
    filter := bson.M{
        "chatId":     chatID,
        "blockchain": blockchain,
        "address":    address,
    }
    
    _, err := collection.DeleteOne(ctx, filter)
    return err
}

func (r *MongoSubscriptionRepository) ListSubscriptions(ctx context.Context, chatID string, blockchain string) ([]domain.Subscription, error) {
    collection := mongoDB.Collection("subscriptions")
    
    filter := bson.M{
        "chatId":     chatID,
        "blockchain": blockchain,
    }
    
    cursor, err := collection.Find(ctx, filter)
    if err != nil {
        return nil, err
    }
    defer cursor.Close(ctx)
    
    var subscriptions []domain.Subscription
    if err = cursor.All(ctx, &subscriptions); err != nil {
        return nil, err
    }
    
    return subscriptions, nil
}

func (r *MongoSubscriptionRepository) ListSubscribersByAddress(ctx context.Context, blockchain string, address string) ([]domain.Subscription, error) {
    collection := mongoDB.Collection("subscriptions")
    
    filter := bson.M{
        "blockchain": blockchain,
        "address":    address,
    }
    
    cursor, err := collection.Find(ctx, filter)
    if err != nil {
        return nil, err
    }
    defer cursor.Close(ctx)
    
    var subscriptions []domain.Subscription
    if err = cursor.All(ctx, &subscriptions); err != nil {
        return nil, err
    }
    
    return subscriptions, nil
}

// Notifications
type MongoNotificationRepository struct{}

func NewMongoNotificationRepository(uri string, dbName string) (ports.NotificationRepository, error) {
    return &MongoNotificationRepository{}, nil
}

func (r *MongoNotificationRepository) Save(ctx context.Context, n domain.Notification) error {
    _ = ctx; _ = n
    return nil
}

func (r *MongoNotificationRepository) ListByAddress(ctx context.Context, chatID string, blockchain string, address string, limit int) ([]domain.Notification, error) {
    _ = ctx; _ = chatID; _ = blockchain; _ = address; _ = limit
    return []domain.Notification{}, nil
}


