<script setup lang="ts">
import { useRouter } from "vue-router";
import { IconApps } from "@/lib/icons";

interface QuickAction {
  id: string;
  label: string;
  description: string;
  icon?: any;
  color?: string;
  path?: string;
  shortcut?: string;
  action?: () => void;
}

const props = withDefaults(defineProps<{
  actions: QuickAction[];
  showSearchHint?: boolean;
}>(), {
  actions: () => [],
  showSearchHint: true,
});

const emit = defineEmits<{
  (e: "open-search"): void;
}>();

const router = useRouter();

function handleActionClick(action: QuickAction) {
  if (action.action) {
    action.action();
    return;
  }
  if (action.path) {
    router.push(action.path);
  }
}

</script>

<template>
  <div class="flex flex-col gap-4">
    <!-- Quick actions -->
    <div class="rounded-xl border border-slate-200 bg-white p-6 shadow-sm">
      <div class="mb-5 flex items-center justify-between gap-3">
        <h3 class="text-[15px] font-semibold text-slate-900">快捷操作</h3>
        <button type="button"
          v-if="showSearchHint"
          class="flex items-center gap-2 rounded-lg border border-slate-200 bg-slate-50 px-2.5 py-1.5 text-xs text-slate-600 transition-all hover:bg-slate-100 hover:text-slate-900 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-slate-400/20 [&_kbd]:rounded [&_kbd]:border [&_kbd]:border-slate-200 [&_kbd]:bg-white [&_kbd]:px-1.5 [&_kbd]:py-0.5 [&_kbd]:font-mono [&_kbd]:text-xs [&_kbd]:text-slate-600"
          @click="emit('open-search')"
        >
          <kbd>⌘K</kbd>
          <span>打开搜索</span>
        </button>
      </div>

      <div class="grid grid-cols-2 gap-3 max-md:grid-cols-1">
        <button
          type="button"
          v-for="action in actions"
          :key="action.id"
          class="flex flex-col items-start gap-2 rounded-xl border border-slate-200 bg-white p-4 text-left shadow-sm transition-all hover:border-slate-300 hover:shadow-md hover:-translate-y-0.5 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-slate-400/20"
          @click="handleActionClick(action)"
        >
          <div class="mb-1 flex h-10 w-10 items-center justify-center rounded-lg border shadow-sm"
               :class="{
                 'border-blue-200 bg-blue-50 text-blue-600': action.color === 'blue',
                 'border-orange-200 bg-orange-50 text-orange-600': action.color === 'orange',
                 'border-[var(--accent)]/20 bg-[var(--accent)]/8 text-[var(--accent)]': action.color === 'teal',
                 'border-cyan-200 bg-cyan-50 text-cyan-600': action.color === 'cyan',
                 'border-slate-200 bg-slate-50 text-slate-600': !action.color,
               }"
          >
            <component :is="action.icon || IconApps" />
          </div>
          <div class="text-sm font-semibold text-slate-900">{{ action.label }}</div>
          <div class="text-xs leading-relaxed text-slate-500">{{ action.description }}</div>
          <div class="text-xs text-slate-400" v-if="action.shortcut">
            {{ action.shortcut }}
          </div>
        </button>
      </div>
    </div>

  </div>
</template>
