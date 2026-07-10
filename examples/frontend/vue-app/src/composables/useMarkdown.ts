import { marked } from 'marked'
import hljs from 'highlight.js'

marked.setOptions({
  // @ts-ignore highlight is deprecated in marked v15 but still works
  highlight: (code: string, lang: string) => {
    if (lang && hljs.getLanguage(lang)) {
      return hljs.highlight(code, { language: lang }).value
    }
    return hljs.highlightAuto(code).value
  },
  breaks: true,
  gfm: true,
})

export function renderMarkdown(text: string): string {
  if (!text) return ''
  return marked.parse(text) as string
}
