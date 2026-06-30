export interface BrowserNotificationConstructor {
  readonly permission: NotificationPermission
  requestPermission: () => Promise<NotificationPermission>
  new (title: string, options?: NotificationOptions): Notification
}

export interface BrowserNotificationClientOptions {
  getNotification?: () => BrowserNotificationConstructor | null
}

export interface BrowserNotificationPayload {
  title: string
  body: string
  tag?: string
  icon?: string
  data?: unknown
  onClick?: () => void
}

const getDefaultNotification = () => {
  if (typeof Notification === 'undefined') {
    return null
  }
  return Notification
}

export const createBrowserNotificationClient = (options: BrowserNotificationClientOptions = {}) => {
  const getNotification = options.getNotification ?? getDefaultNotification

  const isSupported = () => getNotification() !== null

  const getPermission = (): NotificationPermission => {
    const notification = getNotification()
    return notification?.permission ?? 'denied'
  }

  const requestPermission = async (): Promise<NotificationPermission> => {
    const notification = getNotification()
    if (!notification) {
      return 'denied'
    }
    return notification.requestPermission()
  }

  return {
    isSupported,
    getPermission,
    requestPermission,
    getNotification
  }
}

export type BrowserNotificationClient = ReturnType<typeof createBrowserNotificationClient>

export const sendBrowserNotification = (
  client: BrowserNotificationClient,
  payload: BrowserNotificationPayload
) => {
  const notificationConstructor = client.getNotification()
  if (!notificationConstructor || notificationConstructor.permission !== 'granted') {
    return false
  }

  try {
    const notification = new notificationConstructor(payload.title, {
      body: payload.body,
      tag: payload.tag,
      icon: payload.icon,
      data: payload.data
    })

    if (payload.onClick) {
      notification.onclick = () => {
        payload.onClick?.()
        notification.close()
      }
    }

    return true
  } catch {
    return false
  }
}
