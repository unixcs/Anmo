-- 003_service: service_category / service
-- 价格统一整数分（§16），禁止 float。

CREATE TABLE service_category (
  id         CHAR(26)    NOT NULL PRIMARY KEY,
  name       VARCHAR(50) NOT NULL,
  sort       INT         NOT NULL DEFAULT 0,
  status     VARCHAR(20) NOT NULL DEFAULT 'ACTIVE',
  created_at DATETIME    NOT NULL DEFAULT (datetime('now','+8 hours')),
  updated_at DATETIME    NOT NULL DEFAULT (datetime('now','+8 hours'))
);

CREATE TRIGGER trg_service_category_updated_at AFTER UPDATE ON service_category
FOR EACH ROW WHEN NEW.updated_at = OLD.updated_at
BEGIN
  UPDATE service_category SET updated_at = datetime('now','+8 hours') WHERE id = NEW.id;
END;

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
  created_at       DATETIME      NOT NULL DEFAULT (datetime('now','+8 hours')),
  updated_at       DATETIME      NOT NULL DEFAULT (datetime('now','+8 hours')),
  CONSTRAINT ck_service_duration CHECK (duration_minutes > 0),
  CONSTRAINT ck_service_price CHECK (default_price >= 0),
  CONSTRAINT fk_service_category FOREIGN KEY (category_id) REFERENCES service_category (id)
);

CREATE INDEX idx_service_category ON service (category_id, status);

CREATE TRIGGER trg_service_updated_at AFTER UPDATE ON service
FOR EACH ROW WHEN NEW.updated_at = OLD.updated_at
BEGIN
  UPDATE service SET updated_at = datetime('now','+8 hours') WHERE id = NEW.id;
END;
