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
  created_at  DATETIME      NOT NULL DEFAULT CURRENT_TIMESTAMP,
  KEY idx_ops_log_created (created_at),
  KEY idx_ops_log_action (action, created_at),
  KEY idx_ops_log_actor (actor_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE ops_insight_snapshot (
  id           CHAR(26)      NOT NULL PRIMARY KEY,
  snapshot_date DATE         NOT NULL,
  kind         VARCHAR(30)   NOT NULL,
  member_id    CHAR(26)      NOT NULL,
  detail       VARCHAR(1000) NOT NULL DEFAULT '',
  created_at   DATETIME      NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE KEY uk_insight (snapshot_date, kind, member_id),
  CONSTRAINT ck_insight_kind CHECK (kind IN ('LOW_BALANCE','DORMANT','EXPIRING')),
  CONSTRAINT fk_insight_member FOREIGN KEY (member_id) REFERENCES member (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
