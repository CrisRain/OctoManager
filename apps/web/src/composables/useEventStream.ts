import { ref, watch, onUnmounted, type Ref } from "vue";

export type EventStreamStatus = "idle" | "connecting" | "open" | "closed" | "error";

export interface UseEventStreamOptions {
  eventNames?: string[];
  closeOn?: string[];
  heartbeatEvents?: string[];
  includeHeartbeatInLines?: boolean;
}

const MAX_LOG_LINES = 200;

export function useEventStream(
  url: Ref<string | null>,
  options: UseEventStreamOptions = {}
) {
  const lines = ref<string[]>([]);
  const status = ref<EventStreamStatus>("idle");
  const lastHeartbeatAt = ref<number | null>(null);
  const heartbeatPayload = ref<unknown>(null);

  let controller: AbortController | null = null;

  function cleanup() {
    if (controller) {
      controller.abort();
      controller = null;
    }
  }

  async function connect(streamUrl: string) {
    cleanup();

    lines.value = [];
    status.value = "connecting";
    lastHeartbeatAt.value = null;
    heartbeatPayload.value = null;

    const ac = new AbortController();
    controller = ac;

    const closeOnSet = new Set(options.closeOn ?? []);
    const heartbeatSet = new Set(options.heartbeatEvents ?? ["heartbeat"]);

    try {
      const response = await fetch(streamUrl, { signal: ac.signal });

      if (!response.ok || !response.body) {
        status.value = "error";
        return;
      }

      status.value = "open";

      const reader = response.body.getReader();
      const decoder = new TextDecoder();
      let buffer = "";

      while (true) {
        const { done, value } = await reader.read();
        if (done) break;

        buffer += decoder.decode(value, { stream: true });

        // Split on newlines; keep the last incomplete chunk in the buffer.
        const newline = buffer.lastIndexOf("\n");
        if (newline === -1) continue;

        const complete = buffer.slice(0, newline);
        buffer = buffer.slice(newline + 1);

        for (const line of complete.split("\n")) {
          const trimmed = line.trim();
          if (!trimmed) continue;

          let event = "";
          let raw = trimmed;

          try {
            const parsed = JSON.parse(trimmed) as { event?: string; data?: unknown };
            event = parsed.event ?? "";
            raw = JSON.stringify(parsed.data ?? parsed);
          } catch {
            // Not valid JSON — treat the whole line as raw data.
          }

          const isHeartbeat = heartbeatSet.has(event);
          if (isHeartbeat) {
            lastHeartbeatAt.value = Date.now();
            try {
              heartbeatPayload.value = JSON.parse(raw);
            } catch {
              heartbeatPayload.value = raw;
            }
            if (!options.includeHeartbeatInLines) continue;
          }

          lines.value = [...lines.value, raw].slice(-MAX_LOG_LINES);

          if (closeOnSet.has(event)) {
            status.value = "closed";
            reader.cancel();
            return;
          }
        }
      }

      // Stream ended normally.
      if (status.value === "open") status.value = "closed";
    } catch (err) {
      if ((err as Error).name === "AbortError") return;
      status.value = "error";
    }
  }

  watch(
    url,
    (newUrl) => {
      cleanup();
      if (newUrl) {
        connect(newUrl);
      } else {
        lines.value = [];
        status.value = "idle";
      }
    },
    { immediate: true }
  );

  onUnmounted(cleanup);

  return { lines, status, lastHeartbeatAt, heartbeatPayload };
}
