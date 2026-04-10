import assert from 'node:assert/strict'
import test from 'node:test'

import {
  HALL_OF_FAME_PREVIEW_QUERY_KEY,
  readHallOfFamePreviewDraft,
  saveHallOfFamePreviewDraft
} from './manage-preview.helpers'

function createMemoryStorage(): Storage {
  const values = new Map<string, string>()

  return {
    get length() {
      return values.size
    },
    clear() {
      values.clear()
    },
    getItem(key: string) {
      return values.get(key) ?? null
    },
    key(index: number) {
      return Array.from(values.keys())[index] ?? null
    },
    removeItem(key: string) {
      values.delete(key)
    },
    setItem(key: string, value: string) {
      values.set(key, value)
    }
  }
}

test('saveHallOfFamePreviewDraft persists a draft and returns a preview URL with its token', () => {
  const storage = createMemoryStorage()
  const payload: Api.HallOfFame.TempleResponse = {
    config: {
      id: 1,
      background_image: 'bg',
      canvas_width: 2560,
      canvas_height: 1440,
      created_at: '',
      updated_at: ''
    },
    cards: [
      {
        id: 7,
        name: 'Hero Alpha',
        title: 'Founder',
        description: 'Draft preview',
        character_id: 1387156123,
        badge_image: '',
        pos_x: 10,
        pos_y: 20,
        width: 320,
        height: 420,
        style_preset: 'gold',
        custom_bg_color: '',
        custom_text_color: '',
        custom_border_color: '',
        border_style: 'none',
        title_color: '',
        font_size: 18,
        z_index: 0,
        visible: true,
        created_at: '',
        updated_at: ''
      }
    ]
  }

  const previewUrl = saveHallOfFamePreviewDraft(storage, '/hall-of-fame/temple', payload)
  const parsedUrl = new URL(previewUrl, 'https://example.com')
  const previewId = parsedUrl.searchParams.get(HALL_OF_FAME_PREVIEW_QUERY_KEY)

  assert.equal(parsedUrl.pathname, '/hall-of-fame/temple')
  assert.equal(typeof previewId, 'string')
  assert.equal(previewId?.length ? true : false, true)
  assert.deepEqual(readHallOfFamePreviewDraft(storage, previewId ?? ''), payload)
})

test('readHallOfFamePreviewDraft ignores missing or invalid preview payloads', () => {
  const storage = createMemoryStorage()

  assert.equal(readHallOfFamePreviewDraft(storage, ''), null)

  storage.setItem('hall-of-fame-preview:broken', '{bad json')
  assert.equal(readHallOfFamePreviewDraft(storage, 'broken'), null)
})
