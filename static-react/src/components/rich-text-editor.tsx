import { EditorContent, useEditor } from '@tiptap/react'
import StarterKit from '@tiptap/starter-kit'
import Link from '@tiptap/extension-link'
import Image from '@tiptap/extension-image'
import Underline from '@tiptap/extension-underline'
import {
  Bold,
  Code,
  ImagePlus,
  Italic,
  Link as LinkIcon,
  List,
  ListOrdered,
  Quote,
  Redo2,
  Strikethrough,
  Underline as UnderlineIcon,
  Undo2,
} from 'lucide-react'
import { useEffect, useState } from 'react'
import { useI18n } from '@/i18n'
import { Button } from '@/components/ui/button'

const MAX_IMAGE_BYTES = 8 * 1024 * 1024

export function RichTextEditor({
  value,
  onChange,
  ariaLabel,
  imageUploadLabel,
  maxImages,
}: {
  value: string
  onChange: (value: string) => void
  /** Enables local PNG/JPEG upload as data URLs instead of prompting for an image URL. */
  imageUploadLabel?: string
  /** Upper bound on images in the whole document, checked before each upload. */
  maxImages?: number
  ariaLabel: string
}) {
  const { t } = useI18n()
  const [notice, setNotice] = useState('')
  const editor = useEditor({
    extensions: [
      StarterKit.configure({ heading: { levels: [1, 2, 3, 4] }, underline: false, link: false }),
      Underline,
      Link.configure({ openOnClick: false }),
      Image.configure({ allowBase64: Boolean(imageUploadLabel) }),
    ],
    content: value,
    onUpdate: ({ editor: instance }) => onChange(instance.getHTML()),
  })
  useEffect(() => {
    if (editor && value !== editor.getHTML())
      editor.commands.setContent(value, { emitUpdate: false })
  }, [editor, value])
  if (!editor) return null
  const countImages = () => {
    let count = 0
    editor.state.doc.descendants((node) => {
      if (node.type.name === 'image') count++
    })
    return count
  }
  const setLink = () => {
    const url = window.prompt(t('richTextEditor.linkPrompt'))
    if (url) editor.chain().focus().extendMarkRange('link').setLink({ href: url }).run()
  }
  const setImage = () => {
    setNotice('')
    if (imageUploadLabel) {
      if (maxImages !== undefined && countImages() >= maxImages) {
        setNotice(t('richTextEditor.tooManyImages', { count: maxImages }))
        return
      }
      const input = document.createElement('input')
      input.type = 'file'
      input.accept = 'image/png,image/jpeg'
      input.onchange = () => {
        const file = input.files?.[0]
        if (!file) return
        if (
          (file.type !== 'image/png' && file.type !== 'image/jpeg') ||
          file.size > MAX_IMAGE_BYTES
        ) {
          setNotice(t('richTextEditor.imageRejected'))
          return
        }
        const reader = new FileReader()
        reader.onload = () => {
          if (editor.isDestroyed) return
          if (maxImages !== undefined && countImages() >= maxImages) {
            setNotice(t('richTextEditor.tooManyImages', { count: maxImages }))
            return
          }
          editor
            .chain()
            .focus()
            .setImage({ src: String(reader.result) })
            .run()
        }
        reader.onerror = () => setNotice(t('richTextEditor.imageRejected'))
        reader.readAsDataURL(file)
      }
      input.click()
      return
    }
    const src = window.prompt(t('richTextEditor.imagePrompt'))
    if (src) editor.chain().focus().setImage({ src }).run()
  }
  const actions = [
    [Bold, () => editor.chain().focus().toggleBold().run(), 'bold'],
    [Italic, () => editor.chain().focus().toggleItalic().run(), 'italic'],
    [UnderlineIcon, () => editor.chain().focus().toggleUnderline().run(), 'underline'],
    [Strikethrough, () => editor.chain().focus().toggleStrike().run(), 'strikethrough'],
    [List, () => editor.chain().focus().toggleBulletList().run(), 'bulletList'],
    [ListOrdered, () => editor.chain().focus().toggleOrderedList().run(), 'orderedList'],
    [Quote, () => editor.chain().focus().toggleBlockquote().run(), 'quote'],
    [Code, () => editor.chain().focus().toggleCodeBlock().run(), 'codeBlock'],
    [LinkIcon, setLink, 'link'],
    [ImagePlus, setImage, 'image'],
    [Undo2, () => editor.chain().focus().undo().run(), 'undo'],
    [Redo2, () => editor.chain().focus().redo().run(), 'redo'],
  ] as const
  return (
    <div className="rounded-lg border bg-background">
      <div className="flex flex-wrap gap-1 border-b p-2">
        {actions.map(([Icon, action, key]) => (
          <Button
            key={key}
            type="button"
            size="icon-xs"
            variant="ghost"
            onPress={action}
            aria-label={
              key === 'image' && imageUploadLabel ? imageUploadLabel : t(`richTextEditor.${key}`)
            }
          >
            <Icon />
          </Button>
        ))}
      </div>
      {notice && (
        <p role="alert" className="border-b px-3 py-2 text-xs text-destructive">
          {notice}
        </p>
      )}
      <EditorContent
        editor={editor}
        aria-label={ariaLabel}
        className="min-h-32 p-3 text-sm [&_.ProseMirror]:min-h-24 [&_.ProseMirror]:outline-none [&_img]:max-w-full"
      />
    </div>
  )
}
