<script setup lang="ts">
/**
 * LogTerminal — fixed-height terminal with internal scrolling.
 */
import { ref, computed, watch, nextTick, onMounted, onBeforeUnmount } from "vue";
import VirtualList from "./VirtualList.vue";

interface Props {
  /** Raw log lines to display */
  logs: string[];
  title?: string;
  /** Whether the log stream is currently live (controls the status LED) */
  isLive?: boolean;
  /** Terminal height. Accepts CSS length values like "34rem" or numbers in px. */
  height?: string | number;
  showHeader?: boolean;
  emptyLabel?: string;
}

const props = withDefaults(defineProps<Props>(), {
  title: "运行日志",
  isLive: false,
  height: "34em",
  showHeader: true,
  emptyLabel: "暂无日志",
});

// ── State ──────────────────────────────────────────────────────────────────
const follow = ref(true);
const scrollerRef = ref<HTMLDivElement | null>(null);

let resizeObserver: ResizeObserver | null = null;

onMounted(() => {
  if (!scrollerRef.value) return;
  // Listen for visibility/resize changes (e.g. when tab becomes active)
  resizeObserver = new ResizeObserver((entries) => {
    for (const entry of entries) {
      if (entry.contentRect.height > 0 && follow.value) {
        scrollToBottom();
      }
    }
  });
  resizeObserver.observe(scrollerRef.value);
});

onBeforeUnmount(() => {
  if (resizeObserver) {
    resizeObserver.disconnect();
  }
});

// ── Log parsing ────────────────────────────────────────────────────────────
interface ParsedLine {
  /** Stable unique ID — logs are append-only so index is safe */
  id: number;
  source: string;
  level: string;
  message: string;
  payload?: any;
}

const LEVEL_RE = /^\[(.*?)\]\[(.*?)\]\s*(.*)$/;

function parseLine(raw: string, index: number): ParsedLine {
  try {
    const parsed = JSON.parse(raw);
    if (parsed && typeof parsed === 'object') {
      return {
        id: index,
        source: parsed.event_type || '',
        level: parsed.payload?.level || 'info',
        message: parsed.message || (parsed.payload ? '' : JSON.stringify(parsed)),
        payload: parsed.payload || (parsed.message ? null : parsed),
      };
    }
  } catch (e) {
    // Fallback to text parsing if not valid JSON
  }

  const m = LEVEL_RE.exec(raw);
  if (!m) return { id: index, source: "", level: "", message: raw };
  return { id: index, source: m[1] ?? "", level: m[2] ?? "", message: m[3] ?? "" };
}

const parsedItems = computed<ParsedLine[]>(() =>
  props.logs.map((line, i) => parseLine(line, i))
);

// ── Level CSS classes (colors defined in tailwind.css .log-level-* rules) ──
const LEVEL_CLASS: Record<string, string> = {
  error: "log-level-error",
  warn:  "log-level-warn",
  info:  "log-level-info",
  debug: "log-level-debug",
};

function levelClass(level: string): string {
  return LEVEL_CLASS[level.toLowerCase()] ?? "log-level-default";
}

const surfaceStyle = computed(() => ({
  "--terminal-height": typeof props.height === "number" ? `${props.height}px` : props.height,
}));

// ── Auto-scroll to bottom ──────────────────────────────────────────────────
async function scrollToBottom() {
  if (!props.logs.length) return;
  // wait for DOM update
  await nextTick();
  const container = scrollerRef.value;
  if (!container) return;
  // Set immediately
  container.scrollTop = container.scrollHeight;
  // Also set after paint
  requestAnimationFrame(() => {
    container.scrollTop = container.scrollHeight;
  });
}

// logs 仅追加，监听长度变化即可；不需要 deep watch 整个数组
watch(() => props.logs.length, () => {
  if (follow.value) void scrollToBottom();
});

// Also handle user manual scroll to disable auto-follow
function handleScroll(e: Event) {
  const container = e.target as HTMLDivElement;
  // If user scrolled up, disable follow
  const isAtBottom = container.scrollHeight - container.scrollTop <= container.clientHeight + 40; // 40px tolerance
  if (!isAtBottom && follow.value) {
    follow.value = false;
  } else if (isAtBottom && !follow.value) {
    follow.value = true;
  }
}

// Re-enable follow → jump to bottom immediately
watch(follow, (on) => {
  if (on) void scrollToBottom();
});
</script>

<template>
  <div
    class="relative flex flex-col overflow-hidden rounded-xl bg-[#0d1117] font-mono text-xs leading-relaxed shadow-lg ring-1 ring-white/[4%]"
    :style="[surfaceStyle, { height: 'var(--terminal-height)' }]"
  >
    <!-- ── Header bar ────────────────────────────────────────────────────── -->
    <div v-if="showHeader" class="flex flex-shrink-0 items-center gap-3 border-b border-white/[5%] bg-slate-900/60 px-4 py-2.5 backdrop-blur-sm">
      <!-- macOS traffic lights -->
      <div class="flex items-center gap-2" aria-hidden="true">
        <span class="h-3 w-3 rounded-full bg-[#ff5f57]" />
        <span class="h-3 w-3 rounded-full bg-[#febc2e]" />
        <span class="h-3 w-3 rounded-full bg-[#28c840]" />
      </div>

      <!-- Title -->
      <span class="flex-1 truncate text-xs font-medium text-slate-500">{{ title }}</span>

      <!-- Live badge -->
      <span
        v-if="isLive"
        class="inline-flex items-center gap-1 rounded-full bg-emerald-500/10 px-2 py-0.5 text-xs font-bold uppercase tracking-widest text-emerald-400"
      >
        <span class="h-1.5 w-1.5 animate-[pulse-dot-soft_1.8s_ease-in-out_infinite] rounded-full bg-emerald-400" />
        Live
      </span>

      <!-- Line count -->
      <span class="text-xs tabular-nums text-slate-600">{{ logs.length }} lines</span>

      <!-- Follow toggle -->
      <button
        type="button"
        class="rounded border px-2 py-0.5 text-xs font-medium transition-colors focus-visible:outline-none"
        :class="follow
          ? 'border-emerald-700/60 bg-emerald-500/10 text-emerald-400 hover:bg-emerald-500/15'
          : 'border-slate-700 bg-transparent text-slate-500 hover:border-slate-500 hover:text-slate-300'"
        @click="follow = !follow"
      >
        {{ follow ? "↓ Follow" : "Paused" }}
      </button>
    </div>

    <!-- ── Log body ──────────────────────────────────────────────────────── -->
    <div class="relative flex min-h-0 flex-1 flex-col">
      <!-- Empty state -->
      <div v-if="!logs.length" class="flex flex-1 items-center justify-center gap-2 py-12">
        <span class="text-[10px] text-slate-700">▌</span>
        <span class="text-xs text-slate-600">{{ emptyLabel }}</span>
      </div>

      <!-- Scrollable lines -->
      <div v-else ref="scrollerRef" class="flex-1 overflow-x-auto overflow-y-auto overscroll-contain py-2 dark-scroll" @scroll="handleScroll">
        <div class="min-w-max">
          <VirtualList
            :data="parsedItems"
            :item-height="32"
            key-field="id"
            v-slot="{ item }"
          >
            <div
              class="group flex items-center gap-0 hover:bg-white/[3%] w-full h-8 leading-8"
            >
              <!-- Line number gutter -->
              <span class="w-10 flex-shrink-0 select-none pr-3 text-right text-xs text-slate-700 group-hover:text-slate-500">
                {{ item.id + 1 }}
              </span>

              <!-- Level badge -->
              <span
                class="mr-2.5 flex-shrink-0 rounded px-1.5 py-0.5 text-[10px] font-bold uppercase tracking-wide leading-none"
                :class="levelClass(item.level)"
              >{{ item.level || "log" }}</span>

              <!-- Source -->
              <span
                v-if="item.source"
                class="mr-3 flex-shrink-0 text-xs text-slate-500 whitespace-nowrap"
              >{{ item.source }}</span>

              <!-- Message -->
              <div class="flex-1 pr-4 min-w-0 flex items-center gap-2">
                <span v-if="item.message" class="whitespace-nowrap text-slate-300">{{ item.message }}</span>
                <span v-if="item.payload && Object.keys(item.payload).length > 0" class="text-[11px] text-slate-300 bg-white/[0.04] px-2 py-0.5 rounded-md border border-white/[0.08] whitespace-nowrap leading-none h-[22px] flex items-center">{{ JSON.stringify(item.payload) }}</span>
              </div>
            </div>
          </VirtualList>
        </div>
      </div>
    </div>

    <!-- ── Status bar ────────────────────────────────────────────────────── -->
    <div class="flex flex-shrink-0 items-center justify-between border-t border-white/[4%] bg-slate-900/40 px-4 py-1 text-xs">
      <span class="text-slate-700">{{ title }}</span>
      <span
        class="flex items-center gap-2 font-medium"
        :class="isLive ? 'text-emerald-600' : 'text-slate-600'"
      >
        <span
          class="h-1.5 w-1.5 rounded-full"
          :class="isLive ? 'bg-emerald-500 animate-[pulse-dot-soft_1.8s_ease-in-out_infinite]' : 'bg-slate-600'"
        />
        {{ isLive ? "streaming" : "idle" }}
      </span>
    </div>

    <!-- CRT scanline overlay -->
    <div
      class="pointer-events-none absolute inset-0 bg-[repeating-linear-gradient(to_bottom,_transparent_0px,_transparent_3px,_rgba(0,_0,_0,_0.025)_3px,_rgba(0,_0,_0,_0.025)_4px)]"
      aria-hidden="true"
    />
  </div>
</template>
