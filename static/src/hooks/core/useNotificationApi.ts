import { fetchNotifications, fetchUnreadCount, markAllAsRead, markAsRead } from '@/api/notification'

export function useNotificationApi() {
  return {
    fetchNotifications,
    fetchUnreadCount,
    markAllAsRead,
    markAsRead
  }
}
