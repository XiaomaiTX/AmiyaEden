import { EditorContent, useEditor } from '@tiptap/react'
import StarterKit from '@tiptap/starter-kit'
import Link from '@tiptap/extension-link'
import Image from '@tiptap/extension-image'
import Underline from '@tiptap/extension-underline'
import { Bold, Code, ImagePlus, Italic, Link as LinkIcon, List, ListOrdered, Quote, Redo2, Strikethrough, Underline as UnderlineIcon, Undo2 } from 'lucide-react'
import { useEffect } from 'react'
import { Button } from '@/components/ui/button'

export function RichTextEditor({ value, onChange, ariaLabel }: { value: string; onChange: (value: string) => void; ariaLabel: string }) {
  const editor = useEditor({ extensions: [StarterKit.configure({ heading: { levels: [1, 2, 3, 4] } }), Underline, Link.configure({ openOnClick: false }), Image], content: value, onUpdate: ({ editor: instance }) => onChange(instance.getHTML()) })
  useEffect(() => { if (editor && value !== editor.getHTML()) editor.commands.setContent(value, { emitUpdate: false }) }, [editor, value])
  if (!editor) return null
  const setLink = () => { const url = window.prompt('URL'); if (url) editor.chain().focus().extendMarkRange('link').setLink({ href: url }).run() }
  const setImage = () => { const src = window.prompt('Image URL'); if (src) editor.chain().focus().setImage({ src }).run() }
  const actions = [
    [Bold, () => editor.chain().focus().toggleBold().run(), 'bold'], [Italic, () => editor.chain().focus().toggleItalic().run(), 'italic'], [UnderlineIcon, () => editor.chain().focus().toggleUnderline().run(), 'underline'], [Strikethrough, () => editor.chain().focus().toggleStrike().run(), 'strike'],
    [List, () => editor.chain().focus().toggleBulletList().run(), 'bullet list'], [ListOrdered, () => editor.chain().focus().toggleOrderedList().run(), 'ordered list'], [Quote, () => editor.chain().focus().toggleBlockquote().run(), 'quote'], [Code, () => editor.chain().focus().toggleCodeBlock().run(), 'code block'], [LinkIcon, setLink, 'link'], [ImagePlus, setImage, 'image'], [Undo2, () => editor.chain().focus().undo().run(), 'undo'], [Redo2, () => editor.chain().focus().redo().run(), 'redo'],
  ] as const
  return <div className="rounded-lg border bg-background"><div className="flex flex-wrap gap-1 border-b p-2">{actions.map(([Icon, action, label]) => <Button key={label} type="button" size="icon-xs" variant="ghost" onClick={action} aria-label={label}><Icon /></Button>)}</div><EditorContent editor={editor} aria-label={ariaLabel} className="min-h-32 p-3 text-sm [&_.ProseMirror]:min-h-24 [&_.ProseMirror]:outline-none [&_img]:max-w-full" /></div>
}
