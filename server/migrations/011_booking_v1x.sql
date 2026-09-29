-- 011_booking_v1x: V1.x 预约体系升级（PRD .trellis/tasks/2026-09-27-v1x-booking-redeem-ux）。
-- D19: 散客核销 —— payment/redemption.appointment_id 可空（NULL=散客直接核销）；
--      active_lock/valid_lock 生成列在 NULL 时仍为 NULL，唯一索引允许多个 NULL，语义不变。
-- D20: 模糊预约 —— appointment.slot_type ∈ {SPECIFIC, HALF_DAY}；HALF_DAY 落库窗口=半天边界。
-- D22: 闭店日历 —— (date, AM|PM) 粒度；全天闭店=两行。
-- SQLite 翻译说明：
--   * payment/redemption.appointment_id NULL 化已并入 006 建表（SQLite 不能 MODIFY COLUMN）；
--   * appointment.slot_type 已并入 005 建表（含 ck_appointment_slot_type）；
--   * 本文件仅保留闭店日历建表。

CREATE TABLE appointment_closure (
  id           CHAR(26)     NOT NULL PRIMARY KEY,
  closure_date DATE         NOT NULL,
  day_part     VARCHAR(2)   NOT NULL,
  remark       VARCHAR(200) NOT NULL DEFAULT '',
  created_at   DATETIME     NOT NULL DEFAULT (datetime('now','+8 hours')),
  CONSTRAINT uk_closure_date_part UNIQUE (closure_date, day_part),
  CONSTRAINT ck_closure_day_part CHECK (day_part IN ('AM','PM'))
);
