-- 010_payment_idem: 对抗审查 B1/W2 修复。
-- B1: valid_lock 生成列 + UNIQUE 强制"一预约最多一笔 VALID payment"（对齐 redemption.active_lock 模式）。
-- W2: payment 增加 idempotency_key UNIQUE，SettleByPay 幂等回放。
-- SQLite 说明：ALTER TABLE ADD COLUMN 的生成列按文档约束用 VIRTUAL（STORED 仅限建表时）；
-- VIRTUAL 生成列上建 UNIQUE 索引语义不变（唯一性按计算值判定，NULL 不参与去重）。

ALTER TABLE payment ADD COLUMN idempotency_key VARCHAR(64) NULL;

ALTER TABLE payment ADD COLUMN valid_lock CHAR(26)
  GENERATED ALWAYS AS (CASE WHEN status = 'VALID' THEN appointment_id END) VIRTUAL;

CREATE UNIQUE INDEX uk_payment_idempotency ON payment (idempotency_key);
CREATE UNIQUE INDEX uk_payment_valid_lock ON payment (valid_lock);
