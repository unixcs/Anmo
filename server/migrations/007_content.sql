-- 007_content: content_page_config / content_banner / content_announcement / content_system_setting
-- D16: page_config 的 JSON block 引用素材表 id，不复制正文。
-- SQLite 无 JSON 列类型：TEXT + json_valid CHECK 等价（JSON1 内建）。

CREATE TABLE content_page_config (
  id         CHAR(26)    NOT NULL PRIMARY KEY,
  page       VARCHAR(50) NOT NULL,
  blocks     TEXT        NOT NULL CHECK (json_valid(blocks)),
  updated_by CHAR(26)    NULL,
  created_at DATETIME    NOT NULL DEFAULT (datetime('now','+8 hours')),
  updated_at DATETIME    NOT NULL DEFAULT (datetime('now','+8 hours')),
  CONSTRAINT uk_content_page UNIQUE (page)
);

CREATE TRIGGER trg_content_page_config_updated_at AFTER UPDATE ON content_page_config
FOR EACH ROW WHEN NEW.updated_at = OLD.updated_at
BEGIN
  UPDATE content_page_config SET updated_at = datetime('now','+8 hours') WHERE id = NEW.id;
END;

CREATE TABLE content_banner (
  id         CHAR(26)     NOT NULL PRIMARY KEY,
  title      VARCHAR(100) NOT NULL DEFAULT '',
  image      VARCHAR(500) NOT NULL DEFAULT '',
  link       VARCHAR(500) NOT NULL DEFAULT '',
  sort       INT          NOT NULL DEFAULT 0,
  status     VARCHAR(20)  NOT NULL DEFAULT 'ACTIVE',
  created_at DATETIME     NOT NULL DEFAULT (datetime('now','+8 hours')),
  updated_at DATETIME     NOT NULL DEFAULT (datetime('now','+8 hours'))
);

CREATE TRIGGER trg_content_banner_updated_at AFTER UPDATE ON content_banner
FOR EACH ROW WHEN NEW.updated_at = OLD.updated_at
BEGIN
  UPDATE content_banner SET updated_at = datetime('now','+8 hours') WHERE id = NEW.id;
END;

CREATE TABLE content_announcement (
  id         CHAR(26)      NOT NULL PRIMARY KEY,
  title      VARCHAR(200)  NOT NULL,
  content    VARCHAR(2000) NOT NULL DEFAULT '',
  sort       INT           NOT NULL DEFAULT 0,
  status     VARCHAR(20)   NOT NULL DEFAULT 'ACTIVE',
  created_at DATETIME      NOT NULL DEFAULT (datetime('now','+8 hours')),
  updated_at DATETIME      NOT NULL DEFAULT (datetime('now','+8 hours'))
);

CREATE TRIGGER trg_content_announcement_updated_at AFTER UPDATE ON content_announcement
FOR EACH ROW WHEN NEW.updated_at = OLD.updated_at
BEGIN
  UPDATE content_announcement SET updated_at = datetime('now','+8 hours') WHERE id = NEW.id;
END;

CREATE TABLE content_system_setting (
  id           CHAR(26)     NOT NULL PRIMARY KEY,
  setting_key  VARCHAR(50)  NOT NULL,
  setting_value VARCHAR(2000) NOT NULL DEFAULT '',
  remark       VARCHAR(200) NOT NULL DEFAULT '',
  updated_by   CHAR(26)     NULL,
  updated_at   DATETIME     NOT NULL DEFAULT (datetime('now','+8 hours')),
  created_at   DATETIME     NOT NULL DEFAULT (datetime('now','+8 hours')),
  CONSTRAINT uk_setting_key UNIQUE (setting_key)
);

CREATE TRIGGER trg_content_system_setting_updated_at AFTER UPDATE ON content_system_setting
FOR EACH ROW WHEN NEW.updated_at = OLD.updated_at
BEGIN
  UPDATE content_system_setting SET updated_at = datetime('now','+8 hours') WHERE id = NEW.id;
END;
