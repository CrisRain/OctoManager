<script setup lang="ts">
import { computed, ref, watch } from "vue";
import { useRouter } from "vue-router";
import {
  IconSchedule, IconClockCircle, IconHistory, IconPlayArrow,
  IconEdit, IconDelete, IconCopy, IconEye
} from "@/lib/icons";

import { PageHeader, SmartListBar, RowActionsMenu } from "@/components/index";
import { useJobDefinitions, useEnqueueJobExecution, useDeleteJobDefinition } from "@/composables/useJobs";
import { useMessage, useConfirm, useErrorHandler } from "@/composables";
import { to } from "@/router/registry";
import type { JobDefinition } from "@/types";

const router = useRouter();
const message = useMessage();
const confirm = useConfirm();
const { withErrorHandler } = useErrorHandler();

const { data: definitions, loading, refresh } = useJobDefinitions();
const selectedKeys = ref<string[]>([]);
watch(definitions, () => { selectedKeys.value = []; });
const enqueue = useEnqueueJobExecution();
const deleteJobOp = useDeleteJobDefinition();

// 筛选和搜索
const scheduleFilter = ref<string>();
const searchKeyword = ref("");

// 过滤任务
const filteredJobs = computed(() => {
  let result = definitions.value;

  // 按调度类型筛选
  if (scheduleFilter.value === "scheduled") {
    result = result.filter(item => item.schedule?.cron_expression);
  } else if (scheduleFilter.value === "manual") {
    result = result.filter(item => !item.schedule?.cron_expression);
  }

  // 搜索过滤
  if (searchKeyword.value) {
    const keyword = searchKeyword.value.toLowerCase();
    result = result.filter(item =>
      item.name.toLowerCase().includes(keyword) ||
      item.key.toLowerCase().includes(keyword) ||
      item.plugin_key.toLowerCase().includes(keyword)
    );
  }

  return result;
});

// 抽屉状态
const drawerVisible = ref(false);
const currentJob = ref<JobDefinition | null>(null);

// 快速操作
async function handleQuickAction(key: string, job: JobDefinition) {
  switch (key) {
    case "view":
      router.push(to.jobs.detail(job.id));
      break;
    case "edit":
      router.push(to.jobs.edit(job.id));
      break;
    case "execute":
      await handleEnqueue(job.id);
      break;
    case "copy":
      await copyToClipboard(job.key);
      break;
    case "delete":
      await deleteJob(job);
      break;
  }
}

// 立即执行任务
async function handleEnqueue(id: number) {
  await withErrorHandler(
    async () => {
      await enqueue.execute(id);
      message.success("已加入执行队列");
      await refresh();
    },
    { action: "执行任务", showSuccess: false }
  );
}

// 复制到剪贴板
async function copyToClipboard(text: string) {
  try {
    await navigator.clipboard.writeText(text);
    message.success("已复制到剪贴板");
  } catch {
    message.error("复制失败");
  }
}

// 删除任务
async function deleteJob(job: JobDefinition) {
  const confirmed = await confirm.confirmDelete(job.name);
  if (!confirmed) return;

  await withErrorHandler(
    async () => {
      await deleteJobOp.execute(job.id);
      message.success(`已删除任务: ${job.name}`);
    },
    { action: "删除", showSuccess: false }
  );
}

// 批量删除
async function handleBatchDelete(items: JobDefinition[]) {
  const confirmed = await confirm.confirm(`确定要删除选中的 ${items.length} 个任务吗？`);
  if (!confirmed) return;

  await withErrorHandler(
    async () => {
      await Promise.all(items.map((item) => deleteJobOp.execute(item.id)));
      message.success(`已删除 ${items.length} 个任务`);
    },
    { action: "批量删除", showSuccess: false }
  );
}

// 批量导出
async function handleBatchExport(items: JobDefinition[]) {
  const data = JSON.stringify(items, null, 2);
  const blob = new Blob([data], { type: "application/json" });
  const url = URL.createObjectURL(blob);
  const link = document.createElement("a");
  link.href = url;
  link.download = `jobs-${Date.now()}.json`;
  link.click();
  URL.revokeObjectURL(url);
  message.success(`已导出 ${items.length} 个任务`);
}
</script>

<template>
  <div class="page-shell">
    <PageHeader
      title="定时任务"
      subtitle="管理自动化任务，支持 Cron 定时调度和手动触发"
      icon-bg="linear-gradient(135deg, rgba(202,138,4,0.12), rgba(234,179,8,0.12))"
      icon-color="var(--icon-yellow)"
    >
      <template #icon><icon-schedule /></template>
      <template #actions>
        <ui-button type="primary" @click="router.push(to.jobs.create())">
          <template #icon><icon-plus /></template>
          创建任务
        </ui-button>
      </template>
    </PageHeader>

    <!-- 智能工具栏 -->
    <SmartListBar
      :data="filteredJobs"
      :loading="loading"
      v-model:search="searchKeyword"
      v-model:selectedKeys="selectedKeys"
      @refresh="refresh"
      @batch-delete="handleBatchDelete"
      @batch-export="handleBatchExport"
    >
      <template #filters>
        <!-- 调度类型筛选 -->
        <div class="flex flex-wrap items-center gap-2">
          <span class="text-xs font-medium text-slate-500">调度：</span>
          <div class="flex flex-wrap items-center gap-1">
            <button type="button"
              class="filter-chip"
              :class="{ active: !scheduleFilter }"
              @click="scheduleFilter = ''"
            >
              全部
            </button>
            <button type="button"
              class="filter-chip"
              :class="{ active: scheduleFilter === 'scheduled' }"
              @click="scheduleFilter = 'scheduled'"
            >
              定时
            </button>
            <button type="button"
              class="filter-chip"
              :class="{ active: scheduleFilter === 'manual' }"
              @click="scheduleFilter = 'manual'"
            >
              手动
            </button>
          </div>
        </div>
      </template>

      <template #extra-actions>
        <ui-button @click="router.push(to.jobs.executions())">
          <template #icon><icon-history /></template>
          执行记录
        </ui-button>
      </template>
    </SmartListBar>

    <!-- 数据表格 -->
    <ui-card class="mb-4 hidden lg:block">
      <ui-table

        :data="filteredJobs"
        :loading="loading"
        :pagination="{
          showTotal: true,
          pageSizeOptions: [10, 20, 50],
          defaultPageSize: 20,
        }"
        :bordered="false"
        row-key="id"
        :row-selection="{ type: 'checkbox' }"
        v-model:selectedKeys="selectedKeys"
      >
        <template #columns>
          <!-- ID -->
          <ui-table-column title="ID" width="80">
            <template #cell="{ record }">
              <code class="text-xs font-mono font-semibold text-slate-500">#{{ record.id }}</code>
            </template>
          </ui-table-column>

          <!-- 任务名称 -->
          <ui-table-column title="任务名称" data-index="name">
            <template #cell="{ record }">
              <div class="flex items-center gap-3">
                <div class="flex h-9 w-9 flex-shrink-0 items-center justify-center rounded-lg border border-yellow-200 bg-yellow-50 text-sm text-yellow-600 shadow-sm">
                  <icon-schedule />
                </div>
                <div class="flex min-w-0 flex-col gap-0.5">
                  <div class="truncate text-[14px] font-medium text-slate-900">{{ record.name }}</div>
                  <code v-if="record.key" class="text-xs text-slate-500 mono">{{ record.key }}</code>
                </div>
              </div>
            </template>
          </ui-table-column>

          <!-- 插件 · 动作 -->
          <ui-table-column title="插件 · 动作">
            <template #cell="{ record }">
              <span class="inline-flex flex-wrap items-center gap-2">
                <span class="inline-flex items-center rounded-full border px-2.5 py-1 text-xs font-mono font-semibold text-slate-700 border-slate-200 bg-white/65">{{ record.plugin_key }}</span>
                <span class="text-slate-400">·</span>
                <span class="inline-flex items-center rounded border border-slate-200 bg-slate-100 px-1.5 py-0.5 text-xs font-mono text-slate-600">{{ record.action }}</span>
              </span>
            </template>
          </ui-table-column>

          <!-- 调度计划 -->
          <ui-table-column title="调度计划">
            <template #cell="{ record }">
              <span v-if="record.schedule?.cron_expression" class="inline-flex items-center gap-2 rounded-xl border border-amber-200 bg-amber-50 px-3 py-1.5 text-xs font-mono font-semibold text-amber-700">
                <icon-clock-circle class="h-3.5 w-3.5" />
                {{ record.schedule.cron_expression }}
              </span>
              <span v-else class="text-sm text-slate-400">手动触发</span>
            </template>
          </ui-table-column>

          <!-- 快速操作 -->
          <ui-table-column title="操作" align="right">
            <template #cell="{ record }">
              <RowActionsMenu
                :item="record"
                :actions="[
                  { key: 'view', label: '查看详情', icon: 'IconEye' },
                  { key: 'execute', label: '立即执行', icon: 'IconPlayArrow' },
                  { key: 'edit', label: '编辑', icon: 'IconEdit' },
                  { key: 'copy', label: '复制Key', icon: 'IconCopy' },
                  { key: 'delete-divider', divider: true },
                  { key: 'delete', label: '删除', icon: 'IconDelete', danger: true },
                ]"
                @action="handleQuickAction"
              />
            </template>
          </ui-table-column>
        </template>

        <!-- 空状态 -->
        <template #empty>
          <ui-empty
            description="还没有任何任务。任务是自动化的核心，定义要执行什么操作。"
            :workflow-steps="[{label:'插件'},{label:'账号类型'},{label:'账号'},{label:'任务'}]"
          >
            <ui-button type="primary" @click="router.push(to.jobs.create())">
              创建第一个任务
            </ui-button>
          </ui-empty>
        </template>
      </ui-table>
    </ui-card>

    <div class="flex flex-col gap-3 lg:hidden">
      <ui-card
        v-for="record in filteredJobs"
        :key="record.id"
        v-memo="[record.id, record.name, record.key, record.plugin_key, record.action, record.schedule?.cron_expression]"
        class="rounded-xl border p-5 border-slate-200 bg-white shadow-sm"
      >
        <div class="mb-3 flex items-center gap-3">
          <div class="flex h-9 w-9 flex-shrink-0 items-center justify-center rounded-lg border border-yellow-200 bg-yellow-50 text-sm text-yellow-600 shadow-sm">
            <icon-schedule />
          </div>
          <div class="flex min-w-0 flex-1 flex-col gap-0.5">
            <div class="truncate text-sm font-semibold text-slate-900">{{ record.name }}</div>
            <div class="text-xs text-slate-500">
              <code class="text-xs text-slate-500 mono">#{{ record.id }}<template v-if="record.key"> · {{ record.key }}</template></code>
            </div>
          </div>
          <RowActionsMenu
            :item="record"
            :actions="[
              { key: 'view', label: '查看详情', icon: 'IconEye' },
              { key: 'execute', label: '立即执行', icon: 'IconPlayArrow' },
              { key: 'edit', label: '编辑', icon: 'IconEdit' },
              { key: 'copy', label: '复制Key', icon: 'IconCopy' },
              { key: 'delete-divider', divider: true },
              { key: 'delete', label: '删除', icon: 'IconDelete', danger: true },
            ]"
            @action="handleQuickAction"
          />
        </div>

        <div class="flex flex-col gap-2">
          <div class="flex items-center justify-between gap-2">
            <span class="w-12 flex-shrink-0 text-xs font-medium text-slate-500">插件</span>
            <div class="flex flex-wrap items-center gap-1">
              <span class="inline-flex flex-wrap items-center gap-2">
                <span class="inline-flex items-center rounded-full border px-2.5 py-1 text-xs font-mono font-semibold text-slate-700 border-slate-200 bg-white/65">{{ record.plugin_key }}</span>
                <span class="text-slate-400">·</span>
                <span class="inline-flex items-center rounded border border-slate-200 bg-slate-100 px-1.5 py-0.5 text-xs font-mono text-slate-600">{{ record.action }}</span>
              </span>
            </div>
          </div>

          <div class="flex items-center justify-between gap-2">
            <span class="w-12 flex-shrink-0 text-xs font-medium text-slate-500">调度</span>
            <div class="flex flex-wrap items-center gap-1">
              <span v-if="record.schedule?.cron_expression" class="inline-flex items-center gap-2 rounded-xl border border-amber-200 bg-amber-50 px-3 py-1.5 text-xs font-mono font-semibold text-amber-700">
                <icon-clock-circle class="h-3.5 w-3.5" />
                {{ record.schedule.cron_expression }}
              </span>
              <span v-else class="text-sm text-slate-400">手动触发</span>
            </div>
          </div>
        </div>
      </ui-card>

      <ui-card
        v-if="!loading && !filteredJobs.length"
        class="col-span-full empty-state-block"
      >
        <ui-empty
          description="还没有任何任务。任务是自动化的核心，定义要执行什么操作。"
          :workflow-steps="[{label:'插件'},{label:'账号类型'},{label:'账号'},{label:'任务'}]"
        >
          <ui-button type="primary" @click="router.push(to.jobs.create())">
            创建第一个任务
          </ui-button>
        </ui-empty>
      </ui-card>
    </div>
  </div>
</template>
