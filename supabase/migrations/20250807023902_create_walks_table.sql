-- 散歩記録テーブルの作成
-- api.mdのスキーマ仕様に基づいた散歩データテーブル

CREATE TABLE IF NOT EXISTS walks (
    -- 主キー: ユニークな散歩ID
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    
    -- 物語のタイトル
    title VARCHAR(500) NOT NULL,
    
    -- エリア名
    area VARCHAR(255) NOT NULL,
    
    -- 物語の本文
    description TEXT NOT NULL,
    
    -- テーマ
    theme VARCHAR(100) NOT NULL,
    
    -- 訪問したPOIのID配列（JSONB配列）
    poi_ids JSONB DEFAULT '[]'::jsonb,
    
    -- タグ（JSONB配列）
    tags JSONB DEFAULT '[]'::jsonb,
    
    -- 実績時間（分）
    duration_minutes INTEGER NOT NULL CHECK (duration_minutes > 0),
    
    -- 実績距離（メートル）
    distance_meters INTEGER NOT NULL CHECK (distance_meters > 0),
    
    -- ルートの軌跡（Google Maps Polyline形式）
    route_polyline TEXT NOT NULL,
    
    -- 感想
    impressions TEXT,
    
    -- 投稿日時
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL,
    
    -- 更新日時
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL
);

-- 更新日時を自動更新するトリガー関数
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- 更新日時自動更新トリガー
CREATE TRIGGER update_walks_updated_at 
    BEFORE UPDATE ON walks 
    FOR EACH ROW 
    EXECUTE FUNCTION update_updated_at_column();

-- コメント追加
COMMENT ON TABLE walks IS '散歩記録テーブル - ユーザーが完了した散歩の物語と詳細データ';
COMMENT ON COLUMN walks.id IS 'ユニークな散歩ID';
COMMENT ON COLUMN walks.title IS '物語のタイトル';
COMMENT ON COLUMN walks.area IS 'エリア名';
COMMENT ON COLUMN walks.description IS '物語の本文';
COMMENT ON COLUMN walks.theme IS 'テーマ（gourmet, nature, culture等）';
COMMENT ON COLUMN walks.poi_ids IS '訪問したPOIのID配列（JSON）';
COMMENT ON COLUMN walks.tags IS 'タグ配列（JSON）';
COMMENT ON COLUMN walks.duration_minutes IS '実績時間（分）';
COMMENT ON COLUMN walks.distance_meters IS '実績距離（メートル）';
COMMENT ON COLUMN walks.route_polyline IS 'ルートの軌跡（Google Maps Polyline形式）';
COMMENT ON COLUMN walks.impressions IS '感想';
COMMENT ON COLUMN walks.created_at IS '投稿日時';
COMMENT ON COLUMN walks.updated_at IS '更新日時';