import type { WsEvent, WsEventType } from '@/types/daemon'

type EventHandler = (event: WsEvent) => void

export class WsClient {
  private ws: WebSocket | null = null
  private url = ''
  private handlers = new Map<string, Set<EventHandler>>()
  private reconnectAttempt = 0
  private maxReconnectDelay = 30_000 // 30s
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null
  private intentionalClose = false

  // Lifecycle callbacks
  onStatusChange: ((status: 'connected' | 'disconnected' | 'reconnecting') => void) | null = null

  connect(baseUrl: string, token: string): void {
    this.intentionalClose = false
    // Convert http:// to ws://
    const wsUrl = baseUrl.replace(/^http/, 'ws')
    this.url = `${wsUrl}/api/v1/ws?token=${encodeURIComponent(token)}`
    this._connect()
  }

  private _connect(): void {
    if (this.ws) {
      this.ws.close()
      this.ws = null
    }

    this.onStatusChange?.(this.reconnectAttempt > 0 ? 'reconnecting' : 'connected')

    try {
      this.ws = new WebSocket(this.url)
    } catch {
      this._scheduleReconnect()
      return
    }

    this.ws.onopen = () => {
      this.reconnectAttempt = 0
      this.onStatusChange?.('connected')
    }

    this.ws.onmessage = (msg) => {
      try {
        const event: WsEvent = JSON.parse(msg.data)
        this._dispatch(event)
      } catch {
        // Ignore malformed messages
      }
    }

    this.ws.onclose = () => {
      if (!this.intentionalClose) {
        this._scheduleReconnect()
      }
    }

    this.ws.onerror = () => {
      // onclose will fire after this
    }
  }

  private _scheduleReconnect(): void {
    this.onStatusChange?.('reconnecting')
    // Exponential backoff: 1s, 2s, 4s, 8s, 16s, 30s max
    const delay = Math.min(1000 * Math.pow(2, this.reconnectAttempt), this.maxReconnectDelay)
    this.reconnectAttempt++
    this.reconnectTimer = setTimeout(() => this._connect(), delay)
  }

  private _dispatch(event: WsEvent): void {
    const handlers = this.handlers.get(event.type)
    if (handlers) {
      handlers.forEach((h) => h(event))
    }
    // Also dispatch to wildcard handlers
    const wildcardHandlers = this.handlers.get('*')
    if (wildcardHandlers) {
      wildcardHandlers.forEach((h) => h(event))
    }
  }

  on(type: WsEventType | '*', handler: EventHandler): () => void {
    if (!this.handlers.has(type)) {
      this.handlers.set(type, new Set())
    }
    this.handlers.get(type)!.add(handler)
    return () => this.handlers.get(type)?.delete(handler)
  }

  disconnect(): void {
    this.intentionalClose = true
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer)
      this.reconnectTimer = null
    }
    if (this.ws) {
      this.ws.close()
      this.ws = null
    }
    this.onStatusChange?.('disconnected')
  }
}
