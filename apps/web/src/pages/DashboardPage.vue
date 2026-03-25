<script setup lang="ts">
import { computed } from "vue";
import { useRouter } from "vue-router";
import {
  IconApps, IconLayers, IconUser, IconEmail,
  IconSchedule, IconHistory, IconRobot, IconRefresh, IconLoading,
  IconPlus, IconSettings, IconArrowRight, IconThunderbolt, IconCheckCircle
} from "@/lib/icons";

import { useSystemStatus, useDashboardSnapshot } from "@/composables/useDashboard";
import { useCommandPaletteStore } from "@/store/command-palette";
import { to } from "@/router/registry";
import type { DashboardSummary } from "@/composables/useDashboard";

const router = useRouter();
const systemStatus = useSystemStatus();
const snapshot = useDashboardSnapshot();
const commandPalette = useCommandPaletteStore();

// 新手引导：当账号类型、账号、任务都为 0 时显示
const isNewUser = computed(() => {
  const d = snapshot.data.value;
  if (!d) return false;
  return d.accountTypeCount === 0 && d.accountCount === 0 && d.jobDefinitionCount === 0;
});

interface OnboardingStep {
  step: number;
  label: string;
  description: string;
  path: string;
  done: boolean;
}

const onboardingSteps = computed((): OnboardingStep[] => {
  const d = snapshot.data.value;
  return [
    {
      step: 1,
      label: "安装插件",
      description: "插件提供具体的自动化能力（如 GitHub、邮件等）",
      path: to.plugins.list(),
      done: (d?.pluginCount ?? 0) > 0,
    },
    {
      step: 2,
      label: "创建账号类型",
      description: "定义账号的凭证字段结构",
      path: to.accountTypes.create(),
      done: (d?.accountTypeCount ?? 0) > 0,
    },
    {
      step: 3,
      label: "添加账号",
      description: "录入需要自动化管理的账号信息",
      path: to.accounts.create(),
      done: (d?.accountCount ?? 0) > 0,
    },
    {
      step: 4,
      label: "创建任务",
      description: "配置自动化任务，设置触发方式与执行插件",
      path: to.jobs.create(),
      done: (d?.jobDefinitionCount ?? 0) > 0,
    },
  ];
});

// 统计卡片配置
interface StatCard {
  label: string;
  valueKey: keyof Omit<DashboardSummary, "recentExecutions">;
  icon: any;
  /** CSS class from the .icon-* utility set defined in tailwind.css */
  iconClass: string;
  path?: string;
}

const statCards: StatCard[] = [
  {
    label: "插件",
    valueKey: "pluginCount",
    icon: IconApps,
    iconClass: "icon-teal",
    path: to.plugins.list()
  },
  {
    label: "账号类型",
    valueKey: "accountTypeCount",
    icon: IconLayers,
    iconClass: "icon-blue",
    path: to.accountTypes.list()
  },
  {
    label: "账号",
    valueKey: "accountCount",
    icon: IconUser,
    iconClass: "icon-green",
    path: to.accounts.list()
  },
  {
    label: "邮箱账号",
    valueKey: "emailAccountCount",
    icon: IconEmail,
    iconClass: "icon-orange",
    path: to.emailAccounts.list()
  },
  {
    label: "任务定义",
    valueKey: "jobDefinitionCount",
    icon: IconSchedule,
    iconClass: "icon-yellow",
    path: to.jobs.list()
  },
  {
    label: "执行总数",
    valueKey: "jobExecutionCount",
    icon: IconHistory,
    iconClass: "icon-red",
    path: to.jobs.executions()
  },
  {
    label: "Agent",
    valueKey: "agentCount",
    icon: IconRobot,
    iconClass: "icon-cyan",
    path: to.agents.list()
  },
];

// 状态颜色映射
const executionStatusColor: Record<string, string> = {
  pending: "gray",
  running: "blue",
  done: "green",
  success: "green",
  failed: "red",
  cancelled: "gray",
};

function statusColor(status: string): string {
  return executionStatusColor[status] ?? "gray";
}

function executionIcon(status: string) {
  return status === "running" ? IconLoading : IconHistory;
}

// 快速操作配置
interface QuickAction {
  id: string;
  label: string;
  description: string;
  icon: any;
  color: string;
  path: string;
}

const quickActions: QuickAction[] = [
  { id: "add-account", label: "创建账号", description: "创建新的账号", icon: IconUser, color: "blue", path: to.accounts.create() },
  { id: "add-email", label: "创建邮箱账号", description: "创建新的邮箱账号", icon: IconEmail, color: "orange", path: to.emailAccounts.create() },
  { id: "create-job", label: "创建任务", description: "创建新的任务", icon: IconThunderbolt, color: "teal", path: to.jobs.create() },
  { id: "create-agent", label: "创建 Agent", description: "创建新的 Agent", icon: IconRobot, color: "cyan", path: to.agents.create() },
];

function openCommandPalette() {
  commandPalette.open();
}

function refreshDashboard() {
  systemStatus.refresh();
  snapshot.refresh();
}

// 获取图标组件
</script>

<template>
  <div class="page-shell">
    <PageHeader title="控制台" subtitle="查看系统概览并快速访问常用操作">
      <template #actions>
        <ui-button type="secondary" @click="refreshDashboard">
          <template #icon><icon-refresh /></template>
          刷新数据
        </ui-button>
      </template>
    </PageHeader>

    <!-- 系统状态横幅 -->
    <div
      class="mb-6 flex items-center gap-3 rounded-xl border px-5 py-4 text-sm"
      :class="systemStatus.data.value?.database_ok ? 'border-emerald-200 bg-emerald-50/90 text-emerald-800' : 'border-red-200 bg-red-50/90 text-red-700'"
    >
      <span
        class="inline-block h-2.5 w-2.5 flex-shrink-0 rounded-full"
        :class="systemStatus.data.value?.database_ok
          ? 'bg-emerald-500 animate-pulse'
          : 'bg-red-500'"
      />
      <div class="flex flex-1 items-center gap-2 max-md:flex-col max-md:items-start max-md:gap-2">
        <span class="font-semibold">
          {{ systemStatus.data.value?.database_ok ? '系统运行正常' : '系统异常，请检查服务状态' }}
        </span>
        <span v-if="systemStatus.data.value?.database_ok" class="text-xs opacity-70">
          最后检测：{{ new Date(systemStatus.data.value.now).toLocaleString("zh-CN") }}
        </span>
      </div>
    </div>

    <!-- 新手引导卡片：仅在还没有核心数据时显示 -->
    <div
      v-if="isNewUser"
      class="mb-6 rounded-xl border border-blue-200 bg-blue-50/80 p-5"
    >
      <div class="mb-4 flex items-center gap-2">
        <div class="flex h-7 w-7 items-center justify-center rounded-lg bg-blue-100 text-blue-600">
          <icon-schedule class="h-4 w-4" />
        </div>
        <div>
          <p class="text-sm font-semibold text-blue-900">快速开始</p>
          <p class="text-xs text-blue-600">按照以下步骤完成系统初始化</p>
        </div>
      </div>
      <div class="grid grid-cols-1 gap-2 sm:grid-cols-2 lg:grid-cols-4">
        <button
          v-for="step in onboardingSteps"
          :key="step.step"
          type="button"
          class="group flex items-start gap-3 rounded-lg border bg-white/80 px-4 py-3 text-left transition-all hover:border-blue-300 hover:bg-white hover:shadow-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-400/40"
          :class="step.done ? 'border-emerald-200 opacity-70' : 'border-blue-200'"
          @click="!step.done && router.push(step.path)"
        >
          <div
            class="mt-0.5 flex h-5 w-5 flex-shrink-0 items-center justify-center rounded-full text-xs font-bold"
            :class="step.done ? 'bg-emerald-100 text-emerald-600' : 'bg-blue-100 text-blue-600'"
          >
            <icon-check-circle v-if="step.done" class="h-4 w-4" />
            <span v-else>{{ step.step }}</span>
          </div>
          <div class="min-w-0">
            <p class="text-sm font-medium" :class="step.done ? 'text-slate-400 line-through' : 'text-slate-800'">{{ step.label }}</p>
            <p class="mt-0.5 text-xs text-slate-500 leading-relaxed">{{ step.description }}</p>
          </div>
        </button>
      </div>
    </div>

    <!-- 统计卡片 - 可点击跳转 -->
    <div class="mb-7 grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-4 2xl:grid-cols-7">
      <button
        type="button"
        v-for="card in statCards"
        :key="card.label"
        class="group relative flex flex-col items-center gap-2 rounded-xl border border-slate-200 bg-white p-5 text-center shadow-sm transition-all duration-200"
        :class="card.path ? 'cursor-pointer hover:border-slate-300 hover:shadow-md hover:-translate-y-0.5 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--accent)]/20' : 'cursor-default'"
        @click="card.path && router.push(card.path)"
      >
        <div
          class="mb-1 flex h-11 w-11 flex-shrink-0 items-center justify-center rounded-xl transition-transform duration-200 group-hover:scale-110"
          :class="card.iconClass"
        >
          <component :is="card.icon" class="h-5 w-5" />
        </div>
        <div class="text-[28px] font-semibold leading-none tracking-tighter text-slate-900">
          {{ (snapshot.data.value?.[card.valueKey] as number) ?? 0 }}
        </div>
        <div class="text-xs font-medium text-slate-500">{{ card.label }}</div>
        <icon-arrow-right v-if="card.path" class="absolute right-2.5 top-2.5 h-3.5 w-3.5 text-slate-300 transition-colors group-hover:text-slate-500" />
      </button>
    </div>

    <div class="grid grid-cols-1 gap-6 lg:grid-cols-[2fr_3fr]">
      <!-- 左侧：快捷操作面板 -->
      <QuickActionsPanel
        :actions="quickActions"
        @open-search="openCommandPalette"
      />

      <!-- 右侧：最近执行记录 -->
      <ui-card class="min-w-0">
        <template #title>
          <div class="flex items-center gap-2">
            <icon-history class="h-4 w-4 text-[var(--accent)]" />
            <span>最近执行</span>
          </div>
        </template>

        <ui-empty
          v-if="!snapshot.data.value?.recentExecutions?.length"
          description="暂无执行记录"
        />

        <div v-else class="flex flex-col gap-3">
          <button
            type="button"
            v-for="record in snapshot.data.value?.recentExecutions ?? []"
            :key="record.id"
            class="flex w-full items-center gap-3.5 rounded-xl border border-slate-200 bg-white px-3.5 py-3.5 text-left transition-all hover:bg-slate-50 hover:border-slate-300 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-slate-400/20"
            @click="router.push(to.jobs.executionDetail(record.id))"
          >
            <div
              class="flex h-9 w-9 flex-shrink-0 items-center justify-center rounded-lg border shadow-sm"
              :class="{
                'border-slate-200 bg-slate-50 text-slate-500': ['pending', 'cancelled'].includes(record.status) || !['pending', 'running', 'done', 'success', 'failed', 'cancelled'].includes(record.status),
                'border-blue-200 bg-blue-50 text-blue-600': record.status === 'running',
                'border-emerald-200 bg-emerald-50 text-emerald-600': ['done', 'success'].includes(record.status),
                'border-red-200 bg-red-50 text-red-600': record.status === 'failed'
              }"
            >
              <component :is="executionIcon(record.status)" />
            </div>

            <div class="min-w-0 flex-1">
              <div class="mb-2 flex items-center gap-2 justify-between">
                <div class="min-w-0 truncate text-sm font-medium text-slate-900">{{ record.definition_name }}</div>
                <ui-tag
                  :color="statusColor(record.status)"
                  size="small"
                  class="flex-shrink-0"
                >
                  {{ record.status }}
                </ui-tag>
              </div>

              <div class="flex flex-wrap items-center gap-1 text-sm text-slate-500">
                <span class="font-medium">{{ record.plugin_key }}</span>
                <span class="text-slate-400">·</span>
                <span class="truncate">{{ record.action }}</span>
                <span class="text-slate-400">·</span>
                <span class="truncate">{{ record.worker_id || "自动分配" }}</span>
              </div>
            </div>

            <icon-arrow-right class="h-4 w-4 flex-shrink-0 text-slate-400" />
          </button>
        </div>
      </ui-card>
    </div>
  </div>
</template>
