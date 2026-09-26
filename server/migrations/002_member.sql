-- 002_member: member / member_tag / member_tag_rel
-- D3: appointment.member_id NOT NULL 的前置——member.phone 唯一，登录即创建/绑定。

CREATE TABLE member (
  id            CHAR(26)     NOT NULL PRIMARY KEY,
  member_no     VARCHAR(20)  NOT NULL,
  name          VARCHAR(50)  NOT NULL DEFAULT '',
  phone         VARCHAR(20)  NOT NULL,
  gender        VARCHAR(10)  NOT NULL DEFAULT '',
  birthday      DATE         NULL,
  status        VARCHAR(20)  NOT NULL DEFAULT 'ACTIVE',
  remark        VARCHAR(500) NOT NULL DEFAULT '',
  last_visit_at DATETIME     NULL,
  created_at    DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at    DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  UNIQUE KEY uk_member_phone (phone),
  UNIQUE KEY uk_member_no (member_no),
  KEY idx_member_status_created (status, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE member_tag (
  id         CHAR(26)    NOT NULL PRIMARY KEY,
  name       VARCHAR(30) NOT NULL,
  created_at DATETIME    NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE KEY uk_member_tag_name (name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE member_tag_rel (
  id         CHAR(26)  NOT NULL PRIMARY KEY,
  member_id  CHAR(26)  NOT NULL,
  tag_id     CHAR(26)  NOT NULL,
  created_at DATETIME  NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE KEY uk_member_tag_rel (member_id, tag_id),
  KEY idx_member_tag_rel_member (member_id),
  KEY idx_member_tag_rel_tag (tag_id),
  CONSTRAINT fk_mtr_member FOREIGN KEY (member_id) REFERENCES member (id),
  CONSTRAINT fk_mtr_tag FOREIGN KEY (tag_id) REFERENCES member_tag (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 常用标签种子（plan §13）
INSERT INTO member_tag (id, name) VALUES
  ('01J0TAG000000000000000VIP0', 'VIP'),
  ('01J0TAG000000000000000OLD0', '老客户'),
  ('01J0TAG000000000000000NECK', '肩颈'),
  ('01J0TAG000000000000000WAIS', '腰部'),
  ('01J0TAG000000000000000SENS', '敏感');
