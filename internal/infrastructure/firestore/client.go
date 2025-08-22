package firestore

import (
	"context"
	"log"
	"os"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/option"
)

type FirestoreClient struct {
	client *firestore.Client
}

func NewFirestoreClient(ctx context.Context, projectID string) (*FirestoreClient, error) {
	// サービスアカウントキーファイルのパスを設定
	credentialsFile := "befree-firestore-key.json"
	
	// ファイルが存在するかチェック
	if _, err := os.Stat(credentialsFile); os.IsNotExist(err) {
		log.Printf("⚠️  Credentials file not found: %s, trying with default authentication", credentialsFile)
		// デフォルトの認証方法を試す
		client, err := firestore.NewClient(ctx, projectID)
		if err != nil {
			return nil, err
		}
		log.Printf("✅ Firestore client initialized for project: %s (default auth)", projectID)
		return &FirestoreClient{
			client: client,
		}, nil
	}

	// サービスアカウントキーファイルを使用
	client, err := firestore.NewClient(ctx, projectID, option.WithCredentialsFile(credentialsFile))
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
