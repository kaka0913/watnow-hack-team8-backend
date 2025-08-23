package test

import (
	"context"
	"log"
	"os"
	"testing"

	"Team8-App/internal/infrastructure/firestore"
	"Team8-App/internal/repository"
)

func TestFirestoreRouteProposalRepository(t *testing.T) {
	// 環境変数の確認
	projectID := os.Getenv("FIRESTORE_PROJECT_ID")
	if projectID == "" {
		projectID = "befree-468615" // デフォルト値を設定
	}

	credentialsPath := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
	log.Printf("🔧 テスト設定:")
	log.Printf("   FIRESTORE_PROJECT_ID: %s", projectID)
	log.Printf("   GOOGLE_APPLICATION_CREDENTIALS: %s", credentialsPath)

	// Firestoreクライアントの初期化
	ctx := context.Background()
	client, err := firestore.NewFirestoreClient(ctx, projectID)
	if err != nil {
		t.Fatalf("Firestoreクライアントの初期化に失敗: %v", err)
	}
	defer client.Close()

	log.Println("✅ Firestoreクライアント初期化成功")

	// ルート提案リポジトリのテスト
	repo := repository.NewFirestoreRouteProposalRepository(client.GetClient())

	// 全ルート提案の取得テスト
	proposals, err := repo.GetAllRouteProposals(ctx)
	if err != nil {
		t.Fatalf("ルート提案の取得に失敗: %v", err)
	}

	log.Printf("📋 取得されたルート提案数: %d", len(proposals))
	
	if len(proposals) == 0 {
		log.Println("⚠️  ルート提案データが見つかりませんでした")
		log.Println("💡 データがない場合は、まず POST /routes/proposals でルート提案を作成してください")
	} else {
		log.Println("✅ ルート提案データの取得成功")
		for i, proposal := range proposals {
			if i >= 3 { // 最初の3件のみ表示
				log.Printf("   ... 他 %d 件", len(proposals)-3)
				break
			}
			log.Printf("   - [%d] ID: %s, タイトル: %s", i+1, proposal.ProposalID, proposal.Title)
		}
	}

	log.Println("✅ FirestoreRouteProposalRepositoryテスト完了")
}
