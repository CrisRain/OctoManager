<script setup lang="ts">
import { computed, ref } from "vue";
import { useRouter } from "vue-router";
import { IconHistory, IconRefresh, IconEye } from "@/lib/icons";

import { PageHeader, SmartListBar, RowActionsMenu, StatusTag } from "@/components/index";
import { useJobExecutions } from "@/composables/useJobs";
import { useMessage } from "@/composables";
import { to } from "@/router/registry";

const router = useRouter();
const message = useMessage();

const { data: executions, loading, refresh } = useJobExecutions();

// 状态筛选
const statusFilter = ref<string>();
const searchKeyword = ref("");

// 过滤执行记录
const filteredExecutions = computed(() => {
  let result = executions.value;

  if (statusFilter.value) {
    result = result.filter(item => item.status === statusFilter.value);
  }

  if (searchKeyword.value) {
    const keyword = searchKeyword.value.toLowerCase();
    result = result.filter(item =>
      String(item.id).includes(keyword) ||
      item.definition_name?.toLowerCase().includes(keyword) ||
      item.plugin_key?.toLowerCase().includes(keyword) ||
      item.action?.toLowerCase().includes(keyword)
    );
  }

  return result;
});

// 状态配置
const statusOptions = [
  { label: "全部", value: "" },
  { label: "待执行", value: "pending" },
  { label: "运行中", value: "running" },
  { label: "成功", value: "success" },
  { label: "失败", value: "failed" },
  { label: "已取消", value: "cancelled" },
];

// 抽屉状态
const drawerVisible = ref(false);
const currentExecution = ref<any>(null);

// 快速操作
function handleQuickAction(key: string, execution: any) {
  switch (key) {
    case "view":
      goToDetail(execution.id);
      break;
    case "retry":
      // TODO: 实现重试逻辑
      message.info("重试功能开发中");
      break;
  }
}

// 跳转到详情页（保留原有导航）
function goToDetail(id: number) {
  router.push(to.jobs.executionDetail(id));
}

// 复制ID
async function copyId(id: number) {
  try {
    await navigator.clipboard.writeText(String(id));
    message.success("已复制ID");
  } catch {
    message.error("复制失败");
  }
}
</script>

<template>
  <div class="page-shell executions-list-page">
    <PageHeader
      title="执行记录"
      subtitle="查看所有任务的历史执行状态和日志"
      icon-bg="linear-gradient(135deg, rgba(10,132,255,0.12), rgba(10,132,255,0.06))"
      icon-color="var(--icon-purple)"
    >
      <template #icon><icon-history /></template>
      <template #actions>
        <ui-button @click="router.push(to.jobs.list())">返回任务管理</ui-button>
      </template>
    </PageHeader>

    <!-- 智能工具栏 -->
    <SmartListBar
      :data="filteredExecutions"
      :loading="loading"
      v-model:search="searchKeyword"
      @refresh="refresh"
    >
      <template #filters>
        <!-- 状态筛选 -->
        <div class="flex flex-wrap items-center gap-2">
          <span class="text-xs font-medium text-slate-500">状态：</span>
          <div class="flex flex-wrap items-center gap-1">
            <button type="button"
              v-for="option in statusOptions"
              :key="option.value"
              class="filter-chip"
              :class="{ active: statusFilter === option.value }"
              @click="statusFilter = option.value"
            >
              {{ option.label }}
            </button>
          </div>
        </div>
      </template>
    </SmartListBar>

    <!-- 数据表格 -->
    <ui-card class="mb-4 hidden lg:block">
      <ui-table
        :data="filteredExecutions"
        :loading="loading"
        :pagination="{
          showTotal: true,
          pageSizeOptions: [10, 20, 50],
          defaultPageSize: 20,
        }"
        :bordered="false"
        row-key="id"
        :row-selection="{ type: 'checkbox' }"
      >
        <template #columns>
          <!-- 编号 -->
          <ui-table-column title="编号">
            <template #cell="{ record }">
              <code class="text-xs font-mono font-semibold text-slate-500">#{{ record.id }}</code>
            </template>
          </ui-table-column>

          <!-- 任务名称 -->
          <ui-table-column title="任务名称" data-index="definition_name">
            <template #cell="{ record }">
              <span class="text-sm font-medium text-slate-900">{{ record.definition_name }}</span>
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

          <!-- 状态 -->
          <ui-table-column title="状态">
            <template #cell="{ record }">
              <StatusTag :status="record.status" />
            </template>
          </ui-table-column>

          <!-- 节点 -->
          <ui-table-column title="节点">
            <template #cell="{ record }">
              <code v-if="record.worker_id" class="inline-flex items-center rounded border border-slate-200 bg-slate-50 px-1.5 py-0.5 text-xs font-mono text-sky-700">{{ record.worker_id }}</code>
              <span v-else class="text-sm text-slate-400">未分配</span>
            </template>
          </ui-table-column>

          <!-- 快速操作 -->
          <ui-table-column title="操作" align="right">
            <template #cell="{ record }">
              <RowActionsMenu
                :item="record"
                :actions="[
                  { key: 'view', label: '查看详情', icon: 'IconEye' },
                ]"
                @action="handleQuickAction"
              />
            </template>
          </ui-table-column>
        </template>

        <!-- 空状态 -->
        <template #empty>
          <ui-empty description="暂无执行记录">
            <ui-button type="primary" @click="router.push(to.jobs.list())">
              创建并执行任务
            </ui-button>
          </ui-empty>
        </template>
      </ui-table>
    </ui-card>

    <div class="flex flex-col gap-3 lg:hidden">
      <ui-card
        v-for="record in filteredExecutions"
        :key="record.id"
        class="rounded-xl border p-5 border-slate-200 bg-white shadow-sm"
      >
        <div class="mb-3 flex items-center gap-3">
          <div class="flex h-9 w-9 flex-shrink-0 items-center justify-center rounded-lg border border-purple-200 bg-purple-50 text-sm text-purple-600 shadow-sm">
            <icon-history />
          </div>
          <div class="flex min-w-0 flex-1 flex-col gap-0.5">
            <div class="truncate text-sm font-semibold text-slate-900">{{ record.definition_name || `执行记录 #${record.id}` }}</div>
            <div class="text-xs text-slate-500">
              <code class="text-xs font-mono font-semibold text-slate-500">#{{ record.id }}</code>
            </div>
          </div>
          <RowActionsMenu
            :item="record"
            :actions="[
              { key: 'view', label: '查看详情', icon: 'IconEye' },
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
            <span class="w-12 flex-shrink-0 text-xs font-medium text-slate-500">状态</span>
            <div class="flex flex-wrap items-center gap-1">
              <StatusTag :status="record.status" />
            </div>
          </div>

          <div class="flex items-center justify-between gap-2">
            <span class="w-12 flex-shrink-0 text-xs font-medium text-slate-500">节点</span>
            <div class="flex flex-wrap items-center gap-1">
              <code v-if="record.worker_id" class="inline-flex items-center rounded border border-slate-200 bg-slate-50 px-1.5 py-0.5 text-xs font-mono text-sky-700">{{ record.worker_id }}</code>
              <span v-else class="text-sm text-slate-400">未分配</span>
            </div>
          </div>
        </div>
      </ui-card>

      <ui-card
        v-if="!loading && !filteredExecutions.length"
        class="col-span-full empty-state-block"
      >
        <ui-empty description="暂无执行记录">
          <ui-button type="primary" @click="router.push(to.jobs.list())">
            创建并执行任务
          </ui-button>
        </ui-empty>
      </ui-card>
    </div>
  </div>
</template>
