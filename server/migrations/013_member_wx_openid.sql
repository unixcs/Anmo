-- V2 微信小程序（plan §11：微信登录→绑定手机号→同一个 member_id）
-- member.wx_openid：NULL 可重复（未绑定），非 NULL 唯一（一个 openid 只绑一个 member）。
-- 绑定链路见 identity 模块：/api/auth/wx/login → bind_ticket → 顾客短信登录后 /api/auth/wx/bind。

ALTER TABLE member ADD COLUMN wx_openid VARCHAR(64) NULL;

CREATE UNIQUE INDEX uk_member_wx_openid ON member (wx_openid);
