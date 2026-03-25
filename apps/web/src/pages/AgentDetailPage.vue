<script setup lang="ts">
import { computed, ref } from "vue";
import { useRoute, useRouter } from "vue-router";
import { useAgent, useAgentStatus, useStartAgent, useStopAgent, useAgentStream } from "@/composables/useAgents";
import { useMessage } from "@/composables";
import { copyToClipboard, formatDateTime } from "@/shared/utils";
import { getStatusLabel } from "@/shared/utils/status";
import { PageHeader } from "@/components/index";
import LogTerminal from "@/components/LogTerminal.vue";
import { to } from "@/router/registry";

const route = useRoute();
const router = useRouter();
const agentId = Number(route.params.id);
const message = useMessage();

const { data: agent, loading: loadingAgent, refresh: refreshAgent } = useAgent(agentId);
const { data: agentStatus, refresh: refreshAgentStatus } = useAgentStatus(agentId);
const startAgent = useStartAgent();
const stopAgent = useStopAgent();
const {
  lines,
  connected,
  runtimeState: streamRuntimeState,
  desiredState: streamDesiredState,
  lastError: streamLastError,
  statusSnapshot: streamStatus,
  statusLastHeartbeatAt: streamHeartbeatAt,
  updatedAt: streamUpdatedAt,
} = useAgentStream(agentId || null);

const activeTab = ref("status");

const displayState = computed(() =>
  streamRuntimeState.value ??
  agentStatus.value?.runtime_state ??
  agent.value?.runtime_state
);
const displayDesired = computed(() =>
  streamDesiredState.value ??
  agentStatus.value?.desired_state ??
  agent.value?.desired_state
);
const displayError = computed(() => {
  if (streamStatus.value) {
    return streamLastError.value;
  }
  return agentStatus.value?.last_error ?? agent.value?.last_error ?? "";
});
const displayHeartbeatAt = computed(() =>
  streamHeartbeatAt.value ??
  agentStatus.value?.last_heartbeat_at ??
  null
);
const displayStatusUpdatedAt = computed(() =>
  streamUpdatedAt.value ??
  agentStatus.value?.updated_at ??
  null
);

const inputEntries = computed(() => Object.entries(agent.value?.input ?? {}));
const inputCount = computed(() => inputEntries.value.length);
const logCount = computed(() => lines.value.length);

const stateLabel = computed(() => getStatusLabel(displayState.value ?? ""));
const desiredLabel = computed(() => getStatusLabel(displayDesired.value ?? ""));
const streamLabel = computed(() => connected.value ? "已连接" : "未连接");
const statusSyncLabel = computed(() => {
  if (!displayState.value || !displayDesired.value) {
    return "等待状态同步";
  }
  return displayState.value !== displayDesired.value ? "状态同步中" : "状态一致";
});

const overviewDescription = computed(() => {
  if (!agent.value) return "";
  return `该 Agent 持续执行 ${agent.value.plugin_key}.${agent.value.action}，当前状态为 ${stateLabel.value || "未知"}，${connected.value ? "日志流已连接" : "日志流未连接"}。`;
});

const canStart = computed(() =>
  displayDesired.value !== "running" &&
  !["running", "starting"].includes(displayState.value ?? "")
);
const canStop = computed(() =>
  displayDesired.value !== "stopped" &&
  !["stopped", "stopping"].includes(displayState.value ?? "")
);
const pluginDetailPath = computed(() => (agent.value ? to.plugins.detail(agent.value.plugin_key) : ""));

function formatTimestamp(value?: string | null, format: "full" | "relative" = "full"): string {
  return value ? formatDateTime(value, format) : "—";
}

function describeInputValue(value: unknown): string {
  if (Array.isArray(value)) return `数组 · ${value.length} 项`;
  if (value && typeof value === "object") return "对象";
  if (typeof value === "boolean") return "布尔值";
  if (typeof value === "number") return "数值";
  return "文本";
}

function stringifyInputValue(value: unknown): string {
  if (value === null || value === undefined) return "—";
  if (typeof value === "string") return value;
  if (typeof value === "number" || typeof value === "boolean") return String(value);
  return JSON.stringify(value, null, 2);
}

function isBlockValue(value: unknown): boolean {
  if (Array.isArray(value)) return true;
  if (value && typeof value === "object") return true;
  return typeof value === "string" && (value.includes("\n") || value.length > 96);
}

async function handleCopy(label: string, value: string) {
  const copied = await copyToClipboard(value);
  if (copied) {
    message.success(`${label}已复制`);
    return;
  }
  message.error("复制失败");
}

async function handleStart() {
  if (!agent.value) return;
  try {
    await startAgent.execute(agent.value.id);
    await Promise.allSettled([refreshAgent(), refreshAgentStatus()]);
    message.success("启动指令已发送");
  } catch (e) {
    message.error(e instanceof Error ? e.message : "操作失败");
  }
}

async function handleStop() {
  if (!agent.value) return;
  try {
    await stopAgent.execute(agent.value.id);
    await Promise.allSettled([refreshAgent(), refreshAgentStatus()]);
    message.success("停止指令已发送");
  } catch (e) {
    message.error(e instanceof Error ? e.message : "操作失败");
  }
}
</script>

<template>
  <div class="page-shell">
    <PageHeader
      :title="agent ? agent.name : 'Agent 详情'"
      icon-bg="linear-gradient(135deg, rgba(10,132,255,0.12), rgba(10,132,255,0.06))"
      icon-color="var(--icon-purple)"
      :back-to="to.agents.list()"
      back-label="返回 Agent 列表"
    >
      <template #icon><icon-robot /></template>
      <template #subtitle>
        <template v-if="agent">
          <code class="inline-flex items-center rounded-md border border-slate-200 bg-slate-100 px-2 py-0.5 text-xs font-mono text-slate-600">{{ agent.plugin_key }}</code>
          <span class="mx-1.5 text-slate-400">·</span>
          <span class="inline-flex items-center rounded-full border border-slate-200 bg-slate-50 px-2.5 py-1 text-xs font-medium text-slate-600">{{ agent.action }}</span>
        </template>
      </template>
      <template #actions>
        <div v-if="agent" class="flex flex-wrap items-center justify-end gap-2">
          <div class="flex flex-wrap items-center gap-2">
            <StatusTag :status="displayState" />
            <span
              class="inline-flex items-center gap-2 rounded-full border px-3 py-1 text-xs font-medium transition-colors"
              :class="connected
                ? 'border-emerald-200 bg-emerald-50 text-emerald-700'
                : 'border-slate-200 bg-white/60 text-slate-500'"
            >
              <span
                class="h-2 w-2 rounded-full"
                :class="connected ? 'bg-emerald-500 animate-pulse' : 'bg-slate-300'"
              />
              {{ connected ? "实时连接" : "未连接" }}
            </span>
          </div>
          <ui-button
            type="primary"
            :loading="startAgent.loading.value"
            :disabled="!canStart || stopAgent.loading.value"
            @click="handleStart"
          >
            <template #icon><icon-play-arrow /></template>
            启动 Agent
          </ui-button>
          <ui-button
            status="danger"
            :loading="stopAgent.loading.value"
            :disabled="!canStop || startAgent.loading.value"
            @click="handleStop"
          >
            <template #icon><icon-pause /></template>
            停止 Agent
          </ui-button>
        </div>
      </template>
    </PageHeader>

    <div v-if="loadingAgent" class="flex flex-col items-center justify-center gap-4 py-20 text-center">
      <ui-spin size="2.25em" />
    </div>
    <div v-else-if="!agent" class="flex flex-col items-center justify-center gap-4 py-20 text-center">
      <icon-robot class="h-12 w-12 text-slate-400" />
      <p class="text-sm text-slate-500">未找到该 Agent。</p>
      <ui-button @click="router.push(to.agents.list())">返回 Agent 列表</ui-button>
    </div>

    <div v-else class="flex flex-col gap-6">
      <!-- Overview card -->
      <section class="flex flex-col gap-6 rounded-xl border border-slate-200 bg-white p-6 shadow-sm lg:flex-row">
        <div class="flex flex-1 flex-col gap-3">
          <span class="text-xs font-semibold uppercase tracking-wider text-[var(--accent)]">Agent Runtime</span>
          <h2 class="m-0 text-2xl font-bold text-slate-900">{{ agent.name }}</h2>
          <p class="text-sm text-slate-500">{{ overviewDescription }}</p>

          <div class="flex flex-wrap items-center gap-2">
            <StatusTag :status="displayState" />

            <div class="inline-flex items-center gap-2 rounded-full border border-slate-200 bg-slate-50 px-3 py-1.5 text-xs text-slate-500">
              <span>插件</span>
              <code class="text-xs font-semibold text-slate-900 font-mono">{{ agent.plugin_key }}</code>
            </div>

            <div class="inline-flex items-center gap-2 rounded-full border border-slate-200 bg-slate-50 px-3 py-1.5 text-xs text-slate-500">
              <span>动作</span>
              <code class="text-xs font-semibold text-slate-900 font-mono">{{ agent.action }}</code>
            </div>

            <div class="inline-flex items-center gap-2 rounded-full border border-slate-200 bg-slate-50 px-3 py-1.5 text-xs text-slate-500">
              <span>日志连接</span>
              <span class="font-medium text-slate-900">{{ streamLabel }}</span>
            </div>
          </div>
        </div>

        <div class="grid grid-cols-2 gap-3 lg:min-w-[240px]">
          <article class="flex flex-col gap-1 rounded-xl border border-slate-200 bg-slate-50 p-3 shadow-sm">
            <span class="text-xs text-slate-500">期望状态</span>
            <span class="truncate text-sm font-semibold text-slate-900">{{ desiredLabel || "—" }}</span>
            <span class="text-xs text-slate-400">{{ statusSyncLabel }}</span>
          </article>

          <article class="flex flex-col gap-1 rounded-xl border border-slate-200 bg-slate-50 p-3 shadow-sm">
            <span class="text-xs text-slate-500">输入参数</span>
            <span class="text-2xl font-bold text-slate-900">{{ inputCount }}</span>
            <span class="text-xs text-slate-400">{{ inputCount ? "已配置顶层字段" : "未配置输入参数" }}</span>
          </article>

          <article class="flex flex-col gap-1 rounded-xl border border-slate-200 bg-slate-50 p-3 shadow-sm">
            <span class="text-xs text-slate-500">已收日志</span>
            <span class="text-2xl font-bold text-slate-900">{{ logCount }}</span>
            <span class="text-xs text-slate-400">{{ connected ? "实时追加中" : "等待连接" }}</span>
          </article>

          <article class="flex flex-col gap-1 rounded-xl border border-slate-200 bg-slate-50 p-3 shadow-sm">
            <span class="text-xs text-slate-500">最近心跳</span>
            <span class="truncate text-sm font-semibold text-slate-900">{{ formatTimestamp(displayHeartbeatAt, "relative") }}</span>
            <span class="text-xs text-slate-400">{{ formatTimestamp(displayHeartbeatAt) }}</span>
          </article>
        </div>
      </section>

      <!-- Tabs -->
      <ui-tabs v-model:active-key="activeTab" :destroy-on-hide="false">
        <ui-tab-pane key="status" title="运行信息">
          <div class="grid grid-cols-1 gap-6 lg:grid-cols-[minmax(0,_1.15fr)_minmax(16em,_0.85fr)]">
            <!-- Left column -->
            <div class="flex flex-col gap-6">
              <!-- Runtime status card -->
              <ui-card>
                <template #title>
                  <div class="flex items-center gap-2">
                    <icon-dashboard class="h-4 w-4 text-[var(--accent)]" />
                    运行状态
                  </div>
                </template>
                <div class="flex flex-col divide-y divide-slate-100">
                  <div class="flex items-center justify-between gap-4 py-4 max-md:flex-col max-md:items-start max-md:gap-2">
                    <span class="text-xs font-semibold tracking-wider text-slate-500 min-w-24">Agent ID</span>
                    <span class="text-sm font-mono font-medium text-slate-900">#{{ agent.id }}</span>
                  </div>

                  <div class="flex items-center justify-between gap-4 py-4 max-md:flex-col max-md:items-start max-md:gap-2">
                    <span class="text-xs font-semibold tracking-wider text-slate-500 min-w-24">实时状态</span>
                    <StatusTag :status="displayState" />
                  </div>

                  <div class="flex items-center justify-between gap-4 py-4 max-md:flex-col max-md:items-start max-md:gap-2">
                    <span class="text-xs font-semibold tracking-wider text-slate-500 min-w-24">期望状态</span>
                    <StatusTag :status="displayDesired" />
                  </div>

                  <div class="flex items-center justify-between gap-4 py-4 max-md:flex-col max-md:items-start max-md:gap-2">
                    <span class="text-xs font-semibold tracking-wider text-slate-500 min-w-24">日志连接</span>
                    <div class="flex items-center gap-2">
                      <span
                        class="h-2 w-2 rounded-full"
                        :class="connected ? 'bg-emerald-500 animate-pulse' : 'bg-slate-300'"
                      />
                      <span class="text-sm font-medium text-slate-900">{{ streamLabel }}</span>
                    </div>
                  </div>

                  <div class="flex items-center justify-between gap-4 py-4 max-md:flex-col max-md:items-start max-md:gap-2">
                    <span class="text-xs font-semibold tracking-wider text-slate-500 min-w-24">插件</span>
                    <code class="inline-flex items-center rounded-full border border-slate-200 bg-slate-50 px-2.5 py-1 text-xs font-mono font-semibold text-slate-700">{{ agent.plugin_key }}</code>
                  </div>

                  <div class="flex items-center justify-between gap-4 py-4 max-md:flex-col max-md:items-start max-md:gap-2">
                    <span class="text-xs font-semibold tracking-wider text-slate-500 min-w-24">动作</span>
                    <span class="inline-flex items-center rounded-full border border-slate-200 bg-slate-50 px-2.5 py-1 text-xs font-semibold text-slate-700">{{ agent.action }}</span>
                  </div>

                  <div class="flex items-center justify-between gap-4 py-4 max-md:flex-col max-md:items-start max-md:gap-2">
                    <span class="text-xs font-semibold tracking-wider text-slate-500 min-w-24">最近心跳</span>
                    <div class="flex flex-col items-end gap-1 text-right max-md:items-start max-md:text-left">
                      <span class="text-sm font-medium text-slate-900">{{ formatTimestamp(displayHeartbeatAt, "relative") }}</span>
                      <span class="text-xs text-slate-400">{{ formatTimestamp(displayHeartbeatAt) }}</span>
                    </div>
                  </div>

                  <div class="flex items-center justify-between gap-4 py-4 max-md:flex-col max-md:items-start max-md:gap-2">
                    <span class="text-xs font-semibold tracking-wider text-slate-500 min-w-24">状态更新</span>
                    <div class="flex flex-col items-end gap-1 text-right max-md:items-start max-md:text-left">
                      <span class="text-sm font-medium text-slate-900">{{ formatTimestamp(displayStatusUpdatedAt, "relative") }}</span>
                      <span class="text-xs text-slate-400">{{ formatTimestamp(displayStatusUpdatedAt) }}</span>
                    </div>
                  </div>
                </div>

                <div v-if="displayError" class="mt-5 flex items-start gap-3 rounded-xl border border-red-200 bg-red-50 p-4">
                  <icon-close-circle class="h-5 w-5 flex-shrink-0 text-red-500 mt-0.5" />
                  <div>
                    <p class="text-sm font-semibold text-red-700">错误信息</p>
                    <p class="mt-1 text-sm leading-6 text-red-600">{{ displayError }}</p>
                  </div>
                </div>
              </ui-card>

              <!-- Input params card -->
              <ui-card>
                <template #title>
                  <div class="flex items-center gap-2">
                    <icon-code-block class="h-4 w-4 text-purple-600" />
                    输入参数
                  </div>
                </template>

                <div v-if="inputEntries.length" class="flex flex-col gap-4">
                  <article
                    v-for="[key, value] in inputEntries"
                    :key="key"
                    class="rounded-xl border border-slate-200 bg-slate-50 p-4"
                  >
                    <div class="mb-3 flex flex-wrap items-center justify-between gap-3 max-md:flex-col max-md:items-start">
                      <div class="flex flex-wrap items-center gap-2">
                        <code class="inline-flex items-center rounded-md border border-slate-200 bg-slate-100 px-2 py-0.5 text-xs font-mono text-slate-600 whitespace-nowrap">{{ key }}</code>
                        <span class="inline-flex items-center rounded-full border border-slate-200 bg-white/70 px-2.5 py-1 text-xs font-medium text-slate-500">{{ describeInputValue(value) }}</span>
                      </div>
                      <button
                        type="button"
                        class="inline-flex items-center gap-2 rounded-lg border border-slate-200 bg-white px-3 py-1.5 text-xs font-medium text-slate-600 transition-all hover:border-slate-300 hover:bg-slate-50 hover:text-slate-900"
                        @click="handleCopy(`${key} 字段`, stringifyInputValue(value))"
                      >
                        <icon-copy class="h-3.5 w-3.5" />
                        复制
                      </button>
                    </div>

                    <pre v-if="isBlockValue(value)" class="overflow-auto rounded-xl border border-slate-200 bg-slate-50 shadow-sm p-4 text-xs leading-relaxed text-slate-600 font-mono whitespace-pre-wrap break-all">{{ stringifyInputValue(value) }}</pre>
                    <div v-else class="rounded-lg border border-slate-200 bg-white px-4 py-3">
                      <span class="text-sm font-mono font-medium text-slate-900">{{ stringifyInputValue(value) }}</span>
                    </div>
                  </article>
                </div>

                <p v-else class="text-sm leading-6 text-slate-400 italic">该 Agent 未配置输入参数。</p>
              </ui-card>
            </div>

            <!-- Right column -->
            <div class="flex flex-col gap-6">
              <!-- Connection status card -->
              <ui-card>
                <template #title>
                  <div class="flex items-center gap-2">
                    <icon-info-circle class="h-4 w-4 text-sky-600" />
                    连接状态
                  </div>
                </template>
                <div class="flex flex-col divide-y divide-slate-100">
                  <div class="flex items-center justify-between gap-4 py-4 max-md:flex-col max-md:items-start max-md:gap-2">
                    <span class="text-xs font-semibold tracking-wider text-slate-500">日志流</span>
                    <div class="flex items-center gap-2">
                      <span
                        class="h-2 w-2 rounded-full"
                        :class="connected ? 'bg-emerald-500 animate-pulse' : 'bg-slate-300'"
                      />
                      <span class="text-sm font-medium text-slate-900">{{ streamLabel }}</span>
                    </div>
                  </div>

                  <div class="flex items-center justify-between gap-4 py-4 max-md:flex-col max-md:items-start max-md:gap-2">
                    <span class="text-xs font-semibold tracking-wider text-slate-500">状态同步</span>
                    <span class="text-sm font-medium text-slate-900">{{ statusSyncLabel }}</span>
                  </div>

                  <div class="flex items-center justify-between gap-4 py-4 max-md:flex-col max-md:items-start max-md:gap-2">
                    <span class="text-xs font-semibold tracking-wider text-slate-500">最近状态</span>
                    <span class="text-sm font-medium text-slate-900">{{ stateLabel || "—" }}</span>
                  </div>

                  <div class="flex items-center justify-between gap-4 py-4 max-md:flex-col max-md:items-start max-md:gap-2">
                    <span class="text-xs font-semibold tracking-wider text-slate-500">日志条数</span>
                    <span class="text-sm font-medium text-slate-900">{{ logCount }}</span>
                  </div>
                </div>
              </ui-card>

              <!-- Quick actions card -->
              <ui-card>
                <template #title>
                  <div class="flex items-center gap-2">
                    <icon-thunderbolt class="h-4 w-4 text-slate-500" />
                    快捷操作
                  </div>
                </template>
                <p class="text-sm leading-6 text-slate-500">
                  快速跳转到关联插件，或复制 Agent 的关键标识，便于排查和联调。
                </p>
                <div class="mt-4 flex flex-wrap gap-2">
                  <button
                    type="button"
                    class="inline-flex items-center gap-2 rounded-lg border border-slate-200 bg-white px-3.5 py-2 text-sm font-medium text-slate-700 transition-all hover:border-slate-300 hover:bg-slate-50 hover:-translate-y-px"
                    @click="router.push(pluginDetailPath)"
                  >
                    <icon-apps class="h-4 w-4" />
                    查看插件
                  </button>
                  <button
                    type="button"
                    class="inline-flex items-center gap-2 rounded-lg border border-slate-200 bg-white px-3.5 py-2 text-sm font-medium text-slate-700 transition-all hover:border-slate-300 hover:bg-slate-50 hover:-translate-y-px"
                    @click="handleCopy('Agent ID', String(agent.id))"
                  >
                    <icon-copy class="h-4 w-4" />
                    复制 Agent ID
                  </button>
                  <button
                    type="button"
                    class="inline-flex items-center gap-2 rounded-lg border border-slate-200 bg-white px-3.5 py-2 text-sm font-medium text-slate-700 transition-all hover:border-slate-300 hover:bg-slate-50 hover:-translate-y-px"
                    @click="handleCopy('插件动作', `${agent.plugin_key}:${agent.action}`)"
                  >
                    <icon-copy class="h-4 w-4" />
                    复制插件动作
                  </button>
                </div>
              </ui-card>
            </div>
          </div>
        </ui-tab-pane>

        <ui-tab-pane key="logs" title="实时日志">
          <div class="flex flex-col gap-5">
            <section class="flex flex-col gap-4 rounded-xl border border-slate-200 bg-slate-50 p-5 shadow-sm lg:flex-row lg:items-center lg:justify-between">
              <div class="flex flex-col gap-2">
                <span class="text-xs font-semibold uppercase tracking-wider text-[var(--accent)]">Agent Logs</span>
                <h3 class="text-xl font-semibold text-slate-900 tracking-[-0.02em]">实时日志流</h3>
                <p class="text-xs leading-relaxed text-slate-500">
                  当前展示最近 {{ logCount }} 条日志。保持页面开启时，日志会随 SSE 事件实时追加。
                </p>
              </div>
              <div class="flex flex-wrap gap-2">
                <div class="inline-flex items-center gap-2 rounded-full border border-slate-200 bg-white px-3 py-1.5 text-xs text-slate-500">
                  <span>连接状态</span>
                  <span class="font-medium text-slate-900">{{ streamLabel }}</span>
                </div>
                <div class="inline-flex items-center gap-2 rounded-full border border-slate-200 bg-white px-3 py-1.5 text-xs text-slate-500">
                  <span>最近心跳</span>
                  <span class="font-medium text-slate-900">{{ formatTimestamp(displayHeartbeatAt, "relative") }}</span>
                </div>
              </div>
            </section>

            <LogTerminal
              :logs="lines"
              :is-live="connected"
              title="Agent 运行日志"
              empty-label="等待日志输出…"
            />
          </div>
        </ui-tab-pane>
      </ui-tabs>
    </div>
  </div>
</template>
