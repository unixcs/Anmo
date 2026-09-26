# 顾客端会员卡端点

GET /api/me/cards 由 card 模块提供（见 internal/modules/card/handler_customer.go），
返回当前顾客（token member_id）的会员卡列表，用于 CardsPage。
