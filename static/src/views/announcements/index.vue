<template>
  <div class="announcement-board">
    <header class="announcement-masthead">
      <div class="announcement-heading">
        <span class="announcement-mark"><ArtSvgIcon icon="ri:megaphone-line" /></span>
        <h1>{{ t('announcement.title') }}</h1>
      </div>
      <ElButton
        v-if="manage"
        class="announcement-publish"
        :disabled="busy"
        @click="startEdit(true)"
      >
        {{ t('announcement.publish') }}
      </ElButton>
    </header>
    <p v-if="error" role="alert">{{ t('announcement.error') }}</p>
    <div class="announcement-layout">
      <aside class="announcement-index">
        <div class="announcement-index-heading">
          <h2><ArtSvgIcon icon="ri:file-text-line" />{{ t('announcement.archive') }}</h2>
          <span>{{ total }}</span>
        </div>
        <p v-if="!rows.length" class="announcement-empty">{{ t('announcement.empty') }}</p>
        <div class="announcement-entries">
          <button
            v-for="row in rows"
            :key="row.id"
            :disabled="busy"
            class="announcement-entry"
            :class="{
              'is-unread': row.unread && !row.hidden,
              'is-selected': selected?.id === row.id
            }"
            :aria-current="selected?.id === row.id ? 'true' : undefined"
            @click="open(row)"
          >
            <div class="announcement-entry-heading">
              <span class="announcement-entry-title">
                <span
                  v-if="row.unread && !row.hidden"
                  class="announcement-unread-dot"
                  aria-hidden="true"
                />
                <span class="announcement-title-text">{{ row.title }}</span>
              </span>
              <span v-if="row.hidden" class="announcement-status is-hidden">{{
                t('announcement.hidden')
              }}</span>
              <span v-else-if="row.unread" class="announcement-status">{{
                t('announcement.unread')
              }}</span>
            </div>
            <span class="announcement-entry-author">
              <span
                class="announcement-nickname"
                :title="row.author_nickname || t('announcement.unknownAuthor')"
                >{{ row.author_nickname || t('announcement.unknownAuthor') }}</span
              >
              <span
                v-if="row.author_character_name"
                class="announcement-character"
                :title="row.author_character_name"
                >{{ row.author_character_name }}</span
              >
            </span>
            <time class="announcement-entry-time" :datetime="row.created_at">{{
              new Date(row.created_at).toLocaleString()
            }}</time>
          </button>
        </div>
        <ElPagination
          class="announcement-pagination"
          v-model:current-page="page"
          :page-size="10"
          :total="total"
          layout="prev, pager, next"
          @current-change="reload()"
        />
      </aside>
      <section class="announcement-reader" :aria-busy="busy">
        <form v-if="editing" :key="editorKey" class="announcement-editor" @submit.prevent="save">
          <label>{{ t('announcement.subject') }}<ElInput v-model="title" maxlength="200" /></label>
          <p>{{ t('announcement.imageHint') }}</p>
          <Toolbar :editor="editorRef" :default-config="toolbarConfig" mode="simple" />
          <Editor
            v-model="body"
            :default-config="editorConfig"
            mode="simple"
            class="editor"
            @on-created="created"
          />
          <ElCheckbox v-model="hidden">{{ t('announcement.hidden') }}</ElCheckbox>
          <ElCheckbox v-if="selected" v-model="remind">{{ t('announcement.remind') }}</ElCheckbox>
          <footer
            ><ElButton native-type="submit" :disabled="busy">{{ t('announcement.save') }}</ElButton
            ><ElButton :disabled="busy" @click="editing = false">{{
              t('announcement.cancel')
            }}</ElButton></footer
          >
        </form>
        <template v-else-if="selected">
          <header class="announcement-article-heading">
            <h2>{{ selected.title }}</h2>
            <ElButton v-if="manage" :disabled="busy" @click="startEdit(false)">{{
              t('announcement.edit')
            }}</ElButton>
          </header>
          <div class="announcement-byline">
            <div class="announcement-author">
              <span class="announcement-avatar">
                <img
                  v-if="selected.author_character_id > 0"
                  :src="buildEveCharacterPortraitUrl(selected.author_character_id, 64)"
                  alt=""
                />
                <ArtSvgIcon v-else icon="ri:user-line" />
              </span>
              <div class="announcement-author-names">
                <span class="announcement-author-label">{{ t('announcement.author') }}</span>
                <span class="announcement-author-identity">
                  <strong
                    class="announcement-nickname"
                    :title="selected.author_nickname || t('announcement.unknownAuthor')"
                    >{{ selected.author_nickname || t('announcement.unknownAuthor') }}</strong
                  >
                  <span
                    v-if="selected.author_character_name"
                    class="announcement-character"
                    :title="selected.author_character_name"
                    >{{ selected.author_character_name }}</span
                  >
                </span>
              </div>
            </div>
            <time class="announcement-published" :datetime="selected.created_at">
              <ArtSvgIcon icon="ri:time-line" />
              <span
                >{{ t('announcement.publishedAt')
                }}<strong>{{ new Date(selected.created_at).toLocaleString() }}</strong></span
              >
            </time>
          </div>
          <div class="content announcement-body" v-html="selected.body_html" />
        </template>
        <div v-else class="announcement-reader-empty"
          ><ArtSvgIcon icon="ri:megaphone-line" /><p>{{
            t(rows.length ? 'announcement.select' : 'announcement.empty')
          }}</p></div
        >
      </section>
    </div>
  </div>
</template>
<script setup lang="ts">
  import { ref, computed, shallowRef, onMounted, onBeforeUnmount, nextTick } from 'vue'
  import { useI18n } from 'vue-i18n'
  import { Editor, Toolbar } from '@wangeditor/editor-for-vue'
  import type { IDomEditor, IEditorConfig } from '@wangeditor/editor'
  import '@wangeditor/editor/dist/css/style.css'
  import { ElMessage } from 'element-plus'
  import {
    listAnnouncements,
    getAnnouncement,
    createAnnouncement,
    updateAnnouncement,
    readAnnouncement,
    type Announcement
  } from '@/api/generated/client'
  import { unwrapContract } from '@/api/response'
  import { useUserStore } from '@/store/modules/user'
  import { useBadgeStore } from '@/store/modules/badge'
  import { buildEveCharacterPortraitUrl } from '@/utils/eve-image'
  defineOptions({ name: 'Announcements' })
  type AnnouncementRow = Announcement & { unread?: boolean }
  const MAX_IMAGES = 6
  const { t } = useI18n()
  const user = useUserStore()
  const manage = computed(() =>
    (user.info.roles ?? []).some((r) => ['admin', 'super_admin', 'senior_fc'].includes(r))
  )
  const rows = ref<AnnouncementRow[]>([]),
    selected = ref<Announcement | null>(null)
  const page = ref(1),
    total = ref(0),
    busy = ref(false),
    error = ref(false),
    editing = ref(false)
  const title = ref(''),
    body = ref(''),
    hidden = ref(false),
    remind = ref(false)
  const editorRef = shallowRef<IDomEditor>()
  const editorKey = ref(0)
  let request = 0
  let listRequest = 0
  const toolbarConfig = {
    toolbarKeys: [
      'bold',
      'italic',
      'underline',
      'bulletedList',
      'numberedList',
      'blockquote',
      'uploadImage',
      'undo',
      'redo'
    ]
  }
  const editorConfig: Partial<IEditorConfig> = {
    MENU_CONF: {
      uploadImage: {
        maxFileSize: 8 * 1024 * 1024,
        maxNumberOfFiles: MAX_IMAGES,
        allowedFileTypes: ['image/png', 'image/jpeg'],
        customUpload(file: File, insert: (url: string, alt: string, href: string) => void) {
          const reader = new FileReader()
          // maxNumberOfFiles only limits one batch; the server limit applies to the whole post.
          reader.onload = () => {
            if ((editorRef.value?.getElemsByType('image').length ?? 0) >= MAX_IMAGES) {
              ElMessage.error(t('announcement.tooManyImages', { count: MAX_IMAGES }))
              return
            }
            insert(String(reader.result), file.name, '')
          }
          reader.readAsDataURL(file)
        }
      }
    }
  }
  function created(editor: IDomEditor) {
    editorRef.value = editor
  }
  async function reload(openNewest = false) {
    const token = request
    // Fast page switching can resolve out of order; only the latest list request may apply.
    const seq = ++listRequest
    error.value = false
    try {
      const data = await unwrapContract(listAnnouncements({ page: page.value }))
      if (seq !== listRequest) return
      rows.value = data.list
      total.value = data.total
      // On entry show the newest post, unless the user already opened one or started publishing.
      if (openNewest && token === request && data.list[0]) void open(data.list[0])
      await useBadgeStore().fetchBadgeCounts()
    } catch {
      if (seq === listRequest) error.value = true
    }
  }
  async function open(row: AnnouncementRow) {
    const token = ++request
    busy.value = true
    error.value = false
    editing.value = false
    let data: Announcement
    try {
      data = await unwrapContract(getAnnouncement(row.id))
    } catch {
      if (token === request) {
        error.value = true
        busy.value = false
      }
      return
    }
    if (token !== request) return
    selected.value = data
    busy.value = false
    // Each request costs a full round trip, so acknowledgement runs in the background without
    // locking the list, and is skipped when the displayed version is already read.
    if (data.hidden || (!row.unread && data.version === row.version)) return
    await nextTick()
    if (token !== request) return
    try {
      await unwrapContract(readAnnouncement(data.id, { version: data.version }))
      row.unread = false
      row.version = data.version
      await useBadgeStore().fetchBadgeCounts()
    } catch {
      if (token === request) error.value = true
    }
  }
  function startEdit(create: boolean) {
    editorRef.value?.destroy()
    editorRef.value = undefined
    if (create) {
      request++
      selected.value = null
    }
    title.value = selected.value?.title ?? ''
    body.value = selected.value?.body_html ?? ''
    hidden.value = selected.value?.hidden ?? false
    remind.value = false
    editorKey.value++
    editing.value = true
  }
  async function save() {
    busy.value = true
    error.value = false
    try {
      const input = {
        title: title.value,
        body_html: body.value,
        hidden: hidden.value,
        remind: remind.value,
        revision: selected.value?.revision ?? 0
      }
      const id = selected.value?.id
      selected.value = await unwrapContract(
        id ? updateAnnouncement(id, input) : createAnnouncement(input)
      )
      editing.value = false
      await reload()
    } catch {
      error.value = true
    } finally {
      busy.value = false
    }
  }
  onMounted(() => reload(true))
  onBeforeUnmount(() => {
    request++
    editorRef.value?.destroy()
  })
</script>
<style scoped src="./announcements.css" />
<style scoped>
  footer {
    display: flex;
    gap: 12px;
    margin-top: 20px;
  }
  .announcement-editor > label {
    display: block;
    margin-bottom: 16px;
  }
  .editor {
    height: 350px;
    overflow-y: auto;
    border: 1px solid var(--el-border-color);
  }
  .content {
    margin-top: 16px;
    padding-top: 16px;
    border-top: 1px solid var(--el-border-color-lighter);
    font-size: 15px;
    line-height: 1.75;
    color: var(--el-text-color-regular);
    overflow-wrap: anywhere;
  }
  .content > :deep(:first-child) {
    margin-top: 0;
  }
  .content :deep(p) {
    margin: 0 0 12px;
  }
  .content :deep(:is(h1, h2, h3, h4)) {
    margin: 20px 0 10px;
    font-weight: 600;
    line-height: 1.4;
    color: var(--el-text-color-primary);
  }
  .content :deep(h1) {
    font-size: 22px;
  }
  .content :deep(h2) {
    font-size: 19px;
  }
  .content :deep(h3) {
    font-size: 17px;
  }
  .content :deep(h4) {
    font-size: 15px;
  }
  .content :deep(strong) {
    font-weight: 600;
    color: var(--el-text-color-primary);
  }
  .content :deep(a) {
    color: var(--el-color-primary);
    text-decoration: underline;
    text-underline-offset: 2px;
  }
  .content :deep(ul),
  .content :deep(ol) {
    margin: 0 0 12px;
    padding-left: 1.6em;
  }
  .content :deep(ul) {
    list-style: disc;
  }
  .content :deep(ol) {
    list-style: decimal;
  }
  .content :deep(li) {
    margin: 4px 0;
  }
  .content :deep(li::marker) {
    color: var(--el-color-primary);
  }
  .content :deep(blockquote) {
    margin: 0 0 12px;
    padding: 8px 16px;
    border-left: 4px solid var(--el-color-primary);
    border-radius: 0 6px 6px 0;
    background: var(--el-fill-color-light);
    color: var(--el-text-color-secondary);
  }
  .content :deep(blockquote > :last-child) {
    margin-bottom: 0;
  }
  .content :deep(code) {
    padding: 1px 6px;
    border-radius: 4px;
    background: var(--el-fill-color);
    font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
    font-size: 0.9em;
  }
  .content :deep(pre) {
    margin: 0 0 12px;
    padding: 12px;
    border-radius: 6px;
    background: var(--el-fill-color-light);
    overflow-x: auto;
  }
  .content :deep(pre code) {
    padding: 0;
    background: none;
  }
  .content :deep(img) {
    max-width: 100%;
    height: auto;
    margin: 4px 0;
    border-radius: 6px;
  }
  form > p {
    margin: 12px 0;
  }
</style>
