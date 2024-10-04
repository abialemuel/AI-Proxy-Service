package repository

import (
	"context"
	"time"

	"github.com/abialemuel/poly-kit/infrastructure/apm"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// MongoDBRepository The implementation of user.Repository object
type MongoDBRepository struct {
	db *mongo.Database
}

// NewPgDBRepository Generate pg user repository
func NewMongoDBRepository(db *mongo.Database) *MongoDBRepository {
	repo := MongoDBRepository{db: db}

	return &repo
}

// UpsertConversation inserts new messages into the messages collection and updates the conversation
func (r *MongoDBRepository) UpsertConversation(ctx context.Context, userID string, messages []Message, summary *Summary) error {
	ctx, span := apm.StartTransaction(ctx, "Repository::UpsertConversation")
	defer apm.EndTransaction(span)

	conversationsCollection := r.db.Collection("conversations")
	messagesCollection := r.db.Collection("messages")

	// Create filter to find the last conversation by UserID
	filter := bson.M{"user_id": userID}

	// Find options to sort by updated_at in descending order and limit to 1
	findOptions := options.FindOne().SetSort(bson.D{{"updated_at", -1}})

	var conversation Conversation
	err := conversationsCollection.FindOne(ctx, filter, findOptions).Decode(&conversation)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			// No existing conversation found, create a new one
			conversation = Conversation{
				ID:        primitive.NewObjectID(),
				UserID:    userID,
				Summaries: []Summary{},
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}
			_, err = conversationsCollection.InsertOne(ctx, conversation)
			if err != nil {
				return err
			}
		} else {
			return err
		}
	}

	// Prepare messages for bulk insert
	var messageDocs []interface{}
	for _, message := range messages {
		message.ConversationID = conversation.ID
		message.Timestamp = time.Now()
		messageDocs = append(messageDocs, message)
	}

	// Insert new messages into the messages collection
	if len(messageDocs) > 0 {
		_, err = messagesCollection.InsertMany(ctx, messageDocs)
		if err != nil {
			return err
		}
	}

	// Update the conversation's updated_at timestamp
	update := bson.M{
		"$set": bson.M{"updated_at": time.Now()},
	}

	if len(summary.Content) > 0 {
		// Add the new summary to the conversation
		update["$push"] = bson.M{"summaries": summary}
	}

	// Perform the update operation on the conversation
	_, err = conversationsCollection.UpdateOne(ctx, bson.M{"_id": conversation.ID}, update)
	return err
}
