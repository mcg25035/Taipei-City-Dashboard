-- 基礎設施公告事件表
-- 儲存Infra系統發布的掛點公告資訊

CREATE TABLE public.infra_down_events (
    id SERIAL PRIMARY KEY, -- 唯一識別碼 (SERIAL 會自動產生並設定為 PRIMARY KEY)
    type VARCHAR(50) NOT NULL, -- 掛掉的類型 (對應API的type欄位)
    start_time TIMESTAMP WITH TIME ZONE NOT NULL, -- 事件開始時間 (對應API的timeMin)，時間戳數字需要轉換為 TIMESTAMP
    end_time TIMESTAMP WITH TIME ZONE NOT NULL, -- 事件結束時間 (對應API的timeMax)，時間戳數字需要轉換為 TIMESTAMP
    area_data JSONB, -- 停電/掛點區域範圍資料 (JSON 格式)，使用 JSONB 以獲得更好的性能和查詢能力，允許為空 (NULL)
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP -- 記錄建立時間 (公告發布到系統的時間)，包含時區資訊
);

-- 加入欄位說明 (PostgreSQL 的 COMMENT 語法)
COMMENT ON TABLE public.infra_down_events IS '儲存 Infra 系統發布的掛點公告資訊';
COMMENT ON COLUMN public.infra_down_events.id IS '唯一識別碼';
COMMENT ON COLUMN public.infra_down_events.type IS '掛掉的類型';
COMMENT ON COLUMN public.infra_down_events.start_time IS '事件開始時間 (來自 API 的 timeMin)';
COMMENT ON COLUMN public.infra_down_events.end_time IS '事件結束時間 (來自 API 的 timeMax)';
COMMENT ON COLUMN public.infra_down_events.area_data IS '停電/掛點區域範圍資料 (JSON 格式)';
COMMENT ON COLUMN public.infra_down_events.created_at IS '公告資料在資料庫中的建立時間';
