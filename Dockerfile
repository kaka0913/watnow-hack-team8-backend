# Build stage: Go アプリケーションをビルド
FROM golang:1.23-alpine AS builder

# セキュリティ向上のためのapk update & ca-certificates
RUN apk --no-cache add ca-certificates git tzdata

# ワーキングディレクトリを設定
WORKDIR /app

# 依存関係ファイルをコピーし、依存関係をダウンロード
COPY go.mod go.sum ./
RUN go mod download

# ソースコードをコピー
COPY . .

# アプリケーションをビルド（静的リンク、デバッグ情報を削除）
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags='-w -s -extldflags "-static"' \
    -a -installsuffix cgo \
    -o main ./cmd/main.go

FROM gcr.io/distroless/static-debian12:nonroot

# メタデータ
LABEL maintainer="Team8-App"
LABEL description="Team8-App API Server for Cloud Run"
LABEL version="1.0.0"

# ビルド成果物をコピー
COPY --from=builder /app/main /main

# タイムゾーン情報をコピー（必要に応じて）
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# CA証明書をコピー（HTTPS通信に必要）
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# 非rootユーザーで実行（distrolessはnonrootユーザーを含む）
USER nonroot:nonroot

# Cloud Runのポート（環境変数PORTから取得、デフォルト8080）
EXPOSE 8080

# アプリケーションを実行
ENTRYPOINT ["/main"]
