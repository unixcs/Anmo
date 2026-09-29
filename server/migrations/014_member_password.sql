-- 014: 密码体系（plan §二）：phone 可空（纯微信会员）+ password_hash（bcrypt，NULL=未设置）。
-- SQLite 不支持 ALTER COLUMN，重建表；UNIQUE(phone) 对 NULL 天然放行多个。
--
-- FK 包裹说明（与 design §1 的实现差异，实测修正，语义不变）：本项目迁移执行器
-- 把每个 migration 文件包在一个事务里跑（database.Migrate），而 SQLite 规定
-- PRAGMA foreign_keys 在事务内是 no-op，无法按 design 原文用 foreign_keys=OFF
-- 包裹 DROP TABLE member（member_tag_rel / member_card / appointment 三个子表
-- 引用它）。实测可用且等价的序列（modernc.org/sqlite 验证）：
--   PRAGMA defer_foreign_keys = ON（事务内可设，COMMIT 后自动复位）
--   → DROP member 的隐式 DELETE 触发的子表 FK 违例被推迟
--   → 按同名重建 member 并把行插回去（defer 违例计数随父行 INSERT 清除）
--   → COMMIT 复查全部满足，子表 DDL 不改写、FK 约束持续生效。
-- design 原文的 CREATE member_new + RENAME 方案在事务内不可行：RENAME 不清
-- defer 违例计数，且 modernc 内核的 RENAME 会跟随改写子表 FK 引用目标表名。
PRAGMA defer_foreign_keys = ON;
CREATE TABLE member_new (
  id            CHAR(26)     NOT NULL PRIMARY KEY,
  member_no     VARCHAR(20)  NOT NULL,
  name          VARCHAR(50)  NOT NULL DEFAULT '',
  phone         VARCHAR(20)  NULL,
  gender        VARCHAR(10)  NOT NULL DEFAULT '',
  birthday      DATE         NULL,
  status        VARCHAR(20)  NOT NULL DEFAULT 'ACTIVE',
  remark        VARCHAR(500) NOT NULL DEFAULT '',
  last_visit_at DATETIME     NULL,
  password_hash VARCHAR(100) NULL,
  wx_openid     VARCHAR(64)  NULL,
  created_at    DATETIME     NOT NULL DEFAULT (datetime('now','+8 hours')),
  updated_at    DATETIME     NOT NULL DEFAULT (datetime('now','+8 hours')),
  CONSTRAINT uk_member_phone UNIQUE (phone),
  CONSTRAINT uk_member_no UNIQUE (member_no)
);
INSERT INTO member_new (id, member_no, name, phone, gender, birthday, status, remark,
                        last_visit_at, password_hash, wx_openid, created_at, updated_at)
  SELECT id, member_no, name, phone, gender, birthday, status, remark,
         last_visit_at, NULL, wx_openid, created_at, updated_at FROM member;
DROP TABLE member;
CREATE TABLE member (
  id            CHAR(26)     NOT NULL PRIMARY KEY,
  member_no     VARCHAR(20)  NOT NULL,
  name          VARCHAR(50)  NOT NULL DEFAULT '',
  phone         VARCHAR(20)  NULL,
  gender        VARCHAR(10)  NOT NULL DEFAULT '',
  birthday      DATE         NULL,
  status        VARCHAR(20)  NOT NULL DEFAULT 'ACTIVE',
  remark        VARCHAR(500) NOT NULL DEFAULT '',
  last_visit_at DATETIME     NULL,
  password_hash VARCHAR(100) NULL,
  wx_openid     VARCHAR(64)  NULL,
  created_at    DATETIME     NOT NULL DEFAULT (datetime('now','+8 hours')),
  updated_at    DATETIME     NOT NULL DEFAULT (datetime('now','+8 hours')),
  CONSTRAINT uk_member_phone UNIQUE (phone),
  CONSTRAINT uk_member_no UNIQUE (member_no)
);
INSERT INTO member (id, member_no, name, phone, gender, birthday, status, remark,
                    last_visit_at, password_hash, wx_openid, created_at, updated_at)
  SELECT id, member_no, name, phone, gender, birthday, status, remark,
         last_visit_at, password_hash, wx_openid, created_at, updated_at FROM member_new;
DROP TABLE member_new;
CREATE INDEX idx_member_status_created ON member (status, created_at);
CREATE UNIQUE INDEX uk_member_wx_openid ON member (wx_openid);
CREATE TRIGGER trg_member_updated_at AFTER UPDATE ON member
  FOR EACH ROW WHEN NEW.updated_at = OLD.updated_at
  BEGIN UPDATE member SET updated_at = datetime('now','+8 hours') WHERE id = NEW.id; END;
