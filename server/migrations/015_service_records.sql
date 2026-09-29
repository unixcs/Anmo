-- 015_service_records: V2.2 第三批（plan §四/§五/§六）。
-- D28: service_tag（调理部位/服务方式两组标签）+ service_record（结算时采集的服务记录，
--      历史存名称快照，标签增删/停用不影响历史；REVERSED 只置状态不删行）。
-- D29: payment 重建，member_id NOT NULL → NULL（散客不录手机号仅记账，无 member 档案）。
--
-- payment 重建说明（照 014 先例）：SQLite 不支持 ALTER COLUMN，重建表。
-- 旧表 RENAME 方案在事务内不可行（modernc 内核 RENAME 会跟随改写子表 FK 引用目标表名，
-- 且不清 defer 违例计数），故用 014 验证过的序列：
--   PRAGMA defer_foreign_keys = ON（事务内可设，COMMIT 后自动复位）
--   → CREATE payment_tmp → 回填 → DROP payment → 同名 CREATE（最终形态）
--   → 回填（显式列，不含生成列）→ DROP payment_tmp
--   → 重建两个唯一索引 + 普通索引 + 触发器。
-- 最终形态 = 006 全列 + CHECK + fk_pay_appointment + 010 的 idempotency_key UNIQUE、
-- valid_lock VIRTUAL 生成列 + uk_payment_valid_lock；唯一改动 member_id CHAR(26) NULL。
-- 本文件执行到 service_record 建表时 payment 已重建完毕，其 FK 指向新表。
PRAGMA defer_foreign_keys = ON;

-- ---------- D28: 服务标签（商家内部数据，用户端不可见） ----------
CREATE TABLE service_tag (
  id         CHAR(26)    NOT NULL PRIMARY KEY,
  tag_group  VARCHAR(20) NOT NULL,
  name       VARCHAR(32) NOT NULL,
  sort       INTEGER     NOT NULL DEFAULT 0,
  status     VARCHAR(20) NOT NULL DEFAULT 'ACTIVE',
  created_at DATETIME    NOT NULL DEFAULT (datetime('now','+8 hours')),
  updated_at DATETIME    NOT NULL DEFAULT (datetime('now','+8 hours')),
  CONSTRAINT uk_tag_group_name UNIQUE (tag_group, name),
  CONSTRAINT ck_tag_group  CHECK (tag_group IN ('BODY_PART','METHOD')),
  CONSTRAINT ck_tag_status CHECK (status IN ('ACTIVE','DISABLED'))
);

CREATE INDEX idx_tag_group_sort ON service_tag (tag_group, sort);

CREATE TRIGGER trg_service_tag_updated_at AFTER UPDATE ON service_tag
FOR EACH ROW WHEN NEW.updated_at = OLD.updated_at
BEGIN
  UPDATE service_tag SET updated_at = datetime('now','+8 hours') WHERE id = NEW.id;
END;

-- ---------- D29: payment 重建（member_id 放宽为 NULL） ----------
CREATE TABLE payment_tmp (
  id             CHAR(26)     NOT NULL PRIMARY KEY,
  appointment_id CHAR(26)     NULL,
  member_id      CHAR(26)     NULL,
  amount         BIGINT       NOT NULL,
  method         VARCHAR(20)  NOT NULL,
  status         VARCHAR(20)  NOT NULL DEFAULT 'VALID',
  reference_no   VARCHAR(100) NOT NULL DEFAULT '',
  remark         VARCHAR(500) NOT NULL DEFAULT '',
  recorded_by    CHAR(26)     NULL,
  recorded_at    DATETIME     NOT NULL DEFAULT (datetime('now','+8 hours')),
  created_at     DATETIME     NOT NULL DEFAULT (datetime('now','+8 hours')),
  updated_at     DATETIME     NOT NULL DEFAULT (datetime('now','+8 hours')),
  idempotency_key VARCHAR(64) NULL
);

INSERT INTO payment_tmp (id, appointment_id, member_id, amount, method, status,
                         reference_no, remark, recorded_by, recorded_at, created_at, updated_at,
                         idempotency_key)
  SELECT id, appointment_id, member_id, amount, method, status,
         reference_no, remark, recorded_by, recorded_at, created_at, updated_at,
         idempotency_key
    FROM payment;

DROP TABLE payment;

CREATE TABLE payment (
  id             CHAR(26)     NOT NULL PRIMARY KEY,
  appointment_id CHAR(26)     NULL,
  member_id      CHAR(26)     NULL,
  amount         BIGINT       NOT NULL,
  method         VARCHAR(20)  NOT NULL,
  status         VARCHAR(20)  NOT NULL DEFAULT 'VALID',
  reference_no   VARCHAR(100) NOT NULL DEFAULT '',
  remark         VARCHAR(500) NOT NULL DEFAULT '',
  recorded_by    CHAR(26)     NULL,
  recorded_at    DATETIME     NOT NULL DEFAULT (datetime('now','+8 hours')),
  created_at     DATETIME     NOT NULL DEFAULT (datetime('now','+8 hours')),
  updated_at     DATETIME     NOT NULL DEFAULT (datetime('now','+8 hours')),
  idempotency_key VARCHAR(64) NULL,
  valid_lock     CHAR(26)     GENERATED ALWAYS AS
                   (CASE WHEN status = 'VALID' THEN appointment_id END) VIRTUAL,
  CONSTRAINT ck_payment_method CHECK (method IN ('CARD','WECHAT_TRANSFER','CASH','OTHER')),
  CONSTRAINT ck_payment_status CHECK (status IN ('VALID','VOIDED')),
  CONSTRAINT ck_payment_amount CHECK (amount >= 0),
  CONSTRAINT fk_pay_appointment FOREIGN KEY (appointment_id) REFERENCES appointment (id)
);

INSERT INTO payment (id, appointment_id, member_id, amount, method, status,
                     reference_no, remark, recorded_by, recorded_at, created_at, updated_at,
                     idempotency_key)
  SELECT id, appointment_id, member_id, amount, method, status,
         reference_no, remark, recorded_by, recorded_at, created_at, updated_at,
         idempotency_key
    FROM payment_tmp;

DROP TABLE payment_tmp;

CREATE INDEX idx_payment_appointment ON payment (appointment_id);
CREATE INDEX idx_payment_status_recorded ON payment (status, recorded_at);
CREATE UNIQUE INDEX uk_payment_idempotency ON payment (idempotency_key);
CREATE UNIQUE INDEX uk_payment_valid_lock ON payment (valid_lock);

CREATE TRIGGER trg_payment_updated_at AFTER UPDATE ON payment
FOR EACH ROW WHEN NEW.updated_at = OLD.updated_at
BEGIN
  UPDATE payment SET updated_at = datetime('now','+8 hours') WHERE id = NEW.id;
END;

-- ---------- D28: 服务记录（结算事务内落库，历史存快照） ----------
CREATE TABLE service_record (
  id               CHAR(26)     NOT NULL PRIMARY KEY,
  member_id        CHAR(26)     NULL,
  appointment_id   CHAR(26)     NULL,
  payment_id       CHAR(26)     NOT NULL,
  redemption_id    CHAR(26)     NULL,
  service_id       CHAR(26)     NOT NULL,
  service_name     VARCHAR(128) NOT NULL,
  body_parts       VARCHAR(256) NOT NULL DEFAULT '[]',
  service_method   VARCHAR(64)  NOT NULL DEFAULT '',
  tech_note        VARCHAR(200) NOT NULL DEFAULT '',
  merchant_note    VARCHAR(500) NOT NULL DEFAULT '',
  merchant_note_by CHAR(26)     NULL,
  merchant_note_at DATETIME     NULL,
  communicated     INTEGER      NOT NULL DEFAULT 0,
  status           VARCHAR(20)  NOT NULL DEFAULT 'ACTIVE',
  reversed_at      DATETIME     NULL,
  created_by       CHAR(26)     NULL,
  created_at       DATETIME     NOT NULL DEFAULT (datetime('now','+8 hours')),
  updated_at       DATETIME     NOT NULL DEFAULT (datetime('now','+8 hours')),
  CONSTRAINT ck_rec_status CHECK (status IN ('ACTIVE','REVERSED')),
  CONSTRAINT ck_rec_communicated CHECK (communicated IN (0,1)),
  CONSTRAINT fk_rec_member      FOREIGN KEY (member_id)     REFERENCES member (id),
  CONSTRAINT fk_rec_appointment FOREIGN KEY (appointment_id) REFERENCES appointment (id),
  CONSTRAINT fk_rec_payment     FOREIGN KEY (payment_id)    REFERENCES payment (id),
  CONSTRAINT fk_rec_redemption  FOREIGN KEY (redemption_id) REFERENCES redemption (id)
);

CREATE INDEX idx_rec_member_time ON service_record (member_id, created_at);
CREATE INDEX idx_rec_appointment ON service_record (appointment_id);
CREATE INDEX idx_rec_redemption  ON service_record (redemption_id);
CREATE INDEX idx_rec_payment     ON service_record (payment_id);

CREATE TRIGGER trg_service_record_updated_at AFTER UPDATE ON service_record
FOR EACH ROW WHEN NEW.updated_at = OLD.updated_at
BEGIN
  UPDATE service_record SET updated_at = datetime('now','+8 hours') WHERE id = NEW.id;
END;
