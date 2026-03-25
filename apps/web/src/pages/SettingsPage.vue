<script setup lang="ts">
import { computed, onMounted, ref, watch } from "vue";
import {
  IconSettings,
  IconCloud,
  IconLock,
  IconTool,
  IconRefresh,
  IconCheck,
  IconSafe,
  IconInfoCircle,
} from "@/lib/icons";

import { useSystemStatus } from "@/composables/useDashboard";
import { useMessage, useErrorHandler, useSystemConfigs } from "@/composables";
import { useAuthStore } from "@/store";
import { storeToRefs } from "pinia";
import { PageHeader } from "@/components/index";

const { data: statusData, refresh: _refreshStatus } = useSystemStatus();
const message = useMessage();
const { withErrorHandler } = useErrorHandler();

const authStore = useAuthStore();
const { adminKey } = storeToRefs(authStore);
const adminKeySet = computed(() => Boolean(adminKey.value?.trim()));
const adminKeyDraft = ref("");
const adminKeyChanged = computed(() => adminKeyDraft.value !== (adminKey.value ?? ""));

const {
  knownConfigs,
  configs,
  configEdits,
  configSaving,
  loadConfigs,
  saveConfig,
} = useSystemConfigs();

onMounted(() => { void loadConfigs(); });

watch(
  adminKey,
  (value) => {
    adminKeyDraft.value = value ?? "";
  },
  { immediate: true }
);

function persistAdminKey(value: string) {
  authStore.setKey(value);
}

function handleSaveAdminKey() {
  persistAdminKey(adminKeyDraft.value);
  message.success(adminKeyDraft.value.trim() ? "管理员密钥已保存" : "管理员密钥已清空");
}

async function handleSaveConfig(configKey: string) {
  await withErrorHandler(
    async () => {
      await saveConfig(configKey);
      message.success("已保存");
    },
    { action: "保存配置" }
  );
}

// 刷新状态
async function refreshStatus() {
  await withErrorHandler(
    async () => {
      await _refreshStatus();
      message.success("状态已刷新");
    },
    { action: "刷新状态" }
  );
}
</script>

<template>
  <div class="page-shell flex flex-col gap-6">
    <PageHeader
      title="设置"
      subtitle="查看系统运行状态，管理配置、认证与访问安全"
      icon-bg="linear-gradient(135deg, rgba(10,132,255,0.12), rgba(10,132,255,0.06))"
      icon-color="var(--accent)"
    >
      <template #icon><icon-settings /></template>
      <template #actions>
        <ui-button @click="refreshStatus">
          <template #icon><icon-refresh /></template>
          刷新状态
        </ui-button>
      </template>
    </PageHeader>

    <!-- 顶部卡片网格 -->
    <div class="grid grid-cols-1 gap-6 lg:grid-cols-2">
      <!-- 系统状态 -->
      <ui-card class="min-w-0">
        <template #title>
          <div class="flex items-center gap-2">
            <icon-cloud class="h-5 w-5 text-sky-600" />
            <span>系统状态</span>
          </div>
        </template>

        <div class="flex flex-col">
          <div class="flex items-center justify-between gap-4 py-4 max-md:flex-col max-md:items-start max-md:gap-2">
            <span class="text-xs font-semibold tracking-wider text-slate-500">数据库连接</span>
            <div class="flex items-center gap-2">
              <span
                class="inline-block h-2 w-2 flex-shrink-0 rounded-full transition-colors"
                :class="statusData?.database_ok ? 'bg-emerald-500 animate-pulse' : 'bg-red-500'"
              />
              <span
                class="text-sm font-semibold"
                :class="statusData?.database_ok ? 'text-emerald-700' : 'text-red-700'"
              >
                {{ statusData?.database_ok ? "正常" : "异常" }}
              </span>
            </div>
          </div>
          <div class="flex items-center justify-between gap-4 py-4 max-md:flex-col max-md:items-start max-md:gap-2">
            <span class="text-xs font-semibold tracking-wider text-slate-500">已加载插件数</span>
            <span class="text-sm font-medium text-slate-900">{{ statusData?.plugin_count ?? "—" }}</span>
          </div>
          <div v-if="statusData?.now" class="flex items-center justify-between gap-4 py-4 max-md:flex-col max-md:items-start max-md:gap-2">
            <span class="text-xs font-semibold tracking-wider text-slate-500">服务器时间</span>
            <span class="text-sm font-medium text-slate-900 font-mono">{{ new Date(statusData.now).toLocaleString("zh-CN") }}</span>
          </div>
        </div>
      </ui-card>

      <!-- 管理员密钥设置 -->
      <ui-card class="min-w-0">
        <template #title>
          <div class="flex items-center gap-2">
            <icon-lock class="h-5 w-5 text-[var(--accent)]" />
            <span>管理员密钥设置</span>
          </div>
        </template>

        <div class="flex flex-col">
          <div class="flex items-center justify-between gap-4 py-4 max-md:flex-col max-md:items-start max-md:gap-2 border-b-0 pb-0">
            <span class="text-xs font-semibold tracking-wider text-slate-500">浏览器中的 Admin Key</span>
            <div class="flex items-center gap-2">
              <span
                class="inline-block h-2 w-2 flex-shrink-0 rounded-full transition-colors"
                :class="adminKeySet ? 'bg-emerald-500 animate-pulse' : 'bg-red-500'"
              />
              <span
                class="text-sm font-semibold"
                :class="adminKeySet ? 'text-emerald-700' : 'text-red-700'"
              >
                {{ adminKeySet ? "已配置" : "未配置" }}
              </span>
            </div>
          </div>
        </div>

        <form class="mt-4 rounded-xl border p-4 border-slate-200 bg-white/55 shadow-sm" @submit.prevent="handleSaveAdminKey">
          <div class="flex flex-col gap-3">
            <ui-input
              v-model="adminKeyDraft"
              placeholder="输入 Admin Key..."
              allow-clear
              type="password"
              class="w-full"
            />
            <div class="flex flex-col gap-3 [@media(min-width:768px)]:grid [@media(min-width:768px)]:gap-3 [@media(min-width:768px)]:grid-cols-[minmax(0,_1fr)_auto_auto]">
              <p class="text-sm leading-6 text-slate-500">
                保存于当前浏览器，填写后会自动在所有请求里附带 <code>X-Admin-Key</code>。
              </p>
              <ui-button
                class="self-start max-md:w-full max-md:justify-center"
                :disabled="!adminKeyChanged"
                @click="adminKeyDraft = ''"
              >
                清空
              </ui-button>
              <ui-button
                type="primary"
                html-type="submit"
                class="self-start max-md:w-full max-md:justify-center"
                :disabled="!adminKeyChanged"
              >
                <template #icon><icon-check /></template>
                保存
              </ui-button>
            </div>
          </div>
        </form>

        <div class="text-sm leading-6 text-slate-500 mt-4 flex flex-col gap-2 rounded-xl border p-4 border-slate-200 bg-slate-50 shadow-sm">
          <p>您的 Admin Key 已保存在本地浏览器中。</p>
          <p>该值由服务器端的环境变量 <code>ADMIN_KEY</code> 控制。</p>
        </div>
      </ui-card>
    </div>

    <div class="grid grid-cols-1 gap-6 lg:grid-cols-2">
      <ui-card class="min-w-0">
        <template #title>
          <div class="flex items-center gap-2">
            <icon-safe class="h-5 w-5 text-[var(--accent)]" />
            API 密钥管理
          </div>
        </template>

        <div class="flex flex-col gap-4">
          <p class="text-sm leading-6 text-slate-500">
            当前版本使用单个 <strong>Admin Key</strong> 作为 API 访问凭证，服务端通过
            <code class="inline-flex items-center rounded-md border border-slate-200 bg-slate-100 px-2 py-0.5 text-xs font-mono text-slate-600">ADMIN_KEY</code> 进行配置。
          </p>
          <p class="text-sm leading-6 text-slate-500">
            目前不提供多 API Key 的签发、轮换或撤销列表页面，统一通过管理员密钥进行认证。
          </p>

          <div class="overflow-hidden rounded-xl border border-slate-800 bg-slate-900 shadow-sm">
            <div class="flex items-center justify-between gap-3 border-b border-slate-800 px-4 py-2.5">
              <div class="flex items-center gap-2">
                <span class="h-2.5 w-2.5 rounded-full bg-red-400"></span>
                <span class="h-2.5 w-2.5 rounded-full bg-amber-400"></span>
                <span class="h-2.5 w-2.5 rounded-full bg-emerald-400"></span>
              </div>
              <span class="text-xs font-semibold tracking-wider text-slate-500">HTTP Request Header</span>
            </div>
            <div class="px-4 py-4 font-mono text-sm">
              <code><span class="text-[var(--accent)]/80">X-Admin-Key</span><span class="text-slate-500">: </span><span class="text-slate-200">&lt;your-admin-key&gt;</span></code>
            </div>
          </div>

          <p class="text-sm leading-6 text-slate-500 flex items-start gap-2">
            <icon-info-circle class="mt-0.5 flex-shrink-0 text-sky-500" />
            <strong>提示：</strong>在本页填写管理员密钥后，浏览器会自动在请求中携带认证信息。
          </p>
        </div>
      </ui-card>

      <ui-card class="min-w-0">
        <template #title>
          <div class="flex items-center gap-2">
            <icon-lock class="h-5 w-5 text-[var(--accent)]" />
            SSL 安全证书管理
          </div>
        </template>

        <div class="rounded-xl border p-5 shadow-sm border-sky-200 bg-sky-50/70">
          <icon-info-circle class="mb-2 text-sky-600" />
          <p class="text-sm font-semibold text-slate-900">通过服务器配置管理 TLS</p>
          <p class="mt-1 text-sm leading-6 text-slate-600">
            当前版本的 SSL 证书管理通过服务器配置完成，不提供 Web 界面操作。
            请在服务器配置文件或反向代理（如 Nginx、Caddy）中配置 TLS 证书。
          </p>
        </div>
      </ui-card>
    </div>

    <!-- 系统配置 -->
    <ui-card class="min-w-0">
      <template #title>
        <div class="flex items-center gap-2">
          <icon-tool class="h-5 w-5 text-[var(--accent)] text-slate-500" />
          <span>系统配置</span>
        </div>
      </template>

      <template #extra>
        <span class="text-sm leading-6 text-slate-500">配置值以 JSON 格式存储</span>
      </template>

      <div class="flex flex-col gap-4">
        <div v-for="c in knownConfigs" :key="c.key" class="flex items-start gap-3 rounded-xl border p-4 border-slate-200 bg-slate-50 shadow-sm flex-col">
          <div class="flex items-center justify-between gap-3 max-md:grid max-md:grid-cols-[1fr]">
            <span class="text-sm font-semibold text-slate-900">{{ c.label }}</span>
            <code class="inline-flex items-center rounded-md border border-slate-200 bg-slate-100 px-2 py-0.5 text-xs font-mono text-slate-600">{{ c.key }}</code>
          </div>
          <p class="text-sm text-slate-500">{{ c.description }}</p>
          <div class="flex flex-col gap-3 [@media(min-width:768px)]:grid [@media(min-width:768px)]:gap-3 [@media(min-width:768px)]:grid-cols-[minmax(0,_1fr)_auto]">
            <ui-input
              :model-value="configEdits[c.key] ?? ''"
              placeholder='例如: "OctoManger" 或 30'
              class="w-full"
              @update:modelValue="configEdits[c.key] = $event"
            />
            <ui-button
              type="primary"
              class="self-start max-md:w-full max-md:justify-center"
              :disabled="configSaving[c.key] || configEdits[c.key] === configs[c.key]"
              :loading="configSaving[c.key]"
              @click="handleSaveConfig(c.key)"
            >
              <template #icon><icon-check /></template>
              {{ configSaving[c.key] ? "保存中..." : "保存" }}
            </ui-button>
          </div>
        </div>
      </div>
    </ui-card>
  </div>
</template>
