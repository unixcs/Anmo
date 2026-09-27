-- 011_booking_v1x: V1.x 预约体系升级（PRD .trellis/tasks/2026-09-27-v1x-booking-redeem-ux）。
-- D19: 散客核销 —— payment/redemption.appointment_id 可空（NULL=散客直接核销）；
--      active_lock/valid_lock 生成列在 NULL 时仍为 NULL，唯一索引允许多个 NULL，语义不变。
-- D20: 模糊预约 —— appointment.slot_type ∈ {SPECIFIC, HALF_DAY}；HALF_DAY 落库窗口=半天边界。
-- D22: 闭店日历 —— (date, AM|PM) 粒度；全天闭店=两行。

ALTER TABLE payment    MODIFY appointment_id CHAR(26) NULL;
ALTER TABLE redemption MODIFY appointment_id CHAR(26) NULL;

ALTER TABLE appointment
  ADD COLUMN slot_type VARCHAR(12) NOT NULL DEFAULT 'SPECIFIC',
  ADD CONSTRAINT ck_appointment_slot_type CHECK (slot_type IN ('SPECIFIC','HALF_DAY'));

CREATE TABLE appointment_closure (
  id           CHAR(26)        NOT NULL PRIMARY KEY,
  closure_date DATE            NOT NULL,
  day_part     ENUM('AM','PM') NOT NULL,
  remark       VARCHAR(200)    NOT NULL DEFAULT '',
  created_at   DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE KEY uk_closure_date_part (closure_date, day_part)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
