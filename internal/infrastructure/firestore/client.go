package firestore

import (
	"context"
	"log"

	"cloud.google.com/go/firestore"
)

type FirestoreClient struct {
	client *firestore.Client
}

func NewFirestoreClient(ctx context.Context, projectID string) (*FirestoreClient, error) {
	client, err := firestore.NewClient(ctx, projectID)
	if err != nil {
		return nil, err
	}

	log.Printf("✅ Firestore client initialized for project: %s", projectID)
	return &FirestoreClient{
		client: client,
	}, nil
}

func (fc *FirestoreClient) Close() error {
	return fc.client.Close()
}

func (fc *FirestoreClient) GetClient() *firestore.Client {
	return fc.client
}
