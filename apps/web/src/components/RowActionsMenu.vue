<script setup lang="ts">
import { ref, computed } from "vue";
import {
  IconMore,
  IconEdit,
  IconDelete,
  IconEye,
  IconCopy,
  IconPlayArrow,
  IconStop,
  IconRefresh,
} from "@/lib/icons";

interface Action {
  key: string;
  label?: string;
  icon?: any;
  danger?: boolean;
  divider?: boolean;
  disabled?: boolean;
}

interface Props {
  actions?: Action[];
  item?: any;
}

const props = withDefaults(defineProps<Props>(), {
  actions: () => [],
});

const emit = defineEmits<{
  (e: "action", key: string, item: any): void;
}>();

const visible = ref(false);

// 默认操作
const defaultActions: Action[] = [
  { key: "view", label: "查看详情", icon: IconEye },
  { key: "edit", label: "编辑", icon: IconEdit },
  { key: "copy", label: "复制", icon: IconCopy },
  { key: "divider-delete", divider: true },
  { key: "delete", label: "删除", icon: IconDelete, danger: true },
];

const iconMap: Record<string, any> = {
  IconEye,
  IconEdit,
  IconDelete,
  IconCopy,
  IconPlayArrow,
  IconStop,
  IconRefresh,
};

function resolveIcon(icon?: any) {
  if (!icon) return undefined;
  if (typeof icon === "string") return iconMap[icon];
  return icon;
}

// Legacy callers may pass key: "text-slate-300" as a divider marker (CSS class used as sentinel).
// Normalise these to proper dividers for backward compatibility.
const LEGACY_DIVIDER_KEY = "text-slate-300";

const allActions = computed(() => {
  const source = props.actions.length > 0 ? props.actions : defaultActions;
  const normalized = source
    .map((action, index) => {
      if (action.divider || action.key === LEGACY_DIVIDER_KEY) {
        return { key: `${action.key || LEGACY_DIVIDER_KEY}-${index}`, divider: true } satisfies Action;
      }
      return action;
    })
    .filter((action) => action.divider || action.label);

  return normalized.filter((action, index) => {
    if (!action.divider) return true;
    const prev = normalized[index - 1];
    const next = normalized[index + 1];
    return Boolean(prev && next && !prev.divider && !next.divider);
  });
});

function handleAction(key: string) {
  visible.value = false;
  emit("action", key, props.item);
}

function handleMenuClick(e: Event) {
  e.stopPropagation();
}
</script>

<template>
  <ui-dropdown class="row-actions-dropdown" :visible="visible" @popup-visible-change="visible = $event" @click="handleMenuClick">
    <!-- Touch target extended to 44px via before: pseudo-element -->
    <button
      type="button"
      aria-label="更多操作"
      :aria-expanded="visible"
      aria-haspopup="menu"
      class="relative inline-flex h-8 w-8 items-center justify-center rounded-md text-slate-500 transition-colors hover:bg-slate-100 hover:text-slate-900 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--accent)]/50 focus-visible:ring-offset-1 before:absolute before:-inset-[6px] before:content-['']"
    >
      <icon-more aria-hidden="true" />
    </button>
    <template #content>
      <div class="min-w-[140px] py-1.5" role="menu" aria-label="行操作菜单">
        <template v-for="action in allActions" :key="action.key">
          <div v-if="action.divider" class="my-1 border-t border-slate-200" role="separator" />
          <button
            v-else
            type="button"
            role="menuitem"
            class="flex w-full cursor-pointer items-center gap-2.5 rounded-lg border-0 bg-transparent px-3 py-2 text-left text-[14px] text-slate-700 transition-colors hover:bg-slate-100 focus-visible:outline-none focus-visible:bg-slate-100 disabled:cursor-not-allowed disabled:opacity-40"
            :class="{ 'text-red-600 hover:bg-red-50 focus-visible:bg-red-50': action.danger }"
            :disabled="action.disabled"
            @click="handleAction(action.key)"
          >
            <span class="flex h-4 w-4 flex-shrink-0 items-center justify-center" aria-hidden="true">
              <component v-if="action.icon" :is="resolveIcon(action.icon)" class="h-4 w-4" />
            </span>
            <span class="flex-1">{{ action.label }}</span>
          </button>
        </template>
      </div>
    </template>
  </ui-dropdown>
</template>
