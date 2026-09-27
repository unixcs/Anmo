-- V1.x 第二阶段：预约状态机简化 + 结算快照
-- 状态机：WAITING → IN_SERVICE → COMPLETED（异常 WAITING → CANCELLED|NO_SHOW）
-- 用户决策 2026-09-28：取消确认环节，PENDING_CONFIRM/CONFIRMED 退役。
-- 2026-09-28 修订：必须先 DROP 旧 CHECK 再做数据迁移（MySQL CHECK 对 UPDATE 同样生效；
-- 原顺序在含历史行库上违反 ck_appointment_status，yun1 生产首次应用失败）。
-- 本文件从未在任何环境成功应用过（生产重启循环即证），故允许就地修正。

ALTER TABLE appointment DROP CHECK ck_appointment_status;

UPDATE appointment SET status = 'WAITING' WHERE status IN ('PENDING_CONFIRM', 'CONFIRMED');

ALTER TABLE appointment
  ADD CONSTRAINT ck_appointment_status
  CHECK (status IN ('WAITING', 'IN_SERVICE', 'COMPLETED', 'CANCELLED', 'NO_SHOW'));
ALTER TABLE appointment MODIFY COLUMN status VARCHAR(20) NOT NULL DEFAULT 'WAITING';

-- 结算快照：核销记录实际服务名，卡流水记录卡名（防模板改名篡改历史显示）
ALTER TABLE redemption ADD COLUMN service_name VARCHAR(128) NOT NULL DEFAULT '' AFTER service_id;
ALTER TABLE card_transaction ADD COLUMN card_name VARCHAR(128) NOT NULL DEFAULT '' AFTER member_card_id;
