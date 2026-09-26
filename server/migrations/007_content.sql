-- 007_content: content_page_config / content_banner / content_announcement / content_system_setting
-- D16: page_config 的 JSON block 引用素材表 id，不复制正文。

CREATE TABLE content_page_config (
  id         CHAR(26)    NOT NULL PRIMARY KEY,
  page       VARCHAR(50) NOT NULL,
  blocks     JSON        NOT NULL,
  updated_by CHAR(26)    NULL,
  created_at DATETIME    NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME    NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  UNIQUE KEY uk_content_page (page)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE content_banner (
  id         CHAR(26)     NOT NULL PRIMARY KEY,
  title      VARCHAR(100) NOT NULL DEFAULT '',
  image      VARCHAR(500) NOT NULL DEFAULT '',
  link       VARCHAR(500) NOT NULL DEFAULT '',
  sort       INT          NOT NULL DEFAULT 0,
  status     VARCHAR(20)  NOT NULL DEFAULT 'ACTIVE',
  created_at DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE content_announcement (
  id         CHAR(26)      NOT NULL PRIMARY KEY,
  title      VARCHAR(200)  NOT NULL,
  content    VARCHAR(2000) NOT NULL DEFAULT '',
  sort       INT           NOT NULL DEFAULT 0,
  status     VARCHAR(20)   NOT NULL DEFAULT 'ACTIVE',
  created_at DATETIME      NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME      NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE content_system_setting (
  id           CHAR(26)     NOT NULL PRIMARY KEY,
  setting_key  VARCHAR(50)  NOT NULL,
  setting_value VARCHAR(2000) NOT NULL DEFAULT '',
  remark       VARCHAR(200) NOT NULL DEFAULT '',
  updated_by   CHAR(26)     NULL,
  updated_at   DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  created_at   DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE KEY uk_setting_key (setting_key)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
