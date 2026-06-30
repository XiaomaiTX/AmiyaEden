import assert from 'node:assert/strict'
import test from 'node:test'

import {
  createBrowserNotificationClient,
  sendBrowserNotification,
  type BrowserNotificationConstructor
} from './browserNotification'

type ShownNotification = {
  title: string
  options?: NotificationOptions
  instance: Notification
}

const createFakeNotificationConstructor = (
  initialPermission: NotificationPermission,
  shown: ShownNotification[],
  requestResult: NotificationPermission = initialPermission
) => {
  class FakeNotification {
    static permission = initialPermission

    static async requestPermission() {
      FakeNotification.permission = requestResult
      return requestResult
    }

    onclick: ((event: Event) => void) | null = null
    closed = false

    constructor(
      public title: string,
      public options?: NotificationOptions
    ) {
      shown.push({
        title,
        options,
        instance: this as unknown as Notification
      })
    }

    close() {
      this.closed = true
    }
  }

  return FakeNotification as unknown as BrowserNotificationConstructor
}

test('sendBrowserNotification sends when permission is granted and wires click callback', () => {
  const shown: ShownNotification[] = []
  const notificationConstructor = createFakeNotificationConstructor('granted', shown)
  const client = createBrowserNotificationClient({ getNotification: () => notificationConstructor })
  let clicked = false

  const sent = sendBrowserNotification(client, {
    title: 'Timeout',
    body: 'System A is overdue',
    tag: 'galaxy-registry-timeout-1',
    onClick: () => {
      clicked = true
    }
  })

  assert.equal(sent, true)
  assert.equal(shown.length, 1)
  assert.equal(shown[0].title, 'Timeout')
  assert.equal(shown[0].options?.body, 'System A is overdue')
  assert.equal(shown[0].options?.tag, 'galaxy-registry-timeout-1')

  shown[0].instance.onclick?.(new Event('click'))
  assert.equal(clicked, true)
})

test('sendBrowserNotification skips denied permission', () => {
  const shown: ShownNotification[] = []
  const notificationConstructor = createFakeNotificationConstructor('denied', shown)
  const client = createBrowserNotificationClient({ getNotification: () => notificationConstructor })

  assert.equal(
    sendBrowserNotification(client, {
      title: 'Timeout',
      body: 'System A is overdue'
    }),
    false
  )
  assert.equal(shown.length, 0)
})

test('browser notification client can request default permission before sending', async () => {
  const shown: ShownNotification[] = []
  const notificationConstructor = createFakeNotificationConstructor('default', shown, 'granted')
  const client = createBrowserNotificationClient({ getNotification: () => notificationConstructor })

  assert.equal(client.getPermission(), 'default')
  assert.equal(await client.requestPermission(), 'granted')
  assert.equal(client.getPermission(), 'granted')
  assert.equal(
    sendBrowserNotification(client, {
      title: 'Timeout',
      body: 'System A is overdue'
    }),
    true
  )
})
