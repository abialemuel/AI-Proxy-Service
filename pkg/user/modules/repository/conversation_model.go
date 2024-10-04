package repository

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Content represents the actual content of a message.
type Content struct {
	Type     string  `bson:"type"`      // Type of content, either text or image
	Text     *string `bson:"text"`      // Text content of the message
	ImageURL string  `bson:"image_url"` // URL of the image, if applicable
}

// Message represents a single message in a conversation.
type Message struct {
	ID             primitive.ObjectID `bson:"_id,omitempty"`   // Unique identifier for the message
	ConversationID primitive.ObjectID `bson:"conversation_id"` // Foreign key to link with a conversation
	Role           string             `bson:"role"`            // Either user, system, or assistant
	Content        []Content          `bson:"content"`         // The actual message content
	Timestamp      time.Time          `bson:"timestamp"`       // Time when the message was sent
}

// Summary represents a summarized form of a message in a conversation (without ID and timestamps).
type Summary struct {
	Role    string    `bson:"role"`    // Role in the conversation (e.g., user, system, assistant)
	Content []Content `bson:"content"` // The summarized content of the conversation
}

// Conversation represents a conversation tied to a user session.
type Conversation struct {
	ID        primitive.ObjectID `bson:"_id,omitempty"` // Unique identifier for the conversation
	UserID    string             `bson:"user_id"`       // ID of the user associated with the conversation
	Summaries []Summary          `bson:"summaries"`     // Array of summaries for the conversation
	CreatedAt time.Time          `bson:"created_at"`    // Timestamp when the conversation was created
	UpdatedAt time.Time          `bson:"updated_at"`    // Timestamp when the conversation was last updated
}
