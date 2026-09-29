-- 005_appointment: appointment / appointment_service / appointment_status_log
-- SQLite 翻译说明：MySQL 版 012 曾把状态机收紧为 WAITING→IN_SERVICE→COMPLETED
-- （DROP CHECK + UPDATE + ADD CHECK，见 012 注释）。SQLite 无法 DROP/ADD 表级 CHECK，
-- 且全新 SQLite 库不存在中间状态行，故此处直接按 012 之后的最终形态建表
-- （DEFAULT 'WAITING' + 最终状态集 CHECK）；012 中对应 UPDATE/重建步骤为空操作。
-- §71: 冲突判定只看 WAITING/IN_SERVICE（应用层过滤，配合 BEGIN IMMEDIATE 串行化）。
-- D8: 状态迁移一律条件 UPDATE。

CREATE TABLE appointment (
  id              CHAR(26)     NOT NULL PRIMARY KEY,
  appointment_no  VARCHAR(24)  NOT NULL,
  member_id       CHAR(26)     NOT NULL,
  scheduled_start DATETIME     NOT NULL,
  scheduled_end   DATETIME     NOT NULL,
  status          VARCHAR(20)  NOT NULL DEFAULT 'WAITING',
  slot_type       VARCHAR(12)  NOT NULL DEFAULT 'SPECIFIC',
  customer_note   VARCHAR(500) NOT NULL DEFAULT '',
  internal_note   VARCHAR(500) NOT NULL DEFAULT '',
  confirmed_at    DATETIME     NULL,
  started_at      DATETIME     NULL,
  completed_at    DATETIME     NULL,
  cancelled_at    DATETIME     NULL,
  created_at      DATETIME     NOT NULL DEFAULT (datetime('now','+8 hours')),
  updated_at      DATETIME     NOT NULL DEFAULT (datetime('now','+8 hours')),
  CONSTRAINT uk_appointment_no UNIQUE (appointment_no),
  CONSTRAINT ck_appointment_status CHECK (status IN
    ('WAITING','IN_SERVICE','COMPLETED','CANCELLED','NO_SHOW')),
  CONSTRAINT ck_appointment_slot_type CHECK (slot_type IN ('SPECIFIC','HALF_DAY')),
  CONSTRAINT ck_appointment_time CHECK (scheduled_end > scheduled_start),
  CONSTRAINT fk_apt_member FOREIGN KEY (member_id) REFERENCES member (id)
);

CREATE INDEX idx_appointment_member ON appointment (member_id, scheduled_start);
CREATE INDEX idx_appointment_status_start ON appointment (status, scheduled_start);
CREATE INDEX idx_appointment_time_range ON appointment (scheduled_start, scheduled_end);

CREATE TRIGGER trg_appointment_updated_at AFTER UPDATE ON appointment
FOR EACH ROW WHEN NEW.updated_at = OLD.updated_at
BEGIN
  UPDATE appointment SET updated_at = datetime('now','+8 hours') WHERE id = NEW.id;
END;

CREATE TABLE appointment_service (
  id                        CHAR(26)     NOT NULL PRIMARY KEY,
  appointment_id            CHAR(26)     NOT NULL,
  service_id                CHAR(26)     NOT NULL,
  service_name_snapshot     VARCHAR(100) NOT NULL,
  duration_minutes_snapshot INT          NOT NULL,
  price_snapshot            BIGINT       NOT NULL,
  quantity                  INT          NOT NULL DEFAULT 1,
  created_at                DATETIME     NOT NULL DEFAULT (datetime('now','+8 hours')),
  CONSTRAINT fk_aps_appointment FOREIGN KEY (appointment_id) REFERENCES appointment (id),
  CONSTRAINT fk_aps_service FOREIGN KEY (service_id) REFERENCES service (id)
);

CREATE INDEX idx_apt_service_appointment ON appointment_service (appointment_id);

CREATE TABLE appointment_status_log (
  id             CHAR(26)     NOT NULL PRIMARY KEY,
  appointment_id CHAR(26)     NOT NULL,
  from_status    VARCHAR(20)  NOT NULL DEFAULT '',
  to_status      VARCHAR(20)  NOT NULL,
  operator_type  VARCHAR(20)  NOT NULL,
  operator_id    CHAR(26)     NULL,
  remark         VARCHAR(500) NOT NULL DEFAULT '',
  created_at     DATETIME     NOT NULL DEFAULT (datetime('now','+8 hours')),
  CONSTRAINT fk_asl_appointment FOREIGN KEY (appointment_id) REFERENCES appointment (id)
);

CREATE INDEX idx_apt_log_appointment ON appointment_status_log (appointment_id, created_at);
