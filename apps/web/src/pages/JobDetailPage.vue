<script setup lang="ts">
import { computed, ref, onMounted } from "vue";
import { useRoute, useRouter } from "vue-router";
import {
  IconSchedule, IconPlayArrow, IconEdit, IconDelete,
  IconClockCircle, IconCopy, IconCode, IconHistory,
  IconThunderbolt, IconRight, IconRefresh
} from "@/lib/icons";

import { useJobDefinitions, useEnqueueJobExecution, useDeleteJobDefinition, useJobExecutions } from "@/composables/useJobs";
import { useMessage, useConfirm, useErrorHandler } from "@/composables";
import { PageHeader } from "@/components/index";
import { to } from "@/router/registry";

const route = useRoute();
const router = useRouter();
const message = useMessage();
const confirm = useConfirm();
const { withErrorHandler } = useErrorHandler();

const jobId = Number(route.params.id);
const { data: definitions, loading: defLoading } = useJobDefinitions();
const { data: executions, loading: execLoading, refresh: refreshExecutions } = useJobExecutions(jobId);
const enqueue = useEnqueueJobExecution();
const deleteJobOp = useDeleteJobDefinition();

const job = computed(() => definitions.value.find((j) => j.id === jobId));
const recentExecutions = computed(() => executions.value.slice(0, 5));

// 复制到剪贴板
async function copyToClipboard(text: string) {
  try {
    await navigator.clipboard.writeText(text);
    message.success("已复制到剪贴板");
  } catch {
    message.error("复制失败");
  }
}

// 立即执行任务
async function handleEnqueue() {
  if (!job.value) return;

  await withErrorHandler(
    async () => {
      await enqueue.execute(job.value!.id);
      message.success("已加入执行队列");
      // 延迟刷新以等待执行记录生成
      setTimeout(() => {
        refreshExecutions();
      }, 1000);
    },
    { action: "执行任务", showSuccess: false }
  );
}

// 删除任务
async function deleteJob() {
  if (!job.value) return;

  const confirmed = await confirm.confirmDelete(job.value.name);
  if (!confirmed) return;

  await withErrorHandler(
    async () => {
      await deleteJobOp.execute(job.value!.id);
      message.success(`已删除任务: ${job.value!.name}`);
      router.push(to.jobs.list());
    },
    { action: "删除", showSuccess: false }
  );
}

// 格式化日期
function formatDate(dateStr?: string) {
  if (!dateStr) return "-";
  return new Date(dateStr).toLocaleString("zh-CN", {
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit"
  });
}

onMounted(() => {
  refreshExecutions();
});
</script>

<template>
  <div class="page-shell">
    <PageHeader
      :title="job ? job.name : '任务详情'"
      icon-bg="linear-gradient(135deg, rgba(202,138,4,0.12), rgba(234,179,8,0.12))"
      icon-color="var(--icon-yellow)"
      :back-to="to.jobs.list()"
      back-label="返回任务列表"
    >
      <template #icon><icon-schedule /></template>
      <template #subtitle>
        <div class="flex items-center gap-2">
          <code v-if="job" class="inline-flex items-center rounded-md border border-slate-200 bg-slate-100 px-2 py-0.5 text-xs font-mono text-slate-600">{{ job.key }}</code>
          <ui-tag v-if="job" :type="job.enabled ? 'success' : 'default'">{{ job.enabled ? '已启用' : '已禁用' }}</ui-tag>
        </div>
      </template>
      <template #actions>
        <div v-if="job" class="flex flex-wrap items-center justify-end gap-2">
          <ui-button
            type="outline"
            :loading="enqueue.loading.value"
            @click="handleEnqueue"
            :disabled="!job.enabled"
          >
            <template #icon><icon-play-arrow /></template>
            执行
          </ui-button>
          <ui-button type="primary" @click="router.push(to.jobs.edit(job.id))">
            <template #icon><icon-edit /></template>
            编辑
          </ui-button>
          <ui-button type="danger" @click="deleteJob">
            <template #icon><icon-delete /></template>
            删除
          </ui-button>
        </div>
      </template>
    </PageHeader>

    <!-- 加载中 -->
    <div v-if="defLoading" class="empty-state-block"><ui-spin size="2.25em" /></div>
    
    <!-- 任务未找到 -->
    <ui-card v-else-if="!job" class="empty-state-block">
      <ui-empty description="未找到该任务">
        <ui-button type="primary" @click="router.push(to.jobs.list())">
          返回任务列表
        </ui-button>
      </ui-empty>
    </ui-card>

    <!-- 任务详情 -->
    <div v-else class="grid grid-cols-1 gap-6 xl:grid-cols-[1.5fr_1fr] pb-10">
      
      <!-- 左侧主内容区 -->
      <div class="flex flex-col gap-6">
        <!-- 基本信息卡片 -->
        <ui-card class="min-w-0">
          <template #title>
            <div class="flex items-center gap-2">
              <icon-schedule class="h-4 w-4 text-[var(--accent)]" />
              <span>基础配置</span>
            </div>
          </template>

          <div class="flex flex-col divide-y divide-slate-100">
            <div class="flex items-center justify-between gap-4 py-4 max-md:flex-col max-md:items-start max-md:gap-2">
              <span class="text-xs font-semibold tracking-wider text-slate-500 min-w-24">任务名称</span>
              <span class="text-sm font-medium text-slate-900">{{ job.name }}</span>
            </div>

            <div class="flex items-center justify-between gap-4 py-4 max-md:flex-col max-md:items-start max-md:gap-2">
              <span class="text-xs font-semibold tracking-wider text-slate-500 min-w-24">插件/动作</span>
              <div class="flex items-center gap-2">
                <code class="inline-flex items-center rounded-md border border-slate-200 bg-slate-50 px-2 py-0.5 text-xs font-mono text-slate-700">{{ job.plugin_key }}</code>
                <span class="text-slate-400">/</span>
                <span class="inline-flex items-center rounded-md border border-indigo-100 bg-indigo-50 px-2 py-0.5 text-xs font-medium text-indigo-700">{{ job.action }}</span>
              </div>
            </div>

            <div class="flex items-center justify-between gap-4 py-4 max-md:flex-col max-md:items-start max-md:gap-2">
              <span class="text-xs font-semibold tracking-wider text-slate-500 min-w-24">调度方式</span>
              <div>
                <span v-if="job.schedule?.cron_expression" class="inline-flex items-center gap-1.5 rounded-md border border-emerald-200 bg-emerald-50 px-2.5 py-1 text-xs font-medium text-emerald-700">
                  <icon-clock-circle class="w-3.5 h-3.5" />
                  定时任务
                </span>
                <span v-else class="inline-flex items-center gap-1.5 rounded-md border border-slate-200 bg-slate-100 px-2.5 py-1 text-xs font-medium text-slate-600">
                  <icon-thunderbolt class="w-3.5 h-3.5" />
                  手动触发
                </span>
              </div>
            </div>

            <template v-if="job.schedule?.cron_expression">
              <div class="flex items-center justify-between gap-4 py-4 max-md:flex-col max-md:items-start max-md:gap-2">
                <span class="text-xs font-semibold tracking-wider text-slate-500 min-w-24">Cron 表达式</span>
                <code class="font-mono bg-slate-50 px-2 py-1 rounded border border-slate-200 text-sm text-slate-700">{{ job.schedule.cron_expression }}</code>
              </div>
              <div class="flex items-center justify-between gap-4 py-4 max-md:flex-col max-md:items-start max-md:gap-2">
                <span class="text-xs font-semibold tracking-wider text-slate-500 min-w-24">时区</span>
                <span class="text-sm font-medium text-slate-900">{{ job.schedule.timezone || 'UTC' }}</span>
              </div>
            </template>
          </div>
        </ui-card>

        <!-- 输入参数卡片 -->
        <ui-card class="min-w-0 flex-1 flex flex-col">
          <template #title>
            <div class="flex items-center gap-2">
              <icon-code class="h-4 w-4 text-[var(--accent)]" />
              <span>输入参数</span>
            </div>
          </template>

          <div v-if="!job.input || Object.keys(job.input).length === 0" class="flex-1 flex items-center justify-center py-8 text-slate-400 text-sm">
            该任务没有配置输入参数
          </div>
          <div v-else class="overflow-auto rounded-lg border border-slate-200 bg-slate-50 shadow-sm p-4 flex-1">
            <pre class="m-0 whitespace-pre-wrap break-all text-xs leading-relaxed text-slate-600 font-mono">{{ JSON.stringify(job.input, null, 2) }}</pre>
          </div>
        </ui-card>
      </div>

      <!-- 右侧辅助信息区 -->
      <div class="flex flex-col gap-6">
        
        <!-- 标识符信息 -->
        <ui-card class="min-w-0">
          <template #title>
            <div class="flex items-center gap-2">
              <icon-info-circle class="h-4 w-4 text-[var(--accent)]" />
              <span>标识信息</span>
            </div>
          </template>
          <div class="flex flex-col gap-3">
            <div class="flex flex-col gap-1">
              <span class="text-xs text-slate-500">内部 ID</span>
              <span class="font-mono text-sm">{{ job.id }}</span>
            </div>
            <div class="flex flex-col gap-1">
              <span class="text-xs text-slate-500">Key 标识</span>
              <div class="flex items-center justify-between gap-2 bg-slate-50 border border-slate-200 rounded p-2">
                <code class="text-sm font-mono text-slate-700 truncate">{{ job.key }}</code>
                <button 
                  class="text-slate-400 hover:text-[var(--accent)] transition-colors p-1"
                  title="复制"
                  @click="copyToClipboard(job.key)"
                >
                  <icon-copy class="w-4 h-4" />
                </button>
              </div>
            </div>
          </div>
        </ui-card>

        <!-- 近期执行记录 -->
        <ui-card class="min-w-0 flex-1 flex flex-col">
          <template #title>
            <div class="flex items-center justify-between w-full">
              <div class="flex items-center gap-2">
                <icon-history class="h-4 w-4 text-[var(--accent)]" />
                <span>近期执行记录</span>
              </div>
              <button 
                class="text-slate-400 hover:text-[var(--accent)] transition-colors p-1 rounded-md hover:bg-slate-50"
                title="刷新记录"
                @click="refreshExecutions()"
                :class="{ 'animate-spin': execLoading }"
              >
                <icon-refresh class="w-4 h-4" />
              </button>
            </div>
          </template>

          <div v-if="execLoading && executions.length === 0" class="py-8 flex justify-center">
            <ui-spin size="1.5em" />
          </div>
          
          <div v-else-if="recentExecutions.length === 0" class="py-8 flex-1 flex flex-col items-center justify-center gap-3">
            <span class="text-sm text-slate-400">暂无执行记录</span>
            <ui-button size="small" type="outline" @click="handleEnqueue" :loading="enqueue.loading.value" :disabled="!job.enabled">
              <template #icon><icon-play-arrow /></template>
              立即执行
            </ui-button>
          </div>

          <div v-else class="flex flex-col gap-3">
            <div 
              v-for="exec in recentExecutions" 
              :key="exec.id"
              class="flex items-center justify-between p-3 rounded-lg border border-slate-100 bg-slate-50 hover:bg-slate-100 transition-colors cursor-pointer"
              @click="router.push(to.jobs.executionDetail(exec.id))"
            >
              <div class="flex items-center gap-3">
                <ui-tag 
                  :type="exec.status === 'completed' ? 'success' : exec.status === 'failed' ? 'danger' : exec.status === 'running' ? 'warning' : 'default'"
                  size="small"
                >
                  {{ exec.status }}
                </ui-tag>
                <span class="text-xs text-slate-500 font-mono">#{{ exec.id }}</span>
              </div>
              <div class="flex items-center gap-3">
                <span class="text-xs text-slate-500">{{ formatDate(exec.started_at || exec.created_at) }}</span>
                <icon-right class="w-3.5 h-3.5 text-slate-400" />
              </div>
            </div>
          </div>

          <template #extra>
            <div class="mt-4 pt-3 border-t border-slate-100 text-center">
              <ui-button type="text" @click="router.push(to.jobs.executions())" class="text-xs">
                查看全部记录
                <icon-right class="ml-1 w-3 h-3" />
              </ui-button>
            </div>
          </template>
        </ui-card>

      </div>
    </div>
  </div>
</template>
