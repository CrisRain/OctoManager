<script setup lang="ts">
import { computed, ref, watch } from "vue";
import { useRouter } from "vue-router";
import {
  IconRobot, IconPlayArrow, IconStop, IconEye,
  IconEdit, IconDelete
} from "@/lib/icons";

import { PageHeader, SmartListBar, RowActionsMenu, StatusTag } from "@/components/index";
import { useAgents, useDeleteAgent, useStartAgent, useStopAgent } from "@/composables/useAgents";
import { useMessage, useConfirm, useErrorHandler } from "@/composables";
import { to } from "@/router/registry";
import type { Agent } from "@/types";

const router = useRouter();
const message = useMessage();
const confirm = useConfirm();
const { withErrorHandler } = useErrorHandler();

const { data: agents, loading, refresh } = useAgents();
const selectedKeys = ref<string[]>([]);
watch(agents, () => { selectedKeys.value = []; });
const startAgent = useStartAgent();
const stopAgent = useStopAgent();
const deleteAgentAction = useDeleteAgent();

// 状态筛选
const statusFilter = ref<string>();
const searchKeyword = ref("");

// 过滤 Agent
const filteredAgents = computed(() => {
  let result = agents.value;

  if (statusFilter.value) {
    result = result.filter(item => item.runtime_state === statusFilter.value);
  }

  if (searchKeyword.value) {
    const keyword = searchKeyword.value.toLowerCase();
    result = result.filter(item =>
      item.name?.toLowerCase().includes(keyword) ||
      item.plugin_key?.toLowerCase().includes(keyword) ||
      item.action?.toLowerCase().includes(keyword)
    );
  }

  return result;
});

// 状态配置
const statusOptions = [
  { label: "全部", value: "" },
  { label: "运行中", value: "running" },
  { label: "已停止", value: "stopped" },
  { label: "停止中", value: "stopping" },
  { label: "错误", value: "error" },
];

// 抽屉状态
const drawerVisible = ref(false);
const currentAgentId = ref<number | null>(null);
const currentAgent = computed(() =>
  agents.value.find((item) => item.id === currentAgentId.value) ?? null
);

// 快速操作
async function handleQuickAction(key: string, agent: Agent) {
  switch (key) {
    case "view":
      router.push(to.agents.detail(agent.id));
      break;
    case "start":
      await handleStart(agent.id);
      break;
    case "stop":
      await handleStop(agent.id);
      break;
    case "edit":
      router.push(to.agents.edit(agent.id));
      break;
    case "delete":
      await handleDeleteAgent(agent);
      break;
  }
}

// 启动 Agent
async function handleStart(id: number) {
  await withErrorHandler(
    async () => {
      await startAgent.execute(id);
      message.success("启动指令已发送");
      await refresh();
    },
    { action: "启动 Agent", showSuccess: false }
  );
}

// 停止 Agent
async function handleStop(id: number) {
  await withErrorHandler(
    async () => {
      await stopAgent.execute(id);
      message.success("停止指令已发送");
      await refresh();
    },
    { action: "停止 Agent", showSuccess: false }
  );
}

// 删除 Agent
async function handleDeleteAgent(agent: Agent) {
  const confirmed = await confirm.confirmDelete(agent.name);
  if (!confirmed) return;

  await withErrorHandler(
    async () => {
      await deleteAgentAction.execute(agent.id);
      message.success(`已删除 Agent: ${agent.name}`);
      if (currentAgentId.value === agent.id) {
        drawerVisible.value = false;
        currentAgentId.value = null;
      }
      await refresh();
    },
    { action: "删除", showSuccess: false }
  );
}

// 批量删除
async function handleBatchDelete(items: Agent[]) {
  if (!items.length) return;
  const confirmed = await confirm.confirm(`确定要删除选中的 ${items.length} 个 Agent 吗？`);
  if (!confirmed) return;

  await withErrorHandler(
    async () => {
      await Promise.all(items.map((item) => deleteAgentAction.execute(item.id)));
      message.success(`已删除 ${items.length} 个 Agent`);
      if (currentAgentId.value !== null && items.some((item) => item.id === currentAgentId.value)) {
        drawerVisible.value = false;
        currentAgentId.value = null;
      }
      await refresh();
    },
    { action: "批量删除", showSuccess: false }
  );
}

// 批量导出
async function handleBatchExport(items: Agent[]) {
  const data = JSON.stringify(items, null, 2);
  const blob = new Blob([data], { type: "application/json" });
  const url = URL.createObjectURL(blob);
  const link = document.createElement("a");
  link.href = url;
  link.download = `agents-${Date.now()}.json`;
  link.click();
  URL.revokeObjectURL(url);
  message.success(`已导出 ${items.length} 个 Agent`);
}
</script>

<template>
  <div class="page-shell agents-list-page">
    <PageHeader
      title="Agent 管理"
      subtitle="管理持续运行的 Agent 进程"
      icon-bg="linear-gradient(135deg, rgba(10,132,255,0.12), rgba(10,132,255,0.06))"
      icon-color="var(--icon-purple)"
    >
      <template #icon><icon-robot /></template>
      <template #actions>
        <ui-button type="primary" @click="router.push(to.agents.create())">
          <template #icon><icon-plus /></template>
          创建 Agent
        </ui-button>
      </template>
    </PageHeader>

    <!-- 智能工具栏 -->
    <SmartListBar
      :data="filteredAgents"
      :loading="loading"
      v-model:search="searchKeyword"
      v-model:selectedKeys="selectedKeys"
      @refresh="refresh"
      @batch-delete="handleBatchDelete"
      @batch-export="handleBatchExport"
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
        :data="filteredAgents"
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

          <ui-table-column title="名称" data-index="name">
            <template #cell="{ record }">
              <div class="flex items-center gap-3">
                <div class="flex h-9 w-9 flex-shrink-0 items-center justify-center rounded-lg text-sm text-white" :class="{
                  'bg-emerald-500 shadow-sm shadow-emerald-500/20': record.runtime_state === 'running',
                  'bg-slate-400': record.runtime_state === 'stopped' || !['running', 'error'].includes(record.runtime_state),
                  'bg-red-500 shadow-sm shadow-red-500/20': record.runtime_state === 'error'
                }">
                  <icon-robot />
                </div>
                <div class="flex min-w-0 flex-col gap-0.5">
                  <div class="truncate text-[14px] font-medium text-slate-900">{{ record.name }}</div>
                  <code class="font-mono text-xs text-slate-500">{{ record.plugin_key }}:{{ record.action }}</code>
                </div>
              </div>
            </template>
          </ui-table-column>

          <!-- 运行状态 -->
          <ui-table-column title="运行状态">
            <template #cell="{ record }">
              <StatusTag :status="record.runtime_state" />
              <span v-if="record.desired_state !== record.runtime_state" class="text-xs font-mono text-slate-400 ml-1">
                → {{ record.desired_state }}
              </span>
            </template>
          </ui-table-column>

          <!-- 错误信息 -->
          <ui-table-column title="错误信息" data-index="last_error">
            <template #cell="{ record }">
              <span v-if="record.last_error" class="mt-2 text-sm leading-6 text-red-700">{{ record.last_error }}</span>
              <span v-else class="text-sm text-slate-400">—</span>
            </template>
          </ui-table-column>

          <!-- 快速操作 -->
          <ui-table-column title="操作" align="right">
            <template #cell="{ record }">
              <RowActionsMenu
                :item="record"
                :actions="[
                  { key: 'view', label: '查看详情', icon: 'IconEye' },
                  { key: 'start', label: '启动', icon: 'IconPlayArrow', disabled: record.runtime_state === 'running' },
                  { key: 'stop', label: '停止', icon: 'IconStop', disabled: record.runtime_state === 'stopped', danger: true },
                  { key: 'edit', label: '编辑', icon: 'IconEdit' },
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
          <ui-empty description="暂无 Agent">
            <ui-button type="primary" @click="router.push(to.agents.create())">
              创建第一个 Agent
            </ui-button>
          </ui-empty>
        </template>
      </ui-table>
    </ui-card>

    <div class="flex flex-col gap-3 lg:hidden">
      <ui-card
        v-for="record in filteredAgents"
        :key="record.id"
        v-memo="[record.id, record.name, record.plugin_key, record.action, record.runtime_state, record.desired_state]"
        class="rounded-xl border p-5 border-slate-200 bg-white shadow-sm"
      >
        <div class="mb-3 flex items-center gap-3">
          <div class="flex h-9 w-9 flex-shrink-0 items-center justify-center rounded-lg text-sm text-white" :class="{
            'bg-emerald-500 shadow-sm shadow-emerald-500/20': record.runtime_state === 'running',
            'bg-slate-400': record.runtime_state === 'stopped' || !['running', 'error'].includes(record.runtime_state),
            'bg-red-500 shadow-sm shadow-red-500/20': record.runtime_state === 'error'
          }">
            <icon-robot />
          </div>
          <div class="flex min-w-0 flex-1 flex-col gap-0.5">
            <div class="truncate text-sm font-semibold text-slate-900">{{ record.name }}</div>
            <div class="text-xs text-slate-500">
              <code class="font-mono text-xs text-slate-500">#{{ record.id }} · {{ record.plugin_key }}:{{ record.action }}</code>
            </div>
          </div>
          <RowActionsMenu
            :item="record"
            :actions="[
              { key: 'view', label: '查看详情', icon: 'IconEye' },
              { key: 'start', label: '启动', icon: 'IconPlayArrow' },
              { key: 'stop', label: '停止', icon: 'IconStop' },
              { key: 'edit', label: '编辑', icon: 'IconEdit' },
              { key: 'delete-divider', divider: true },
              { key: 'delete', label: '删除', icon: 'IconDelete', danger: true },
            ]"
            @action="handleQuickAction"
          />
        </div>

        <div class="flex flex-col gap-2">
          <div class="flex items-center justify-between gap-2">
            <span class="w-12 flex-shrink-0 text-xs font-medium text-slate-500">状态</span>
            <div class="flex flex-wrap items-center gap-1">
              <StatusTag :status="record.runtime_state" />
              <span v-if="record.desired_state !== record.runtime_state" class="text-xs font-mono text-slate-400 ml-1">
                → {{ record.desired_state }}
              </span>
            </div>
          </div>

          <div v-if="record.last_error" class="flex items-center justify-between gap-2">
            <span class="w-12 flex-shrink-0 text-xs font-medium text-slate-500">错误</span>
            <div class="flex flex-wrap items-center gap-1">
              <span class="text-xs text-red-600 truncate">{{ record.last_error }}</span>
            </div>
          </div>
        </div>
      </ui-card>

      <ui-card
        v-if="!loading && !filteredAgents.length"
        class="col-span-full empty-state-block"
      >
        <ui-empty description="暂无 Agent">
          <ui-button type="primary" @click="router.push(to.agents.create())">
            创建第一个 Agent
          </ui-button>
        </ui-empty>
      </ui-card>
    </div>
  </div>
</template>
