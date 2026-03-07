CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE videos (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id         VARCHAR(255)     NOT NULL,
    user_email      VARCHAR(255)     NOT NULL DEFAULT '',
    video_key       VARCHAR(1024)    NOT NULL,
    zip_key         VARCHAR(1024)    NOT NULL DEFAULT '',
    original_name   VARCHAR(1024)    NOT NULL DEFAULT '',
    status          VARCHAR(20)      NOT NULL DEFAULT 'PENDING',
    frame_count     INTEGER          NOT NULL DEFAULT 0,
    file_size       BIGINT           NOT NULL DEFAULT 0,
    video_duration  DOUBLE PRECISION NOT NULL DEFAULT 0,
    error_message   TEXT             NOT NULL DEFAULT '',
    created_at      TIMESTAMPTZ      NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ      NOT NULL DEFAULT NOW(),
    completed_at    TIMESTAMPTZ
);

CREATE INDEX idx_videos_user_id    ON videos(user_id);
CREATE INDEX idx_videos_status     ON videos(status);
CREATE INDEX idx_videos_created_at ON videos(created_at);
