// Package mongo implements ports.ConversationRepository against MongoDB.
package mongo

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/abialemuel/AI-Proxy-Service/internal/domain"
	"github.com/abialemuel/AI-Proxy-Service/internal/ports"
)

// ConversationRepo persists Conversation aggregates.
type ConversationRepo struct {
	col *mongo.Collection
}

func NewConversationRepo(db *mongo.Database) *ConversationRepo {
	return &ConversationRepo{col: db.Collection("conversations")}
}

// Document is the persistence representation; intentionally separate from
// domain.Conversation to keep mongo concerns out of the core.
type document struct {
	ID        string    `bson:"_id,omitempty"`
	UserID    string    `bson:"user_id"`
	Messages  []msgDoc  `bson:"messages"`
	Summary   string    `bson:"summary,omitempty"`
	CreatedAt time.Time `bson:"created_at"`
	UpdatedAt time.Time `bson:"updated_at"`
}

type msgDoc struct {
	Role  string    `bson:"role"`
	Parts []partDoc `bson:"parts"`
}

type partDoc struct {
	Kind     string `bson:"kind"`
	Text     string `bson:"text,omitempty"`
	ImageURL string `bson:"image_url,omitempty"`
}

func (r *ConversationRepo) GetByUser(ctx context.Context, userID string) (*domain.Conversation, error) {
	var d document
	err := r.col.FindOne(ctx, bson.M{"user_id": userID}).Decode(&d)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, err
	}
	return fromDoc(d), nil
}

func (r *ConversationRepo) Save(ctx context.Context, c *domain.Conversation) error {
	d := toDoc(c)
	opts := options.Replace().SetUpsert(true)
	_, err := r.col.ReplaceOne(ctx, bson.M{"user_id": c.UserID}, d, opts)
	return err
}

func (r *ConversationRepo) Delete(ctx context.Context, userID string) error {
	_, err := r.col.DeleteOne(ctx, bson.M{"user_id": userID})
	return err
}

func toDoc(c *domain.Conversation) document {
	d := document{
		UserID:    c.UserID,
		Summary:   c.Summary,
		CreatedAt: c.CreatedAt,
		UpdatedAt: c.UpdatedAt,
	}
	for _, m := range c.Messages {
		md := msgDoc{Role: string(m.Role)}
		for _, p := range m.Parts {
			md.Parts = append(md.Parts, partDoc{
				Kind: string(p.Kind), Text: p.Text, ImageURL: p.ImageURL,
			})
		}
		d.Messages = append(d.Messages, md)
	}
	return d
}

func fromDoc(d document) *domain.Conversation {
	c := &domain.Conversation{
		ID:        d.ID,
		UserID:    d.UserID,
		Summary:   d.Summary,
		CreatedAt: d.CreatedAt,
		UpdatedAt: d.UpdatedAt,
	}
	for _, md := range d.Messages {
		m := domain.Message{Role: domain.Role(md.Role)}
		for _, p := range md.Parts {
			m.Parts = append(m.Parts, domain.ContentPart{
				Kind: domain.PartKind(p.Kind), Text: p.Text, ImageURL: p.ImageURL,
			})
		}
		c.Messages = append(c.Messages, m)
	}
	return c
}

var _ ports.ConversationRepository = (*ConversationRepo)(nil)
