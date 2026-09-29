-- 006_transaction: payment / redemption / redemption_reversal
-- D1: payment.status ∈ {VALID, VOIDED}；收款统计只算 VALID；撤销核销同事务置 VOIDED。
-- W-E: redemption 撤销只置 REVERSED 不删行；active_lock 生成列 + UNIQUE 保证
--      "一预约最多一笔有效核销"且支持撤销后重新核销；idempotency_key 为请求级 UUID。
-- SQLite 翻译说明：
--   * MySQL 版 011 曾将 payment/redemption.appointment_id 放宽为 NULL（D19 散客核销），
--     全新库直接按最终形态建 NULL 列；011 中对应 MODIFY 为空操作。
--   * MySQL 版 012 曾给 redemption ADD COLUMN service_name（结算快照），此处直接并入建表。

CREATE TABLE payment (
  id             CHAR(26)     NOT NULL PRIMARY KEY,
  appointment_id CHAR(26)     NULL,
  member_id      CHAR(26)     NOT NULL,
  amount         BIGINT       NOT NULL,
  method         VARCHAR(20)  NOT NULL,
  status         VARCHAR(20)  NOT NULL DEFAULT 'VALID',
  reference_no   VARCHAR(100) NOT NULL DEFAULT '',
  remark         VARCHAR(500) NOT NULL DEFAULT '',
  recorded_by    CHAR(26)     NULL,
  recorded_at    DATETIME     NOT NULL DEFAULT (datetime('now','+8 hours')),
  created_at     DATETIME     NOT NULL DEFAULT (datetime('now','+8 hours')),
  updated_at     DATETIME     NOT NULL DEFAULT (datetime('now','+8 hours')),
  CONSTRAINT ck_payment_method CHECK (method IN ('CARD','WECHAT_TRANSFER','CASH','OTHER')),
  CONSTRAINT ck_payment_status CHECK (status IN ('VALID','VOIDED')),
  CONSTRAINT ck_payment_amount CHECK (amount >= 0),
  CONSTRAINT fk_pay_appointment FOREIGN KEY (appointment_id) REFERENCES appointment (id)
);

CREATE INDEX idx_payment_appointment ON payment (appointment_id);
CREATE INDEX idx_payment_status_recorded ON payment (status, recorded_at);

CREATE TRIGGER trg_payment_updated_at AFTER UPDATE ON payment
FOR EACH ROW WHEN NEW.updated_at = OLD.updated_at
BEGIN
  UPDATE payment SET updated_at = datetime('now','+8 hours') WHERE id = NEW.id;
END;

CREATE TABLE redemption (
  id              CHAR(26)    NOT NULL PRIMARY KEY,
  appointment_id  CHAR(26)    NULL,
  member_id       CHAR(26)    NOT NULL,
  member_card_id  CHAR(26)    NOT NULL,
  service_id      CHAR(26)    NOT NULL,
  service_name    VARCHAR(128) NOT NULL DEFAULT '',
  quantity        INT         NOT NULL DEFAULT 1,
  before_count    INT         NOT NULL,
  after_count     INT         NOT NULL,
  status          VARCHAR(20) NOT NULL DEFAULT 'SUCCESS',
  idempotency_key VARCHAR(64) NOT NULL,
  active_lock     CHAR(26)    GENERATED ALWAYS AS (
                    CASE WHEN status = 'SUCCESS' THEN appointment_id ELSE NULL END
                  ) STORED,
  operator_id     CHAR(26)    NULL,
  created_at      DATETIME    NOT NULL DEFAULT (datetime('now','+8 hours')),
  CONSTRAINT uk_redemption_idempotency UNIQUE (idempotency_key),
  CONSTRAINT uk_redemption_active_lock UNIQUE (active_lock),
  CONSTRAINT ck_redemption_status CHECK (status IN ('SUCCESS','REVERSED')),
  CONSTRAINT ck_redemption_quantity CHECK (quantity > 0),
  CONSTRAINT fk_rdm_appointment FOREIGN KEY (appointment_id) REFERENCES appointment (id),
  CONSTRAINT fk_rdm_card FOREIGN KEY (member_card_id) REFERENCES member_card (id)
);

CREATE INDEX idx_redemption_appointment ON redemption (appointment_id);
CREATE INDEX idx_redemption_card ON redemption (member_card_id);

CREATE TABLE redemption_reversal (
  id           CHAR(26)     NOT NULL PRIMARY KEY,
  redemption_id CHAR(26)    NOT NULL,
  reason       VARCHAR(500) NOT NULL DEFAULT '',
  operator_id  CHAR(26)     NULL,
  created_at   DATETIME     NOT NULL DEFAULT (datetime('now','+8 hours')),
  CONSTRAINT uk_reversal_redemption UNIQUE (redemption_id),
  CONSTRAINT fk_rr_redemption FOREIGN KEY (redemption_id) REFERENCES redemption (id)
);
