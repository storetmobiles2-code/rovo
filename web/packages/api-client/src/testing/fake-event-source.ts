import type { EventSourceLike, StreamEventLike } from '../sse';

/**
 * Deterministic EventSource double for tests. Inspect `FakeEventSource.instances`, then drive
 * it with `open()`, `emit(name, data)`, `fail()`.
 */
export class FakeEventSource implements EventSourceLike {
  static instances: FakeEventSource[] = [];
  static reset() {
    FakeEventSource.instances = [];
  }
  static get latest(): FakeEventSource | undefined {
    return FakeEventSource.instances.at(-1);
  }
  static factory = (url: string): FakeEventSource => new FakeEventSource(url);

  readyState = 0;
  closed = false;
  onopen: ((ev: unknown) => void) | null = null;
  onerror: ((ev: unknown) => void) | null = null;
  private listeners = new Map<string, Array<(ev: StreamEventLike) => void>>();

  constructor(readonly url: string) {
    FakeEventSource.instances.push(this);
  }

  addEventListener(type: string, listener: (ev: StreamEventLike) => void): void {
    const list = this.listeners.get(type) ?? [];
    list.push(listener);
    this.listeners.set(type, list);
  }

  close(): void {
    this.readyState = 2;
    this.closed = true;
  }

  open(): void {
    this.readyState = 1;
    this.onopen?.({});
  }

  emit(type: string, data: unknown): void {
    const payload = typeof data === 'string' ? data : JSON.stringify(data);
    for (const l of this.listeners.get(type) ?? []) l({ type, data: payload });
  }

  fail(): void {
    this.readyState = 2;
    this.onerror?.({});
  }
}
