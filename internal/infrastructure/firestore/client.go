package firestore

import (
	"context"
	"fmt"
	"log"
	"os"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/option"
)

type FirestoreClient struct {
	client *firestore.Client
}

func NewFirestoreClient(ctx context.Context, projectID string) (*FirestoreClient, error) {
	var client *firestore.Client
	var err error

	// Cloud Run環境の検出
	isCloudRun := os.Getenv("K_SERVICE") != "" || os.Getenv("PORT") != ""

	if isCloudRun {
		// Cloud Run環境ではデフォルト認証を使用
		log.Printf("☁️ Cloud Run環境: デフォルト認証を使用")
		client, err = firestore.NewClient(ctx, projectID)
		if err != nil {
			return nil, fmt.Errorf("failed to create Firestore client with default auth: %w", err)
		}
		log.Printf("✅ Firestore client initialized for project: %s (Cloud Run default auth)", projectID)
	} else {
		// ローカル環境では環境変数またはファイルから認証
		credentialsFile := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")

		if credentialsFile == "" {
			credentialsFile = "befree-firestore-key.json"
		}

		if _, fileErr := os.Stat(credentialsFile); fileErr != nil {
			log.Printf("⚠️ Credentials file not found: %s, trying with default authentication", credentialsFile)
			client, err = firestore.NewClient(ctx, projectID)
		} else {
			log.Printf("📄 Using credentials file: %s", credentialsFile)
			option := option.WithCredentialsFile(credentialsFile)
			client, err = firestore.NewClient(ctx, projectID, option)
		}

		if err != nil {
			return nil, fmt.Errorf("failed to create Firestore client: %w", err)
		}
		log.Printf("✅ Firestore client initialized for project: %s", projectID)
	}

	return &FirestoreClient{client: client}, nil
}

func (fc *FirestoreClient) Close() error {
	return fc.client.Close()
}

func (fc *FirestoreClient) GetClient() *firestore.Client {
	return fc.client
}
