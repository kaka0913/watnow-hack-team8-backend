# Goコーディング規則

## 概要
このドキュメントは、Goコーディング規則を定めています。Clean Architectureパターンを採用した会議室予約システムの開発において、一貫性のある高品質なコードを維持するためのガイドラインです。

## 目次
1. [命名規則](#命名規則)
2. [パッケージ構成](#パッケージ構成)
3. [インターフェース設計](#インターフェース設計)
4. [エラーハンドリング](#エラーハンドリング)
5. [データベース操作](#データベース操作)
6. [コメント規則](#コメント規則)
7. [テスト](#テスト)
8. [依存性注入](#依存性注入)
9. [並行処理](#並行処理)

## 命名規則

### 基本原則
- パブリックな要素は英語で命名
- プライベートな要素は英語で命名（日本語コメントで補完）
- 略語は大文字で統一（例：`ID`, `URL`, `API`）

### 型とインターフェース
```go
// Good: インターフェースは "-er" または明確な名前
type UserRepository interface {}
type CalendarUsecase interface {}

// Good: 構造体は名詞
type MeetingRoom struct {}
type AuthSession struct {}

// Good: カスタムエラータイプ
type AuthTimeoutError struct {
    err error
}
```

### 関数とメソッド
```go
// Good: 動詞で始まる
func FindAvailableMeetingRooms() {}
func ReserveMeetingRoom() {}
func CancelMeetingEvent() {}

// Good: コンストラクタ関数はNew+型名
func NewUserRepository() UserRepository {}
func NewCalendarUsecase() CalendarUsecase {}
```

### 変数
```go
// Good: 略語を避ける
var meetingRoomRepository MeetingRoomRepository
var userUsecase UserUsecase

// Good: スライスや複数形は複数形で
var rooms []MeetingRoom
var events []MeetingEvent
```

## パッケージ構成

### パッケージ名
- 短く、小文字で統一
- 複数単語の場合はアンダースコアを使用しない
- パッケージの責務を明確に表現

```go
// Good
package usecase
package repository
package controller

// Bad
package user_usecase
package meetingRoomRepository
```

### インポート順序
```go
import (
    // 1. 標準ライブラリ
    "errors"
    "fmt"
    "time"
    
    // 2. サードパーティライブラリ
    "github.com/slack-go/slack"
    "golang.org/x/oauth2"
    "gorm.io/gorm"
    
    // 3. 内部パッケージ
    "quickmtg/internal/domain"
    "quickmtg/internal/application/repository"
)
```

## インターフェース設計

### インターフェース定義の原則
- インターフェースは使用する側（アプリケーション層）で定義
- 小さく、焦点を絞ったインターフェースを作成
- 実装の詳細に依存しない抽象化

```go
// Good: アプリケーション層でインターフェースを定義
package repository

type MeetingRoomRepository interface {
    FindCandidateMeetingRooms(officeName string, floor int, capacity int) ([]domain.MeetingRoom, error)
    FindMeetingRooms(resourceEmails []string) ([]domain.MeetingRoom, error)
    GetAllMeetingRoomResourceNames() ([]string, error)
}
```

### 実装の命名
```go
// Good: 実装は具体的な技術を含む名前
type PostgresMeetingRoomRepository struct {
    Dbc *gorm.DB
}

type GoogleCalendarResourceRepository struct {
    // 実装詳細
}
```

## エラーハンドリング

### カスタムエラータイプの使用
プロジェクトではビジネスロジックに応じたカスタムエラータイプを定義し、適切なエラーハンドリングを行います。

```go
// Good: カスタムエラータイプの定義
type AuthTimeoutError struct {
    err error
}

func (e *AuthTimeoutError) Error() string {
    return "認証開始から時間が経ち過ぎているため認証できませんでした。再度認証を行ってください。"
}

func NewAuthTimeoutError(err error) *AuthTimeoutError {
    return &AuthTimeoutError{err: err}
}
```

### エラーチェックパターン
```go
// Good: エラーは即座にチェック
user, err := c.UserRepository.Get(SlackId)
if err != nil {
    return nil, err
}

// Good: 型アサーションによるエラー分岐
var authTimeoutErr *AuthTimeoutError
if errors.As(err, &authTimeoutErr) {
    // 特定のエラータイプに対する処理
    return handleAuthTimeout()
}
```

### エラーラッピング
```go
// Good: エラーをラップして詳細を追加
if err != nil {
    return fmt.Errorf("failed to find meeting room: %w", err)
}
```

## データベース操作

### リポジトリパターン
```go
// Good: リポジトリ実装
type PostgresMeetingRoomRepository struct {
    Dbc *gorm.DB
}

func NewMeetingRoomRepository(db *gorm.DB) PostgresMeetingRoomRepository {
    return PostgresMeetingRoomRepository{Dbc: db}
}

func (repo PostgresMeetingRoomRepository) FindCandidateMeetingRooms(
    officeName string, 
    floor int, 
    capacity int,
) ([]domain.MeetingRoom, error) {
    var rooms []Result
    // データベース操作実装
    return utils.Map(convertToMeetingRoom, rooms), nil
}
```

## コメント規則

### 日本語コメントの使用
プロジェクトでは日本語でのコメントを積極的に使用し、ビジネスロジックや複雑な処理の説明を行います。

```go
// Good: 関数の詳細な説明（日本語）
// 空いてる部屋を探す関数
// コメントの数字は予約フロー図に対応している。
func (c calendarUsecaseImpl) FindAvailableMeetingRooms(
    SlackId string, 
    people int, 
    start time.Time, 
    end time.Time, 
    floor int,
) (map[int]domain.MeetingRoom, error) {
    // 1. 指定されたフロアの予約可能な優先予約会議室が存在するか
    if rooms := c.checkFavoriteRooms(availableRoomsByFloor, userFavoriteMeetingRooms, userFavoriteFloor, floor); rooms != nil {
        return rooms, nil
    }
    // 実装続く...
}
```

## テスト

### テストファイル命名
```go
// テストファイル: *_test.go
// 例: meeting_room_test.go, calendar_test.go
```

### テスト関数命名
```go
// Good: Test + 関数名 + 条件
func TestCheckStandardOccupancy_WithValidInput_ReturnsTrue(t *testing.T) {
    // テスト実装
}

func TestValidateMeetingRoomCapacity_WithZeroPeople_ReturnsError(t *testing.T) {
    // テスト実装
}
```

## 依存性注入

### コンストラクタパターン
```go
// Good: 依存性は引数で注入
func NewCalendarUsecase(
    userRepository repository.UserRepository,
    meetingRoomRepository repository.MeetingRoomRepository,
    calendarResourceRepository repository.CalendarResourceRepository,
    oauth2TokenSourceRepository repository.OAuth2TokenSourceRepository,
) CalendarUsecase {
    return calendarUsecaseImpl{
        UserRepository:              userRepository,
        MeetingRoomRepository:       meetingRoomRepository,
        CalendarResourceRepository:  calendarResourceRepository,
        OAuth2TokenSourceRepository: oauth2TokenSourceRepository,
    }
}
```

### main関数でのワイヤリング
```go
func main() {
    db := db.ConnectDB()
    
    // リポジトリの初期化
    userRepository := infraRepository.NewUserRepository(db)
    meetingRoomRepository := infraRepository.NewMeetingRoomRepository(db)
    
    // ユースケースの初期化
    userUsecase := usecase.NewUserUsecase(userRepository, meetingRoomRepository)
    calendarUsecase := usecase.NewCalendarUsecase(userRepository, meetingRoomRepository, calendarRepository, oauth2TokenSourceRepository)
    
    // アプリケーション起動
}
```

## 並行処理

### Goroutineの使用
```go
// Good: 時間のかかる処理は別goroutineで実行
func (c calendarUsecaseImpl) ReserveMeetingRoom(
    SlackId string, 
    ParticipantEmails []string, 
    StartAt time.Time, 
    EndAt time.Time, 
    MeetingRoom domain.MeetingRoom,
) (*domain.MeetingEvent, <-chan bool, error) {
    isDoubleBooking := make(chan bool)
    
    // 予約処理
    result, err := c.CalendarResourceRepository.CreateEvent(event, token)
    if err != nil {
        return nil, isDoubleBooking, err
    }

    // ダブルブッキングをgoroutineで検知する
    go func() {
        isDoubleBooking <- c.detectDoubleBooking(result, token)
        close(isDoubleBooking)
    }()

    return result, isDoubleBooking, nil
}
```

### チャネルの使用
```go
// Good: チャネルを使った非同期通信
isDoubleBooking := make(chan bool)

// goroutineでの処理
go func() {
    defer close(isDoubleBooking)
    isDoubleBooking <- checkResult()
}()

// 結果の受信
if <-isDoubleBooking {
    // ダブルブッキングの処理
}
```

## 追加ガイドライン

### ファイル構成
- 1つのファイルに1つの主要な型を定義
- 関連する小さな型は同じファイルに含めても良い
- 大きなファイル（400行以上）は分割を検討

### 定数の定義
```go
// Good: 定数はパッケージレベルで定義
const (
    StatusAccepted    = "accepted"
    StatusNeedsAction = "needsAction"
    StatusDeclined    = "declined"
)
```

### ユーティリティ関数
```go
// Good: 汎用的な関数はutilsパッケージに
func Map[T, U any](fn func(T) U, slice []T) []U {
    result := make([]U, len(slice))
    for i, v := range slice {
        result[i] = fn(v)
    }
    return result
}
```

---

このコーディング規則に従うことで、プロジェクト全体で一貫性のある、保守しやすいコードを維持できます。新しい要件や技術的な課題が発生した際は、この規則を適宜更新してください。 