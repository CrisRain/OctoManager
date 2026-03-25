<script setup lang="ts">
import { ref, computed, watch } from "vue";
import { IconSearch, IconCommand } from "@/lib/icons";

interface CommandItem {
  id: string;
  label: string;
  description?: string;
  icon?: any;
  shortcut?: string;
  keywords?: string[];
  action?: () => void;
}

const props = withDefaults(defineProps<{
  commands: CommandItem[];
  open?: boolean;
  query?: string;
}>(), {
  commands: () => [],
  open: false,
  query: "",
});

const emit = defineEmits<{
  (e: "update:open", value: boolean): void;
  (e: "update:query", value: string): void;
  (e: "execute", command: CommandItem): void;
}>();

const isOpen = computed({
  get: () => props.open ?? false,
  set: (value) => emit("update:open", value),
});

const searchQuery = computed({
  get: () => props.query ?? "",
  set: (value) => emit("update:query", value),
});

const filteredCommands = computed(() => {
  const keyword = searchQuery.value.trim().toLowerCase();
  if (!keyword) return props.commands;

  return props.commands.filter((command) => {
    const tokens = [
      command.label,
      command.description ?? "",
      ...(command.keywords ?? []),
    ]
      .join(" ")
      .toLowerCase();
    return tokens.includes(keyword);
  });
});

// ── 键盘导航 ──────────────────────────────────────────────
const activeIndex = ref(0);

// 打开面板时重置索引
watch(isOpen, (open) => {
  if (open) activeIndex.value = 0;
});

// 搜索变化时重置索引
watch(filteredCommands, () => {
  activeIndex.value = 0;
});

function onKeyDown(e: KeyboardEvent) {
  if (!filteredCommands.value.length) return;

  if (e.key === "ArrowDown") {
    e.preventDefault();
    activeIndex.value = (activeIndex.value + 1) % filteredCommands.value.length;
    scrollActiveIntoView();
  } else if (e.key === "ArrowUp") {
    e.preventDefault();
    activeIndex.value =
      (activeIndex.value - 1 + filteredCommands.value.length) % filteredCommands.value.length;
    scrollActiveIntoView();
  } else if (e.key === "Enter") {
    e.preventDefault();
    const cmd = filteredCommands.value[activeIndex.value];
    if (cmd) executeCommand(cmd);
  }
}

const listRef = ref<HTMLElement | null>(null);

function scrollActiveIntoView() {
  // Allow DOM to update first
  requestAnimationFrame(() => {
    const el = listRef.value?.children[activeIndex.value] as HTMLElement | undefined;
    el?.scrollIntoView({ block: "nearest" });
  });
}

// ── 搜索关键词高亮 ─────────────────────────────────────────
function highlight(text: string): string {
  const keyword = searchQuery.value.trim();
  if (!keyword) return text;
  const escaped = keyword.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
  return text.replace(
    new RegExp(`(${escaped})`, "gi"),
    '<mark class="bg-[var(--accent)]/15 text-[var(--accent)] rounded-sm not-italic font-semibold">$1</mark>',
  );
}

function executeCommand(command: CommandItem) {
  if (command.action) {
    command.action();
  }
  emit("execute", command);
  emit("update:open", false);
  emit("update:query", "");
}
</script>

<template>
  <div class="keyboard-shortcuts">
    <!-- Command palette -->
    <ui-modal v-model:visible="isOpen" :footer="false" :closable="false" class="search-modal">
      <div class="panel-surface flex flex-col overflow-hidden bg-slate-50" role="dialog" aria-label="命令面板" aria-modal="true">
        <div class="flex items-center gap-3.5 border-b border-slate-200 bg-slate-50/85 px-5 py-4">
          <icon-search class="h-4 w-4 flex-shrink-0 text-slate-500" aria-hidden="true" />
          <input
            v-model="searchQuery"
            type="text"
            role="combobox"
            aria-autocomplete="list"
            :aria-expanded="filteredCommands.length > 0"
            aria-controls="command-palette-list"
            :aria-activedescendant="filteredCommands[activeIndex]?.id ? `cmd-${filteredCommands[activeIndex].id}` : undefined"
            class="flex-1 bg-transparent text-base font-medium text-slate-900 placeholder:text-slate-500 outline-none"
            placeholder="搜索页面、功能..."
            autofocus
            @keydown="onKeyDown"
          />
          <kbd class="rounded-full border border-slate-200 bg-white/70 px-2.5 py-1 font-mono text-xs font-medium text-slate-500" aria-label="按 ESC 关闭">ESC</kbd>
        </div>

        <div
          id="command-palette-list"
          ref="listRef"
          role="listbox"
          aria-label="命令列表"
          class="overflow-y-auto px-3 py-3 max-h-[60vh] dark-scroll"
        >
          <button
            type="button"
            role="option"
            v-for="(command, index) in filteredCommands"
            :key="command.id"
            :id="`cmd-${command.id}`"
            :aria-selected="activeIndex === index"
            class="flex w-full items-center gap-3.5 rounded-xl border border-transparent bg-transparent px-4 py-3 text-left transition-all hover:border-slate-200 hover:bg-white/92 hover:shadow-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--accent)]/50"
            :class="{ 'border-slate-200 bg-white/92 shadow-sm': activeIndex === index }"
            @click="executeCommand(command)"
            @mouseenter="activeIndex = index"
          >
            <div class="flex h-8 w-8 flex-shrink-0 items-center justify-center rounded-lg bg-slate-50 border border-slate-200 text-slate-500 shadow-sm" aria-hidden="true">
              <component :is="command.icon || IconSearch" class="h-4 w-4" />
            </div>
            <div class="flex min-w-0 flex-1 flex-col gap-0.5">
              <!-- v-html is safe: text comes from internal command definitions, not API/user input -->
              <div class="text-sm font-semibold text-slate-900" v-html="highlight(command.label)" />
              <div v-if="command.description" class="text-xs leading-5 text-slate-500" v-html="highlight(command.description)" />
            </div>
            <div class="flex-shrink-0 rounded-full bg-slate-100 px-2 py-1 text-xs text-slate-500" aria-hidden="true">
              <span v-if="command.shortcut">{{ command.shortcut }}</span>
              <span v-else-if="index === 0">↵</span>
              <span v-else>{{ index + 1 }}</span>
            </div>
          </button>

          <div v-if="!filteredCommands.length" role="status" class="py-10 text-center text-sm text-slate-500">
            未找到相关结果
          </div>
        </div>

        <!-- 键盘提示 -->
        <div class="border-t border-slate-200 px-5 py-2.5 flex items-center gap-4 text-xs text-slate-400" aria-hidden="true">
          <span><kbd class="font-mono">↑↓</kbd> 导航</span>
          <span><kbd class="font-mono">↵</kbd> 执行</span>
          <span><kbd class="font-mono">ESC</kbd> 关闭</span>
        </div>
      </div>
    </ui-modal>

    <!-- Floating trigger: 44×44 (h-11 w-11 = 44px) -->
    <button type="button"
      aria-label="打开命令面板 (⌘K)"
      class="fixed bottom-[calc(1.25rem_+_env(safe-area-inset-bottom))] right-5 z-30 inline-flex h-11 w-11 items-center justify-center rounded-full border border-slate-200 bg-white/92 text-slate-500 shadow-md backdrop-blur transition-all hover:border-[var(--accent)]/30 hover:text-[var(--accent)] hover:-translate-y-0.5 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--accent)]/50 focus-visible:ring-offset-2"
      @click="emit('update:open', true)"
    >
      <icon-command aria-hidden="true" />
    </button>
  </div>
</template>
