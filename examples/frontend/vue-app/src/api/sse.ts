// SSE client: fetch()-based reader for POST-initiated SSE streams (chat),
// and EventSource for GET-initiated SSE streams (plan events).

import type { SSEEvent } from '@/types'

/**
 * Stream SSE events from a POST endpoint (chat).
 * Uses fetch() with ReadableStream because EventSource only supports GET.
 */
export async function streamChat(
  url: string,
  body: { message: string; modelId?: string; provider?: string },
  onEvent: (event: SSEEvent) => void,
  signal?: AbortSignal,
): Promise<void> {
  const response = await fetch(url, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
    signal,
  })

  if (!response.ok) {
    throw new Error(`HTTP ${response.status}`)
  }

  const reader = response.body!.getReader()
  const decoder = new TextDecoder()
  let buffer = ''

  while (true) {
    const { done, value } = await reader.read()
    if (done) break

    buffer += decoder.decode(value, { stream: true })
    const lines = buffer.split('\n')
    buffer = lines.pop() || ''

    for (const line of lines) {
      if (line.startsWith('data: ')) {
        try {
          const event: SSEEvent = JSON.parse(line.slice(6))
          onEvent(event)
        } catch {
          // skip malformed frames
        }
      }
    }
  }
}

/**
 * Connect to a GET SSE endpoint (plan events).
 * Returns a cleanup function that closes the connection.
 */
export function connectSSE(
  url: string,
  onEvent: (event: SSEEvent) => void,
  onError?: (err: Event) => void,
): () => void {
  const es = new EventSource(url)

  es.onmessage = (msg) => {
    try {
      const event: SSEEvent = JSON.parse(msg.data)
      onEvent(event)
    } catch {
      // skip malformed
    }
  }

  es.onerror = (err) => {
    if (onError) onError(err)
  }

  return () => es.close()
}
