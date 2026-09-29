-- 008_ops: ops_operation_log / ops_insight_snapshot
-- D7: 操作日志由 middleware 写入，业务模块不 import ops。

CREATE TABLE ops_operation_log (
  id          CHAR(26)      NOT NULL PRIMARY KEY,
  actor_type  VARCHAR(20)   NOT NULL DEFAULT 'ADMIN',
  actor_id    CHAR(26)      NULL,
  action      VARCHAR(50)   NOT NULL,
  target_type VARCHAR(30)   NOT NULL DEFAULT '',
  target_id   CHAR(26)      NULL,
  detail      VARCHAR(2000) NOT NULL DEFAULT '',
  ip          VARCHAR(45)   NOT NULL DEFAULT '',
  created_at  DATETIME      NOT NULL DEFAULT (datetime('now','+8 hours'))
);

CREATE INDEX idx_ops_log_created ON ops_operation_log (created_at);
CREATE INDEX idx_ops_log_action ON ops_operation_log (action, created_at);
CREATE INDEX idx_ops_log_actor ON ops_operation_log (actor_id);

CREATE TABLE ops_insight_snapshot (
  id           CHAR(26)      NOT NULL PRIMARY KEY,
  snapshot_date DATE         NOT NULL,
  kind         VARCHAR(30)   NOT NULL,
  member_id    CHAR(26)      NOT NULL,
  detail       VARCHAR(1000) NOT NULL DEFAULT '',
  created_at   DATETIME      NOT NULL DEFAULT (datetime('now','+8 hours')),
  CONSTRAINT uk_insight UNIQUE (snapshot_date, kind, member_id),
  CONSTRAINT ck_insight_kind CHECK (kind IN ('LOW_BALANCE','DORMANT','EXPIRING')),
  CONSTRAINT fk_insight_member FOREIGN KEY (member_id) REFERENCES member (id)
);
