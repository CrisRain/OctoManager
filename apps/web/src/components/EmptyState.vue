<script setup lang="ts">
import { computed } from "vue";
import {
  IconFile, IconUser, IconEmail, IconCheckCircle,
  IconCloseCircle, IconRefresh, IconPlus, IconArrowRight
} from "@/lib/icons";

interface WorkflowStep {
  label: string;
  description?: string;
}

interface Props {
  type?: "empty" | "error" | "success" | "loading";
  title?: string;
  description?: string;
  icon?: any;
  actionText?: string;
  hideAction?: boolean;
  /** 操作流程步骤，显示在描述下方，帮助用户了解下一步 */
  workflowSteps?: WorkflowStep[];
}

const props = withDefaults(defineProps<Props>(), {
  type: "empty",
  description: "",
  hideAction: false,
});

const emit = defineEmits<{
  (e: "action"): void;
}>();

/**
 * 默认图标配置
 */
const defaultIcons = {
  empty: IconFile,
  error: IconCloseCircle,
  success: IconCheckCircle,
  loading: IconRefresh,
};

/**
 * 默认标题配置
 */
const defaultTitles = {
  empty: "暂无数据",
  error: "出错了",
  success: "操作成功",
  loading: "加载中...",
};

/**
 * 默认描述配置
 */
const defaultDescriptions = {
  empty: "这里还没有任何内容",
  error: "加载失败，请稍后重试",
  success: "操作已完成",
  loading: "正在努力加载中...",
};

const displayIcon = computed(() => props.icon || defaultIcons[props.type]);
const displayTitle = computed(() => props.title || defaultTitles[props.type]);
const displayDescription = computed(() => props.description || defaultDescriptions[props.type]);
const isLoading = computed(() => props.type === "loading");

function handleAction() {
  emit("action");
}
</script>

<template>
  <div class="flex flex-col items-center gap-4 px-6 py-16 text-center" :class="{ 'py-12': isLoading }">
    <div class="flex h-14 w-14 items-center justify-center rounded-xl border border-slate-200 bg-white shadow-sm" :class="{
      'text-slate-400': type === 'empty',
      'text-red-500 border-red-100 bg-red-50/50': type === 'error',
      'text-emerald-500 border-emerald-100 bg-emerald-50/50': type === 'success',
      'text-blue-500 border-blue-100 bg-blue-50/50': type === 'loading',
    }">
      <component :is="displayIcon" class="h-6 w-6" :class="{ 'animate-spin': isLoading }" />
    </div>
    <div class="flex flex-col gap-2">
      <h3 class="m-0 text-lg font-semibold tracking-tight text-slate-900">{{ displayTitle }}</h3>
      <slot name="description">
        <p v-if="displayDescription" class="text-sm text-slate-500 max-w-[280px] mx-auto">
          {{ displayDescription }}
        </p>
      </slot>
    </div>
    <!-- 操作流程提示 -->
    <div v-if="workflowSteps?.length" class="mt-1 flex flex-wrap items-center justify-center gap-1 text-xs text-slate-400">
      <template v-for="(step, i) in workflowSteps" :key="i">
        <span class="rounded-md border border-slate-200 bg-white px-2 py-1 font-medium text-slate-600">{{ step.label }}</span>
        <icon-arrow-right v-if="i < workflowSteps.length - 1" class="h-3 w-3 flex-shrink-0" />
      </template>
    </div>
    <div v-if="!hideAction" class="mt-4">
      <slot name="action">
        <ui-button v-if="actionText" type="primary" @click="handleAction">
          <template #icon>
            <icon-plus />
          </template>
          {{ actionText }}
        </ui-button>
      </slot>
    </div>
  </div>
</template>
