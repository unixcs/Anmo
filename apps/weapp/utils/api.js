// api.js — 端点目录 + 包装（与顾客 H5 core/api/endpoints.ts 完全同口径）。
const http = require('./request')

const endpoints = {
  sendSms: '/api/auth/sms/send',
  verifySms: '/api/auth/sms/verify',
  wxLogin: '/api/auth/wx/login',
  wxBind: '/api/auth/wx/bind',
  home: '/api/home',
  publicSettings: '/api/settings',
  storeStatus: '/api/store/status',
  catalog: '/api/services',
  myProfile: '/api/me/profile',
  myCards: '/api/me/cards',
  myCardTransactions: (id) => '/api/me/cards/' + id + '/transactions',
  appointments: '/api/appointments',
  bookingOptions: '/api/booking-options',
  appointment: (id) => '/api/appointments/' + id,
  cancel: (id) => '/api/appointments/' + id + '/cancel',
  reschedule: (id) => '/api/appointments/' + id + '/reschedule',
}

const api = {
  // ---- 登录（V2 §11：微信登录→绑定手机号→同一个 member_id）----
  wxLogin: (code) => http.post(endpoints.wxLogin, { code }),
  wxBind: (bindTicket) => http.post(endpoints.wxBind, { bind_ticket: bindTicket }),
  sendSms: (phone) => http.post(endpoints.sendSms, { phone }),
  verifySms: (phone, code) => http.post(endpoints.verifySms, { phone, code }),

  // ---- 内容 ----
  home: () => http.get(endpoints.home),
  publicSettings: () => http.get(endpoints.publicSettings),
  storeStatus: () => http.get(endpoints.storeStatus),
  catalog: () => http.get(endpoints.catalog),

  // ---- 我的 ----
  myProfile: () => http.get(endpoints.myProfile),
  updateProfile: (patch) => http.put(endpoints.myProfile, patch),
  myCards: () => http.get(endpoints.myCards),
  myCardTransactions: (cardId) => http.get(endpoints.myCardTransactions(cardId)),

  // ---- 预约（D20：具体时间或模糊半天二选一）----
  myAppointments: (status) =>
    http.get(
      endpoints.appointments + '?status=' + (status || '') + '&page=1&per_page=50',
    ),
  appointment: (id) => http.get(endpoints.appointment(id)),
  bookingOptions: (date) => http.get(endpoints.bookingOptions + '?date=' + encodeURIComponent(date)),
  createAppointment: (serviceId, target, note) =>
    http.post(endpoints.appointments, Object.assign({ service_id: serviceId }, target, { note })),
  cancelAppointment: (id) => http.put(endpoints.cancel(id)),
  rescheduleAppointment: (id, target) => http.put(endpoints.reschedule(id), target),
}

module.exports = { api }
