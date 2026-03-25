<script setup lang="ts">
import { ref, computed, watch } from "vue";
import {
  IconClose,
  IconEdit,
  IconDelete,
  IconRefresh,
} from "@/lib/icons";

interface DrawerTab {
  key: string;
  label: string;
  icon?: any;
}

interface Props {
  open: boolean;
  title?: string;
  loading?: boolean;
  showActions?: boolean;
  tabs?: DrawerTab[];
  tab?: string;
}

const props = withDefaults(defineProps<Props>(), {
  open: false,
  loading: false,
  showActions: true,
});

const emit = defineEmits<{
  (e: "update:open", value: boolean): void;
  (e: "update:tab", value: string): void;
  (e: "close"): void;
  (e: "edit"): void;
  (e: "delete"): void;
  (e: "refresh"): void;
}>();

const innerOpen = computed({
  get: () => props.open,
  set: (val) => emit("update:open", val),
});

const defaultTabs: DrawerTab[] = [
  { key: "detail", label: "详细信息" },
  { key: "activity", label: "活动记录" },
  { key: "settings", label: "设置" },
];

const resolvedTabs = computed(() =>
  props.tabs && props.tabs.length ? props.tabs : defaultTabs
);

const currentTab = ref(props.tab ?? resolvedTabs.value[0]?.key ?? "detail");

watch(
  () => props.tab,
  (val) => {
    if (val !== undefined && val !== currentTab.value) {
      currentTab.value = val;
    }
  }
);

watch(resolvedTabs, (tabs) => {
  if (!tabs.length) return;
  if (!tabs.find((tab) => tab.key === currentTab.value)) {
    currentTab.value = tabs[0].key;
    emit("update:tab", currentTab.value);
  }
});

function setTab(key: string) {
  currentTab.value = key;
  emit("update:tab", key);
}

function handleClose() {
  emit("close");
  innerOpen.value = false;
}

function handleEdit() {
  emit("edit");
}

function handleDelete() {
  emit("delete");
}

function handleRefresh() {
  emit("refresh");
}
</script>

<template>
  <ui-drawer
    :visible="innerOpen"
    :footer="false"
    :header="false"
    :closable="false"
    placement="right"
    class="detail-drawer"
    @cancel="handleClose"
  >
    <!-- Header -->
    <div class="flex flex-shrink-0 items-center justify-between border-b border-slate-200 bg-slate-50 px-6 py-5">
      <div class="flex min-w-0 flex-1 items-center gap-3">
        <!-- 44px tap zone via before: -->
        <button
          type="button"
          aria-label="关闭详情面板"
          class="relative inline-flex h-9 w-9 items-center justify-center rounded-lg text-slate-400 transition-all hover:bg-slate-100 hover:text-slate-900 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--accent)]/50 focus-visible:ring-offset-1 before:absolute before:-inset-[4px] before:content-['']"
          @click="handleClose"
        >
          <icon-close aria-hidden="true" />
        </button>
        <div class="flex flex-col gap-1">
          <h2 class="m-0 text-lg font-semibold tracking-[-0.03em] text-slate-900">{{ title || "详情" }}</h2>
          <p v-if="$slots.subtitle" class="text-xs text-slate-500">
            <slot name="subtitle" />
          </p>
        </div>
      </div>

      <div v-if="showActions" class="flex flex-shrink-0 flex-wrap items-center justify-end gap-3 max-md:w-full max-md:justify-start">
        <button
          type="button"
          aria-label="刷新"
          :disabled="loading"
          class="relative inline-flex h-9 w-9 items-center justify-center rounded-lg text-slate-500 transition-all hover:bg-slate-100 hover:text-slate-900 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--accent)]/50 focus-visible:ring-offset-1 disabled:cursor-not-allowed disabled:opacity-40 before:absolute before:-inset-[4px] before:content-['']"
          @click="handleRefresh"
        >
          <icon-refresh aria-hidden="true" />
        </button>
        <button
          type="button"
          aria-label="编辑"
          class="relative inline-flex h-9 w-9 items-center justify-center rounded-lg text-slate-500 transition-all hover:bg-slate-100 hover:text-slate-900 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--accent)]/50 focus-visible:ring-offset-1 before:absolute before:-inset-[4px] before:content-['']"
          @click="handleEdit"
        >
          <icon-edit aria-hidden="true" />
        </button>
        <button
          type="button"
          aria-label="删除"
          class="relative inline-flex h-9 w-9 items-center justify-center rounded-lg text-slate-500 transition-all hover:bg-red-50 hover:text-red-600 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-red-400/50 focus-visible:ring-offset-1 before:absolute before:-inset-[4px] before:content-['']"
          @click="handleDelete"
        >
          <icon-delete aria-hidden="true" />
        </button>
      </div>
    </div>

    <!-- Tabs -->
    <div
      v-if="resolvedTabs.length"
      role="tablist"
      :aria-label="title ? `${title} 标签页` : '详情标签页'"
      class="flex flex-shrink-0 items-center gap-1 border-b border-slate-200 bg-slate-50/50 px-4 py-2"
    >
      <button type="button"
        v-for="tab in resolvedTabs"
        :key="tab.key"
        role="tab"
        :aria-selected="currentTab === tab.key"
        :aria-controls="`drawer-panel-${tab.key}`"
        class="flex cursor-pointer items-center gap-2 rounded-lg border border-transparent bg-transparent px-4 py-2 text-[14px] font-medium text-slate-500 transition-all hover:bg-slate-100 hover:text-slate-700 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--accent)]/50 focus-visible:ring-offset-1"
        :class="{ 'bg-white shadow-sm border-slate-200 text-[var(--accent)] font-semibold': currentTab === tab.key }"
        @click="setTab(tab.key)"
      >
        <component v-if="tab.icon" :is="tab.icon" class="h-3.5 w-3.5" aria-hidden="true" />
        {{ tab.label }}
      </button>
    </div>

    <!-- Body: keyed Transition for tab fade -->
    <div class="flex-1 overflow-y-auto p-6">
      <ui-spin v-if="loading" :loading="true" tip="加载中..." class="flex items-center justify-center py-12" />

      <Transition v-else name="tab-fade" mode="out-in">
        <div
          :key="currentTab"
          role="tabpanel"
          :id="`drawer-panel-${currentTab}`"
          class="drawer-panel"
        >
          <slot :name="currentTab">
            <ui-empty v-if="currentTab === 'activity'" description="暂无活动记录" />
            <ui-empty v-else-if="currentTab === 'settings'" description="暂无可设置项" />
            <div v-else class="py-8 text-center text-sm text-slate-500">暂无内容</div>
          </slot>
        </div>
      </Transition>
    </div>

    <!-- Footer -->
    <div v-if="$slots.footer" class="flex flex-shrink-0 items-center justify-end gap-2 border-t border-slate-200 bg-white/60 px-5 py-4">
      <slot name="footer" />
    </div>
  </ui-drawer>
</template>
