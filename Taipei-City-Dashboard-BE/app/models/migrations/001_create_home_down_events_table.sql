-- 使用者回報事件表
-- 儲存使用者回報的掛點事件資訊

CREATE TABLE public.home_down_events (
    id SERIAL PRIMARY KEY, -- 唯一識別碼 (SERIAL 會自動產生並設定為 PRIMARY KEY，等同於 auto-increment)
    latitude DECIMAL(10, 8) NOT NULL, -- 回報地點的緯度 (經緯度座標的緯度部分)，DECIMAL 確保精度
    longitude DECIMAL(11, 8) NOT NULL, -- 回報地點的經度 (經緯度座標的經度部分)，DECIMAL 確保精度
    type VARCHAR(50) NOT NULL, -- 掛掉的類型 (對應API的type欄位)
    name VARCHAR(255), -- 使用者輸入的代稱，允許為空 (NULL)
    reported_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP -- 回報時間 (事件記錄建立時間)，包含時區資訊
);

-- 加入欄位說明 (PostgreSQL 的 COMMENT 語法)
COMMENT ON TABLE public.home_down_events IS '儲存使用者透過 /api/v1/homeDown 回報的掛點事件資訊';
COMMENT ON COLUMN public.home_down_events.id IS '唯一識別碼';
COMMENT ON COLUMN public.home_down_events.latitude IS '回報地點的緯度';
COMMENT ON COLUMN public.home_down_events.longitude IS '回報地點的經度';
COMMENT ON COLUMN public.home_down_events.type IS '掛掉的類型';
COMMENT ON COLUMN public.home_down_events.name IS '使用者輸入的代稱';
COMMENT ON COLUMN public.home_down_events.reported_at IS '回報時間';
