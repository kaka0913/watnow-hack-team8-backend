# Team8-App

Team8のGoアプリケーションプロジェクトです。

## 概要

Team8のモバイルアプリのためのサーバー用のプロジェクトです。

## 機能

- `/` - ホームページ
- `/api/health` - ヘルスチェックAPI
- `/api/team` - チーム情報API

## 使用方法

### プロジェクトの実行

```bash
go run main.go
```

### アクセス

- ホームページ: http://localhost:8080/
- ヘルスチェック: http://localhost:8080/api/health
- チーム情報: http://localhost:8080/api/team

## 開発

### 依存関係のインストール



### ビルド

```bash
go build -o bin/team8-app
```

### 実行

```bash
./bin/team8-app
```