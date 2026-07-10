<template>
  <div class="markdown-content" v-html="html" />
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { renderMarkdown } from '@/composables/useMarkdown'

const props = defineProps<{ content: string }>()
const html = computed(() => renderMarkdown(props.content))
</script>

<style>
.markdown-content {
  line-height: 1.6;
  word-break: break-word;
}
.markdown-content pre {
  background: rgba(0,0,0,0.2);
  padding: 12px 16px;
  border-radius: 6px;
  overflow-x: auto;
  margin: 8px 0;
}
.markdown-content code {
  font-family: 'Fira Code', 'Cascadia Code', monospace;
  font-size: 0.9em;
}
.markdown-content p { margin: 4px 0; }
.markdown-content ul, .markdown-content ol { padding-left: 20px; margin: 4px 0; }
.markdown-content h1, .markdown-content h2, .markdown-content h3 { margin: 12px 0 4px; }
.markdown-content blockquote {
  border-left: 3px solid var(--n-color-target);
  padding-left: 12px;
  margin: 8px 0;
  opacity: 0.8;
}
</style>
