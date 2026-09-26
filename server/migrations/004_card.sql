-- 004_card: card_template / card_service_rule / member_card / card_transaction
-- D4: card_service_rule 挂模板级；D10: 续卡=再发新卡(复用 ISSUE)；D11: type 仅枚举不产生逻辑；
-- D12: 作废仅置状态不写次数流水；D13: remaining_count=0 由核销事务置 USED_UP，EXPIRED 惰性+sweep。
-- 余额不是唯一真相：card_transaction 才是历史（§125）。

CREATE TABLE card_template (
  id            CHAR(26)     NOT NULL PRIMARY KEY,
  name          VARCHAR(100) NOT NULL,
  type          VARCHAR(20)  NOT NULL DEFAULT 'COUNT',
  total_count   INT          NOT NULL,
  validity_type VARCHAR(20)  NOT NULL DEFAULT 'PERMANENT',
  valid_from    DATE         NULL,
  valid_until   DATE         NULL,
  price         BIGINT       NOT NULL DEFAULT 0,
  status        VARCHAR(20)  NOT NULL DEFAULT 'ACTIVE',
  created_at    DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at    DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  UNIQUE KEY uk_card_template_name (name),
  CONSTRAINT ck_card_template_type CHECK (type IN ('COUNT','ACTIVITY')),
  CONSTRAINT ck_card_template_validity CHECK (validity_type IN ('PERMANENT','FIXED')),
  CONSTRAINT ck_card_template_count CHECK (total_count > 0),
  CONSTRAINT ck_card_template_price CHECK (price >= 0)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE card_service_rule (
  id              CHAR(26) NOT NULL PRIMARY KEY,
  card_template_id CHAR(26) NOT NULL,
  service_id      CHAR(26) NOT NULL,
  created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE KEY uk_card_service_rule (card_template_id, service_id),
  CONSTRAINT fk_csr_template FOREIGN KEY (card_template_id) REFERENCES card_template (id),
  CONSTRAINT fk_csr_service FOREIGN KEY (service_id) REFERENCES service (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE member_card (
  id              CHAR(26)    NOT NULL PRIMARY KEY,
  member_id       CHAR(26)    NOT NULL,
  card_template_id CHAR(26)   NOT NULL,
  total_count     INT         NOT NULL,
  remaining_count INT         NOT NULL,
  valid_from      DATE        NOT NULL,
  valid_until     DATE        NULL,
  status          VARCHAR(20) NOT NULL DEFAULT 'ACTIVE',
  issued_at       DATETIME    NOT NULL DEFAULT CURRENT_TIMESTAMP,
  issued_by       CHAR(26)    NULL,
  created_at      DATETIME    NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at      DATETIME    NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  KEY idx_member_card_member (member_id, status),
  CONSTRAINT ck_member_card_remaining CHECK (remaining_count >= 0),
  CONSTRAINT ck_member_card_total CHECK (total_count > 0),
  CONSTRAINT ck_member_card_status CHECK (status IN ('ACTIVE','USED_UP','EXPIRED','CANCELLED')),
  CONSTRAINT fk_mc_member FOREIGN KEY (member_id) REFERENCES member (id),
  CONSTRAINT fk_mc_template FOREIGN KEY (card_template_id) REFERENCES card_template (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE card_transaction (
  id             CHAR(26)     NOT NULL PRIMARY KEY,
  member_card_id CHAR(26)     NOT NULL,
  member_id      CHAR(26)     NOT NULL,
  type           VARCHAR(20)  NOT NULL,
  quantity       INT          NOT NULL,
  before_count   INT          NOT NULL,
  after_count    INT          NOT NULL,
  reference_type VARCHAR(30)  NOT NULL DEFAULT '',
  reference_id   CHAR(26)     NULL,
  remark         VARCHAR(500) NOT NULL DEFAULT '',
  operator_id    CHAR(26)     NULL,
  created_at     DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
  KEY idx_card_txn_card (member_card_id, created_at),
  CONSTRAINT ck_card_txn_type CHECK (type IN ('ISSUE','REDEEM','REVERSAL','ADJUSTMENT','REFUND')),
  CONSTRAINT fk_ct_card FOREIGN KEY (member_card_id) REFERENCES member_card (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
