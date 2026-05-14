-- todos テーブル
CREATE TABLE IF NOT EXISTS todos (
    id           UUID         PRIMARY KEY,
    title        VARCHAR(200) NOT NULL,
    due_date     TIMESTAMPTZ,
    priority     SMALLINT     NOT NULL,
    status       VARCHAR(16)  NOT NULL,
    completed_at TIMESTAMPTZ
);

-- 期限切れ判定用のindex
CREATE INDEX IF NOT EXISTS idx_todos_status_due_date
    ON todos (status, due_date)
    WHERE status = 'pending';
