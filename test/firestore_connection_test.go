package test

import (
	"context"
	"log"
	"os"
	"testing"

	"Team8-App/internal/infrastructure/firestore"
)

func TestFirestoreConnection(t *testing.T) {
	// 環境変数の確認
	projectID := os.Getenv("FIRESTORE_PROJECT_ID")
	if projectID == "" {
		t.Fatal("FIRESTORE_PROJECT_ID environment variable is not set")
	}

	credentialsPath := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
	if credentialsPath == "" {
		t.Fatal("GOOGLE_APPLICATION_CREDENTIALS environment variable is not set")
	}

	log.Printf("🔧 テスト設定:")
	log.Printf("   FIRESTORE_PROJECT_ID: %s", projectID)
	log.Printf("   GOOGLE_APPLICATION_CREDENTIALS: %s", credentialsPath)

	// Firestoreクライアントの初期化テスト
	ctx := context.Background()
	client, err := firestore.NewFirestoreClient(ctx, projectID)
	if err != nil {
		t.Fatalf("Firestoreクライアントの初期化に失敗: %v", err)
	}
	defer client.Close()

	log.Println("✅ Firestoreクライアントの初期化成功")

	// 基本的な読み取りテスト（コレクション一覧取得）
	firestoreClient := client.GetClient()
	collections := firestoreClient.Collections(ctx)
	
	collectionList := []string{}
	for {
		collectionRef, err := collections.Next()
		if err != nil {
			break // イテレータの終了
		}
		collectionList = append(collectionList, collectionRef.ID)
	}

	log.Printf("📚 利用可能なコレクション数: %d", len(collectionList))
	for _, collectionID := range collectionList {
		log.Printf("   - %s", collectionID)
	}

	log.Println("✅ Firestore接続テスト完了")
}
