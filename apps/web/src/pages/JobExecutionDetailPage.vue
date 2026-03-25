<script setup lang="ts">
import { computed } from "vue";
import { useRoute, useRouter } from "vue-router";
import { IconThunderbolt, IconLeft, IconCodeBlock, IconRefresh } from "@/lib/icons";

import { useJobExecutions, useJobExecutionStream } from "@/composables/useJobs";
import { PageHeader, StatusTag } from "@/components/index";
import LogTerminal from "@/components/LogTerminal.vue";
import { to } from "@/router/registry";

const route = useRoute();
const router = useRouter();
const executionId = Number(route.params.id);

const { data: executions, loading } = useJobExecutions();
const execution = computed(() => executions.value.find((e) => e.id === executionId));

const stream = useJobExecutionStream(executionId || null);
const isLive = computed(() => stream.status.value === "open" || stream.status.value === "connecting");

function refreshExecution() {
  router.go(0);
}
</script>

<template>
  <div class="page-shell">
    <PageHeader
      :title="execution ? `执行记录 #${execution.id}` : '执行记录详情'"
      icon-bg="linear-gradient(135deg, rgba(10,132,255,0.12), rgba(10,132,255,0.06))"
      icon-color="var(--icon-purple)"
      :back-to="to.jobs.executions()"
      back-label="返回执行列表"
    >
      <template #icon><icon-thunderbolt /></template>
      <template #subtitle>
        <template v-if="execution">
          <code class="inline-flex items-center rounded-md border border-slate-200 bg-slate-100 px-2 py-0.5 text-xs font-mono text-slate-600">{{ execution.definition_name }}</code>
          <span class="mx-1.5 text-slate-400">·</span>
          <code class="inline-flex items-center rounded-full border px-2.5 py-1 text-xs font-mono font-semibold text-slate-700 border-slate-200 bg-white/65">{{ execution.plugin_key }}</code>
          <span class="mx-1.5 text-slate-400">:</span>
          <span class="inline-flex items-center rounded border border-slate-200 bg-slate-100 px-1.5 py-0.5 text-xs font-mono text-slate-600">{{ execution.action }}</span>
        </template>
      </template>
      <template #actions>
        <div v-if="execution" class="flex flex-wrap items-center justify-end gap-2">
          <ui-button @click="refreshExecution">
            <template #icon><icon-refresh /></template>
            刷新
          </ui-button>
          <ui-button type="primary" @click="router.push(to.jobs.list())">
            返回任务管理
          </ui-button>
        </div>
      </template>
    </PageHeader>

    <!-- Loading -->
    <div v-if="loading" class="empty-state-block">
      <ui-spin size="2.25em" />
    </div>

    <!-- Not found -->
    <ui-card v-else-if="!execution" class="empty-state-block">
      <ui-empty description="未找到该执行记录">
        <ui-button type="primary" @click="router.push(to.jobs.executions())">
          返回执行列表
        </ui-button>
      </ui-empty>
    </ui-card>

    <!-- Content -->
    <div v-else class="grid grid-cols-1 gap-6 lg:grid-cols-2">
      <!-- Left: detail panel -->
      <div class="min-w-0 flex flex-col">
        <ui-card class="min-w-0 flex-1">
          <template #title>
            <div class="flex items-center gap-2">
              <icon-info-circle class="h-4.5 w-4.5 text-[var(--accent)]" />
              <span>执行信息</span>
            </div>
          </template>

          <div class="flex flex-col divide-y divide-slate-100">
            <div class="flex items-center justify-between gap-4 py-4 first:pt-0 max-md:flex-col max-md:items-start max-md:gap-2">
              <span class="text-xs font-semibold tracking-wider text-slate-500">执行状态</span>
              <StatusTag :status="execution.status" />
            </div>

            <div class="flex items-center justify-between gap-4 py-4 max-md:flex-col max-md:items-start max-md:gap-2">
              <span class="text-xs font-semibold tracking-wider text-slate-500">任务名称</span>
              <span class="text-sm font-medium text-slate-900">{{ execution.definition_name }}</span>
            </div>

            <div class="flex items-center justify-between gap-4 py-4 max-md:flex-col max-md:items-start max-md:gap-2">
              <span class="text-xs font-semibold tracking-wider text-slate-500">插件</span>
              <code class="inline-flex items-center rounded-full border px-2.5 py-1 text-xs font-mono font-medium text-slate-700 border-slate-200 bg-slate-50">{{ execution.plugin_key }}</code>
            </div>

            <div class="flex items-center justify-between gap-4 py-4 max-md:flex-col max-md:items-start max-md:gap-2">
              <span class="text-xs font-semibold tracking-wider text-slate-500">动作</span>
              <span class="inline-flex items-center rounded-full border border-slate-200 bg-slate-50 px-2.5 py-1 text-xs font-medium text-slate-700">{{ execution.action }}</span>
            </div>

            <div class="flex items-center justify-between gap-4 py-4 max-md:flex-col max-md:items-start max-md:gap-2">
              <span class="text-xs font-semibold tracking-wider text-slate-500">执行节点</span>
              <code v-if="execution.worker_id" class="inline-flex items-center rounded-full border px-2.5 py-1 text-xs font-mono font-medium text-slate-700 border-slate-200 bg-slate-50">
                {{ execution.worker_id }}
              </code>
              <span v-else class="text-sm text-slate-400">未分配</span>
            </div>

            <div v-if="execution.input" class="flex flex-col gap-3 py-4 last:pb-0">
              <span class="text-xs font-semibold tracking-wider text-slate-500">输入参数</span>
              <div class="overflow-auto rounded-xl border border-slate-200 bg-slate-50 shadow-sm p-4">
                <pre class="m-0 whitespace-pre-wrap break-all text-xs leading-relaxed text-slate-600 font-mono">{{ JSON.stringify(execution.input, null, 2) }}</pre>
              </div>
            </div>
          </div>
        </ui-card>
      </div>

      <!-- Right: log terminal -->
      <div class="min-w-0 flex flex-col">
        <ui-card class="min-w-0 flex-1 flex flex-col [&>.ui-card-body]:flex-1 [&>.ui-card-body]:flex [&>.ui-card-body]:flex-col [&>.ui-card-body]:min-h-0 [&>.ui-card-body]:p-5">
          <template #title>
            <div class="flex items-center gap-2">
              <icon-code-block class="h-4.5 w-4.5 text-[var(--accent)]" />
              <span>实时日志</span>
              <span class="text-slate-300 mx-1">·</span>
              <div class="flex items-center gap-1.5 text-xs font-medium" :class="isLive ? 'text-green-600' : 'text-slate-400'">
                <span class="h-1.5 w-1.5 rounded-full" :class="isLive ? 'bg-green-500 animate-pulse' : 'bg-slate-300'" />
                {{ isLive ? "实时连接" : "离线缓冲" }}
              </div>
            </div>
          </template>

          <LogTerminal
            class="flex-1 min-h-[500px]"
            height="100%"
            :logs="stream.lines.value"
            :is-live="isLive"
            :show-header="false"
            empty-label="等待事件流…"
          />
        </ui-card>
      </div>
    </div>
  </div>
</template>
