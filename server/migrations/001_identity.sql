-- 001_identity: identity_user / identity_role / identity_permission
-- D2: 角色由 identity_user.role 枚举实现；role/permission 两表为静态种子，不做动态授权。

CREATE TABLE identity_role (
  id          CHAR(26)     NOT NULL PRIMARY KEY,
  code        VARCHAR(20)  NOT NULL,
  name        VARCHAR(50)  NOT NULL,
  remark      VARCHAR(200) NOT NULL DEFAULT '',
  created_at  DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE KEY uk_identity_role_code (code)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE identity_permission (
  id          CHAR(26)     NOT NULL PRIMARY KEY,
  code        VARCHAR(50)  NOT NULL,
  name        VARCHAR(50)  NOT NULL,
  remark      VARCHAR(200) NOT NULL DEFAULT '',
  created_at  DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE KEY uk_identity_permission_code (code)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE identity_user (
  id            CHAR(26)     NOT NULL PRIMARY KEY,
  phone         VARCHAR(20)  NOT NULL,
  password_hash VARCHAR(100) NOT NULL,
  name          VARCHAR(50)  NOT NULL DEFAULT '',
  role          VARCHAR(20)  NOT NULL DEFAULT 'OPERATOR',
  status        VARCHAR(20)  NOT NULL DEFAULT 'ACTIVE',
  created_at    DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at    DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  UNIQUE KEY uk_identity_user_phone (phone),
  CONSTRAINT ck_identity_user_role CHECK (role IN ('OWNER','OPERATOR'))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

INSERT INTO identity_role (id, code, name, remark) VALUES
  ('01J0ROLE0000000000000OWNR', 'OWNER',    '老板',   '店主，全部权限'),
  ('01J0ROLE0000000000000OPER', 'OPERATOR', '运营',   '家人协助后台，V1 权限与 OWNER 相同');

INSERT INTO identity_permission (id, code, name) VALUES
  ('01J0PERM0000000000000MBR0', 'member:manage',     '会员管理'),
  ('01J0PERM0000000000000SRV0', 'service:manage',    '服务项目管理'),
  ('01J0PERM0000000000000CRD0', 'card:manage',       '会员卡管理'),
  ('01J0PERM0000000000000APT0', 'appointment:manage','预约管理'),
  ('01J0PERM0000000000000TXN0', 'transaction:manage','结算/收款/核销'),
  ('01J0PERM0000000000000CNT0', 'content:manage',    '内容管理'),
  ('01J0PERM0000000000000OPS0', 'ops:manage',        '日志与洞察');
