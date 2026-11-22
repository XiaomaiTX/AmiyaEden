<template>
  <div class="announcements">
    <header
      ><h1>{{ t('announcement.title') }}</h1
      ><ElButton v-if="manage" :disabled="busy" @click="startEdit(true)">{{
        t('announcement.publish')
      }}</ElButton></header
    >
    <p v-if="error" role="alert">{{ t('announcement.error') }}</p>
    <div class="board">
      <aside>
        <p v-if="!rows.length">{{ t('announcement.empty') }}</p>
        <button
          v-for="row in rows"
          :key="row.id"
          :disabled="busy"
          class="entry"
          :class="{ unread: row.unread && !row.hidden, selected: selected?.id === row.id }"
          @click="open(row)"
        >
          <div class="entry-head">
            <span class="entry-title">
              <span v-if="row.unread && !row.hidden" class="unread-dot" aria-hidden="true" />
              <span class="title-text">{{ row.title }}</span>
            </span>
            <span v-if="row.hidden" class="badge badge-muted">{{ t('announcement.hidden') }}</span>
            <span v-else-if="row.unread" class="badge badge-unread">{{
              t('announcement.unread')
            }}</span>
          </div>
          <time>{{ new Date(row.created_at).toLocaleString() }}</time>
        </button>
        <ElPagination
          v-model:current-page="page"
          :page-size="20"
          :total="total"
          layout="prev, pager, next"
          @current-change="reload()"
        />
      </aside>
      <section :aria-busy="busy">
        <form v-if="editing" :key="editorKey" @submit.prevent="save">
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
          <header
            ><h2>{{ selected.title }}</h2
            ><ElButton v-if="manage" :disabled="busy" @click="startEdit(false)">{{
              t('announcement.edit')
            }}</ElButton></header
          >
          <time>{{ new Date(selected.created_at).toLocaleString() }}</time>
          <div class="content" v-html="selected.body_html" />
        </template>
        <p v-else>{{ t('announcement.select') }}</p>
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
    saveAnnouncement,
    readAnnouncement
  } from '@/api/announcements'
  import type { Announcement } from '@/types/announcement'
  import { useUserStore } from '@/store/modules/user'
  import { useBadgeStore } from '@/store/modules/badge'
  defineOptions({ name: 'Announcements' })
  const MAX_IMAGES = 6
  const { t } = useI18n()
  const user = useUserStore()
  const manage = computed(() =>
    (user.info.roles ?? []).some((r) => ['admin', 'super_admin', 'senior_fc'].includes(r))
  )
  const rows = ref<Announcement[]>([]),
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
    error.value = false
    try {
      const data = await listAnnouncements(page.value)
      rows.value = data.list
      total.value = data.total
      // On entry show the newest post, unless the user already opened one or started publishing.
      if (openNewest && token === request && data.list[0]) void open(data.list[0])
      await useBadgeStore().fetchBadgeCounts()
    } catch {
      error.value = true
    }
  }
  async function open(row: Announcement) {
    const token = ++request
    busy.value = true
    error.value = false
    editing.value = false
    let data: Announcement
    try {
      data = await getAnnouncement(row.id)
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
      await readAnnouncement(data.id, data.version)
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
      selected.value = await saveAnnouncement(selected.value?.id ?? null, {
        title: title.value,
        body_html: body.value,
        hidden: hidden.value,
        remind: remind.value,
        revision: selected.value?.revision ?? 0
      })
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
<style scoped>
  .announcements {
    padding: 24px;
  }
  header,
  footer {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 16px;
    margin-bottom: 16px;
  }
  h1 {
    font-size: 24px;
  }
  h2 {
    font-size: 22px;
    font-weight: 600;
    line-height: 1.4;
  }
  .board {
    display: grid;
    grid-template-columns: 320px minmax(0, 1fr);
    gap: 20px;
  }
  .entry {
    display: block;
    width: 100%;
    text-align: left;
    padding: 16px;
    border: 1px solid var(--el-border-color);
    border-radius: 8px;
    margin-bottom: 8px;
    cursor: pointer;
    background: var(--el-bg-color);
    transition:
      border-color 0.15s ease,
      background-color 0.15s ease;
  }
  .entry:hover {
    background: var(--el-fill-color-light);
  }
  .entry-head {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    gap: 8px;
  }
  .entry-title {
    display: flex;
    align-items: flex-start;
    gap: 8px;
    min-width: 0;
  }
  .title-text {
    font-weight: 500;
    line-height: 1.4;
    display: -webkit-box;
    -webkit-line-clamp: 2;
    line-clamp: 2;
    -webkit-box-orient: vertical;
    overflow: hidden;
    word-break: break-word;
  }
  .unread .title-text {
    font-weight: 700;
    color: var(--el-text-color-primary);
  }
  .unread-dot {
    width: 8px;
    height: 8px;
    margin-top: 6px;
    border-radius: 9999px;
    background: var(--el-color-primary);
    flex-shrink: 0;
  }
  .badge {
    flex-shrink: 0;
    padding: 2px 8px;
    border-radius: 9999px;
    font-size: 12px;
    line-height: 1.4;
    font-weight: 600;
  }
  .badge-unread {
    background: var(--el-color-primary);
    color: #fff;
  }
  .badge-muted {
    background: var(--el-fill-color);
    color: var(--el-text-color-secondary);
    font-weight: 500;
  }
  .entry time,
  time {
    display: block;
    font-size: 12px;
    opacity: 0.7;
    margin-top: 8px;
  }
  .unread {
    border-color: var(--el-color-primary);
    border-left: 4px solid var(--el-color-primary);
    background: var(--el-color-primary-light-9);
  }
  .unread:hover {
    background: var(--el-color-primary-light-8);
  }
  .selected {
    outline: 2px solid var(--el-color-primary);
  }
  section {
    min-width: 0;
    border: 1px solid var(--el-border-color);
    border-radius: 8px;
    padding: 20px;
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
  .content :deep(hr) {
    margin: 20px 0;
    border: 0;
    border-top: 1px solid var(--el-border-color);
  }
  .content :deep(table) {
    width: 100%;
    margin: 0 0 12px;
    border-collapse: collapse;
  }
  .content :deep(:is(th, td)) {
    padding: 6px 10px;
    border: 1px solid var(--el-border-color);
    text-align: left;
  }
  .content :deep(th) {
    background: var(--el-fill-color-light);
    font-weight: 600;
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
  @media (max-width: 900px) {
    .board {
      grid-template-columns: 1fr;
    }
  }
</style>
