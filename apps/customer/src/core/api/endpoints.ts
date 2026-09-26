/**
 * core/api — endpoint catalog + typed wrappers.
 */
import { http } from './http'
import type { Appointment, Member, MemberCard, Page, ServiceCategory, ServiceItem } from '../models/models'

export const endpoints = {
  sendSms: '/api/auth/sms/send',
  verifySms: '/api/auth/sms/verify',
  catalog: '/api/services',
  myProfile: '/api/me/profile',
  myCards: '/api/me/cards',
  appointments: '/api/appointments',
  appointment: (id: string) => `/api/appointments/${id}`,
  cancel: (id: string) => `/api/appointments/${id}/cancel`,
  reschedule: (id: string) => `/api/appointments/${id}/reschedule`,
}

export interface Catalog {
  categories: ServiceCategory[]
  services: ServiceItem[]
}

export const api = {
  sendSms: (phone: string) => http.post<{ sent: boolean }>(endpoints.sendSms, { phone }),
  verifySms: (phone: string, code: string) =>
    http.post<{ token: string; member_id: string }>(endpoints.verifySms, { phone, code }),
  catalog: () => http.get<Catalog>(endpoints.catalog),
  myProfile: () => http.get<{ member: Member; tags: { id: string; name: string }[] }>(endpoints.myProfile),
  updateProfile: (patch: Partial<Pick<Member, 'name' | 'gender' | 'birthday'>>) =>
    http.put<{ member: Member }>(endpoints.myProfile, patch),
  myCards: () => http.get<MemberCard[]>(endpoints.myCards),
  myAppointments: (status: string) =>
    http.get<Page<Appointment>>(`${endpoints.appointments}?status=${status}&page=1&per_page=50`),
  appointment: (id: string) =>
    http.get<{ appointment: Appointment; services: { service_name_snapshot: string; duration_minutes_snapshot: number; price_snapshot: number }[] }>(endpoints.appointment(id)),
  createAppointment: (serviceId: string, startTime: string, note: string) =>
    http.post<Appointment>(endpoints.appointments, { service_id: serviceId, start_time: startTime, note }),
  cancelAppointment: (id: string) => http.put<Appointment>(endpoints.cancel(id)),
  rescheduleAppointment: (id: string, startTime: string) =>
    http.put<Appointment>(endpoints.reschedule(id), { start_time: startTime }),
}
