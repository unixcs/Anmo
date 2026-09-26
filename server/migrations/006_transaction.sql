-- 006_transaction: payment / redemption / redemption_reversal
-- D1: payment.status ∈ {VALID, VOIDED}；收款统计只算 VALID；撤销核销同事务置 VOIDED。
-- W-E: redemption 撤销只置 REVERSED 不删行；active_lock 生成列 + UNIQUE 保证
--      "一预约最多一笔有效核销"且支持撤销后重新核销；idempotency_key 为请求级 UUID。

CREATE TABLE payment (
  id             CHAR(26)     NOT NULL PRIMARY KEY,
  appointment_id CHAR(26)     NOT NULL,
  member_id      CHAR(26)     NOT NULL,
  amount         BIGINT       NOT NULL,
  method         VARCHAR(20)  NOT NULL,
  status         VARCHAR(20)  NOT NULL DEFAULT 'VALID',
  reference_no   VARCHAR(100) NOT NULL DEFAULT '',
  remark         VARCHAR(500) NOT NULL DEFAULT '',
  recorded_by    CHAR(26)     NULL,
  recorded_at    DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
  created_at     DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at     DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  KEY idx_payment_appointment (appointment_id),
  KEY idx_payment_status_recorded (status, recorded_at),
  CONSTRAINT ck_payment_method CHECK (method IN ('CARD','WECHAT_TRANSFER','CASH','OTHER')),
  CONSTRAINT ck_payment_status CHECK (status IN ('VALID','VOIDED')),
  CONSTRAINT ck_payment_amount CHECK (amount >= 0),
  CONSTRAINT fk_pay_appointment FOREIGN KEY (appointment_id) REFERENCES appointment (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE redemption (
  id              CHAR(26)    NOT NULL PRIMARY KEY,
  appointment_id  CHAR(26)    NOT NULL,
  member_id       CHAR(26)    NOT NULL,
  member_card_id  CHAR(26)    NOT NULL,
  service_id      CHAR(26)    NOT NULL,
  quantity        INT         NOT NULL DEFAULT 1,
  before_count    INT         NOT NULL,
  after_count     INT         NOT NULL,
  status          VARCHAR(20) NOT NULL DEFAULT 'SUCCESS',
  idempotency_key VARCHAR(64) NOT NULL,
  active_lock     CHAR(26)    GENERATED ALWAYS AS (
                    CASE WHEN status = 'SUCCESS' THEN appointment_id ELSE NULL END
                  ) STORED,
  operator_id     CHAR(26)    NULL,
  created_at      DATETIME    NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE KEY uk_redemption_idempotency (idempotency_key),
  UNIQUE KEY uk_redemption_active_lock (active_lock),
  KEY idx_redemption_appointment (appointment_id),
  KEY idx_redemption_card (member_card_id),
  CONSTRAINT ck_redemption_status CHECK (status IN ('SUCCESS','REVERSED')),
  CONSTRAINT ck_redemption_quantity CHECK (quantity > 0),
  CONSTRAINT fk_rdm_appointment FOREIGN KEY (appointment_id) REFERENCES appointment (id),
  CONSTRAINT fk_rdm_card FOREIGN KEY (member_card_id) REFERENCES member_card (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE redemption_reversal (
  id           CHAR(26)     NOT NULL PRIMARY KEY,
  redemption_id CHAR(26)    NOT NULL,
  reason       VARCHAR(500) NOT NULL DEFAULT '',
  operator_id  CHAR(26)     NULL,
  created_at   DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE KEY uk_reversal_redemption (redemption_id),
  CONSTRAINT fk_rr_redemption FOREIGN KEY (redemption_id) REFERENCES redemption (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
