-- 010_payment_idem: 对抗审查 B1/W2 修复。
-- B1: valid_lock 生成列 + UNIQUE 强制"一预约最多一笔 VALID payment"（对齐 redemption.active_lock 模式）。
-- W2: payment 增加 idempotency_key UNIQUE，SettleByPay 幂等回放。

ALTER TABLE payment
  ADD COLUMN idempotency_key VARCHAR(64) NULL,
  ADD COLUMN valid_lock CHAR(26) GENERATED ALWAYS AS (
    CASE WHEN status = 'VALID' THEN appointment_id ELSE NULL END
  ) STORED,
  ADD UNIQUE KEY uk_payment_idempotency (idempotency_key),
  ADD UNIQUE KEY uk_payment_valid_lock (valid_lock);
