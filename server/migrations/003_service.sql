-- 003_service: service_category / service
-- 价格统一整数分（§16），禁止 float。

CREATE TABLE service_category (
  id         CHAR(26)    NOT NULL PRIMARY KEY,
  name       VARCHAR(50) NOT NULL,
  sort       INT         NOT NULL DEFAULT 0,
  status     VARCHAR(20) NOT NULL DEFAULT 'ACTIVE',
  created_at DATETIME    NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME    NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE service (
  id               CHAR(26)      NOT NULL PRIMARY KEY,
  category_id      CHAR(26)      NOT NULL,
  name             VARCHAR(100)  NOT NULL,
  description      VARCHAR(1000) NOT NULL DEFAULT '',
  duration_minutes INT           NOT NULL,
  default_price    BIGINT        NOT NULL,
  cover_image      VARCHAR(500)  NOT NULL DEFAULT '',
  status           VARCHAR(20)   NOT NULL DEFAULT 'ACTIVE',
  sort             INT           NOT NULL DEFAULT 0,
  created_at       DATETIME      NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at       DATETIME      NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  KEY idx_service_category (category_id, status),
  CONSTRAINT ck_service_duration CHECK (duration_minutes > 0),
  CONSTRAINT ck_service_price CHECK (default_price >= 0),
  CONSTRAINT fk_service_category FOREIGN KEY (category_id) REFERENCES service_category (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
