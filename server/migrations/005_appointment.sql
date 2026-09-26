-- 005_appointment: appointment / appointment_service / appointment_status_log
-- §71: 冲突判定只看 PENDING_CONFIRM/CONFIRMED/IN_SERVICE（应用层过滤，配合 GET_LOCK 日历锁 D5）。
-- D8: 状态迁移一律条件 UPDATE。

CREATE TABLE appointment (
  id              CHAR(26)     NOT NULL PRIMARY KEY,
  appointment_no  VARCHAR(24)  NOT NULL,
  member_id       CHAR(26)     NOT NULL,
  scheduled_start DATETIME     NOT NULL,
  scheduled_end   DATETIME     NOT NULL,
  status          VARCHAR(20)  NOT NULL DEFAULT 'PENDING_CONFIRM',
  customer_note   VARCHAR(500) NOT NULL DEFAULT '',
  internal_note   VARCHAR(500) NOT NULL DEFAULT '',
  confirmed_at    DATETIME     NULL,
  started_at      DATETIME     NULL,
  completed_at    DATETIME     NULL,
  cancelled_at    DATETIME     NULL,
  created_at      DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at      DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  UNIQUE KEY uk_appointment_no (appointment_no),
  KEY idx_appointment_member (member_id, scheduled_start),
  KEY idx_appointment_status_start (status, scheduled_start),
  KEY idx_appointment_time_range (scheduled_start, scheduled_end),
  CONSTRAINT ck_appointment_status CHECK (status IN
    ('PENDING_CONFIRM','CONFIRMED','IN_SERVICE','COMPLETED','CANCELLED','NO_SHOW')),
  CONSTRAINT ck_appointment_time CHECK (scheduled_end > scheduled_start),
  CONSTRAINT fk_apt_member FOREIGN KEY (member_id) REFERENCES member (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE appointment_service (
  id                        CHAR(26)     NOT NULL PRIMARY KEY,
  appointment_id            CHAR(26)     NOT NULL,
  service_id                CHAR(26)     NOT NULL,
  service_name_snapshot     VARCHAR(100) NOT NULL,
  duration_minutes_snapshot INT          NOT NULL,
  price_snapshot            BIGINT       NOT NULL,
  quantity                  INT          NOT NULL DEFAULT 1,
  created_at                DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
  KEY idx_apt_service_appointment (appointment_id),
  CONSTRAINT fk_aps_appointment FOREIGN KEY (appointment_id) REFERENCES appointment (id),
  CONSTRAINT fk_aps_service FOREIGN KEY (service_id) REFERENCES service (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE appointment_status_log (
  id             CHAR(26)     NOT NULL PRIMARY KEY,
  appointment_id CHAR(26)     NOT NULL,
  from_status    VARCHAR(20)  NOT NULL DEFAULT '',
  to_status      VARCHAR(20)  NOT NULL,
  operator_type  VARCHAR(20)  NOT NULL,
  operator_id    CHAR(26)     NULL,
  remark         VARCHAR(500) NOT NULL DEFAULT '',
  created_at     DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
  KEY idx_apt_log_appointment (appointment_id, created_at),
  CONSTRAINT fk_asl_appointment FOREIGN KEY (appointment_id) REFERENCES appointment (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
