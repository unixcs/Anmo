-- 009_sequence: sys_sequence 原子计数器（技术表，非业务实体）。
-- 业务编号（member_no/appointment_no）用它生成：UPDATE 行锁阻塞并发，
-- 提交后序列单调递增，规避 MAX+1 的未提交可见性竞态。
-- SQLite：写事务由 _txlock=immediate 全库串行化，单写者模型下等价成立。

CREATE TABLE sys_sequence (
  name  VARCHAR(50) NOT NULL PRIMARY KEY,
  value BIGINT      NOT NULL DEFAULT 0
);
