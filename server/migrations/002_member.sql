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
  created_at    DATETIME     NOT NULL DEFAULT (datetime('now','+8 hours')),
  updated_at    DATETIME     NOT NULL DEFAULT (datetime('now','+8 hours')),
  CONSTRAINT uk_member_phone UNIQUE (phone),
  CONSTRAINT uk_member_no UNIQUE (member_no)
);

CREATE INDEX idx_member_status_created ON member (status, created_at);

CREATE TRIGGER trg_member_updated_at AFTER UPDATE ON member
FOR EACH ROW WHEN NEW.updated_at = OLD.updated_at
BEGIN
  UPDATE member SET updated_at = datetime('now','+8 hours') WHERE id = NEW.id;
END;

CREATE TABLE member_tag (
  id         CHAR(26)    NOT NULL PRIMARY KEY,
  name       VARCHAR(30) NOT NULL,
  created_at DATETIME    NOT NULL DEFAULT (datetime('now','+8 hours')),
  CONSTRAINT uk_member_tag_name UNIQUE (name)
);

CREATE TABLE member_tag_rel (
  id         CHAR(26)  NOT NULL PRIMARY KEY,
  member_id  CHAR(26)  NOT NULL,
  tag_id     CHAR(26)  NOT NULL,
  created_at DATETIME  NOT NULL DEFAULT (datetime('now','+8 hours')),
  CONSTRAINT uk_member_tag_rel UNIQUE (member_id, tag_id),
  CONSTRAINT fk_mtr_member FOREIGN KEY (member_id) REFERENCES member (id),
  CONSTRAINT fk_mtr_tag FOREIGN KEY (tag_id) REFERENCES member_tag (id)
);

CREATE INDEX idx_member_tag_rel_member ON member_tag_rel (member_id);
CREATE INDEX idx_member_tag_rel_tag ON member_tag_rel (tag_id);

-- 常用标签种子（plan §13）
INSERT INTO member_tag (id, name) VALUES
  ('01J0TAG000000000000000VIP0', 'VIP'),
  ('01J0TAG000000000000000OLD0', '老客户'),
  ('01J0TAG000000000000000NECK', '肩颈'),
  ('01J0TAG000000000000000WAIS', '腰部'),
  ('01J0TAG000000000000000SENS', '敏感');
