<script setup lang="ts">
import { ref, computed, watch } from "vue";
import { useRoute } from "vue-router";
import { usePlugins, useSyncPlugins, usePluginSettings } from "@/composables/usePlugins";
import { useMessage } from "@/composables";
import { PageHeader } from "@/components/index";
import { to } from "@/router/registry";

const route = useRoute();
const pluginKey = route.params.id as string;
const message = useMessage();

const { data: plugins, loading, refresh } = usePlugins();
const plugin = computed(() => plugins.value.find((p) => p.manifest.key === pluginKey));

// ── Sync ──────────────────────────────────────────────────────────────────
const sync = useSyncPlugins();

async function handleSync() {
  try {
    const result = await sync.execute();
    await refresh();
    if (result.failed === 0) {
      message.success(`已同步 ${result.synced} 个账号类型`);
    } else {
      const errMsg = result.errors.join("; ");
      message.warning(`部分同步失败：${errMsg}`);
    }
  } catch (e) {
    message.error(e instanceof Error ? e.message : "同步失败");
  }
}

// ── Settings ──────────────────────────────────────────────────────────────
const settings = usePluginSettings(pluginKey);
const settingValues = ref<Record<string, string>>({});

// Load settings once plugin data is available
watch(plugin, async (p) => {
  if (!p) return;
  await settings.load();
  // Seed form from loaded values
  const vals: Record<string, string> = {};
  for (const s of (p.manifest.settings ?? [])) {
    vals[s.key] = settings.data.value[s.key] != null
      ? String(settings.data.value[s.key])
      : "";
  }
  settingValues.value = vals;
}, { immediate: true });

const SECRET_SETTING_KEYS = new Set(["token", "password", "secret", "api_key", "access_token", "refresh_token", "client_secret", "twocaptcha_api_key"]);

function isSecretSetting(key: string, secret: boolean): boolean {
  return secret || SECRET_SETTING_KEYS.has(key.toLowerCase());
}

async function saveSettings() {
  const payload: Record<string, unknown> = {};
  for (const [k, v] of Object.entries(settingValues.value)) {
    if (v !== "") payload[k] = v;
  }
  try {
    await settings.save(payload);
    message.success("设置已保存");
  } catch (e) {
    message.error(e instanceof Error ? e.message : "保存失败");
  }
}
</script>

<template>
  <div class="page-shell">
    <PageHeader
      :title="plugin ? plugin.manifest.name : '插件详情'"
      icon-bg="linear-gradient(135deg, rgba(236,72,153,0.12), rgba(219,39,119,0.12))"
      icon-color="var(--icon-pink)"
      :back-to="to.plugins.list()"
      back-label="返回插件列表"
    >
      <template #icon><icon-apps /></template>
      <template #subtitle>
        <template v-if="plugin">
          <code class="inline-flex items-center rounded-md border border-slate-200 bg-slate-100 px-2 py-0.5 text-xs font-mono text-slate-600 text-[var(--accent)]">v{{ plugin.manifest.version }}</code>
          &nbsp;·&nbsp;
          <span class="inline-flex items-center rounded-full border border-slate-200 bg-slate-50 px-2.5 py-1 text-xs font-medium text-slate-600">@{{ plugin.manifest.key }}</span>
        </template>
      </template>
    </PageHeader>

    <div v-if="loading" class="flex flex-col items-center justify-center gap-3 rounded-xl border border-dashed px-6 py-16 text-center border-slate-200 bg-white/55 shadow-sm">
      <ui-spin size="2.25em" />
    </div>
    <div v-else-if="!plugin" class="flex flex-col items-center justify-center gap-3 rounded-xl border border-dashed px-6 py-16 text-center border-slate-200 bg-white/55 shadow-sm min-h-[240px]">
      <icon-apps class="h-12 w-12 text-slate-400" />
      <p class="text-base font-semibold text-slate-900">未找到该插件。</p>
    </div>

    <div v-else class="grid grid-cols-1 gap-6 lg:grid-cols-[minmax(0,_1.15fr)_minmax(16em,_0.85fr)]">
      <!-- Actions list -->
      <ui-card class="min-w-0 flex-1 rounded-xl border overflow-hidden border-slate-200 bg-white shadow">
        <template #title>
          <div class="flex items-center gap-2">
            <icon-thunderbolt class="h-4 w-4 text-[var(--accent)]" />
            可用操作
          </div>
        </template>
        <div class="flex flex-col gap-3">
          <article
            v-for="act in plugin.manifest.actions"
            :key="act.key"
            class="flex flex-col items-start gap-2.5 rounded-xl border p-4 border-slate-200 bg-slate-50 shadow-sm"
          >
            <div class="flex w-full flex-wrap items-center gap-3 max-md:flex-col max-md:items-start max-md:gap-2">
              <code class="inline-flex items-center rounded-md border border-slate-200 bg-slate-100 px-2 py-0.5 text-xs font-mono text-slate-600 flex-shrink-0 self-start">{{ act.key }}</code>
              <span class="text-sm font-semibold text-slate-900">{{ act.name }}</span>
            </div>
            <p v-if="act.description" class="m-0 text-sm leading-6 text-slate-500">{{ act.description }}</p>
          </article>
          <p v-if="!plugin.manifest.actions.length" class="text-sm leading-6 text-slate-500 italic">
            该插件暂无可用操作。
          </p>
        </div>
      </ui-card>

      <!-- Sidebar info -->
      <div class="flex flex-col gap-6">
        <ui-card class="min-w-0 flex-1 rounded-xl border overflow-hidden border-slate-200 bg-white shadow">
          <template #title>
            <div class="flex items-center gap-2">
              <icon-info-circle class="h-4 w-4 text-sky-600" />
              插件信息
            </div>
          </template>
          <div class="flex flex-col">
            <div class="flex items-center justify-between gap-4 border-b border-slate-100 py-4 first:pt-0 last:border-b-0 last:pb-0 max-md:flex-col max-md:items-start max-md:gap-2">
              <span class="text-xs font-semibold tracking-wider text-slate-500">健康状态</span>
              <div class="inline-flex items-center gap-2">
                <span class="inline-block h-2 w-2 flex-shrink-0 rounded-full" :class="plugin.healthy ? 'bg-emerald-500 animate-pulse' : 'bg-red-500'" />
                <ui-tag :color="plugin.healthy ? 'green' : 'red'">
                  {{ plugin.healthy ? "healthy" : "degraded" }}
                </ui-tag>
              </div>
            </div>
            <div v-if="plugin.manifest.description" class="flex items-center justify-between gap-4 border-b border-slate-100 py-4 first:pt-0 last:border-b-0 last:pb-0 max-md:flex-col max-md:items-start max-md:gap-2">
              <span class="text-xs font-semibold tracking-wider text-slate-500">描述</span>
              <span class="text-sm font-medium text-slate-900 leading-7 text-slate-700 ml-auto text-left max-md:text-left">{{ plugin.manifest.description }}</span>
            </div>
            <div class="flex items-center justify-between gap-4 border-b border-slate-100 py-4 first:pt-0 last:border-b-0 last:pb-0 max-md:flex-col max-md:items-start max-md:gap-2 flex-col gap-3">
              <span class="text-xs font-semibold tracking-wider text-slate-500 mb-0">权限列表</span>
              <div v-if="plugin.manifest.capabilities.length" class="flex flex-wrap gap-2">
                <ui-tag v-for="cap in plugin.manifest.capabilities" :key="cap" size="small" class="whitespace-nowrap" color="blue">
                  {{ cap }}
                </ui-tag>
              </div>
              <span v-else class="text-sm leading-6 text-slate-500 italic">无特殊权限</span>
            </div>
          </div>
        </ui-card>

        <ui-card class="min-w-0 flex-1 rounded-xl border overflow-hidden border-slate-200 bg-white shadow">
          <template #title>
            <div class="flex items-center gap-2">
              <icon-tool class="h-4 w-4 text-amber-600" />
              操作
            </div>
          </template>
          <div class="flex flex-col gap-4">
            <div class="flex items-start gap-3 rounded-xl border p-4 border-slate-200 bg-slate-50 shadow-sm flex-col">
              <p class="text-sm font-semibold text-slate-900">同步账号类型</p>
              <p class="text-sm leading-6 text-slate-500">将插件定义的账号类型同步到系统中</p>
              <ui-button
                size="small"
                type="outline"
                :loading="sync.loading.value"
                @click="handleSync"
              >
                <template #icon><icon-sync /></template>
                立即同步
              </ui-button>
            </div>
          </div>
        </ui-card>

        <!-- Settings panel -->
        <ui-card
          v-if="plugin.manifest.settings?.length"
          class="min-w-0 flex-1 rounded-xl border overflow-hidden border-slate-200 bg-white shadow"
        >
          <template #title>
            <div class="flex items-center gap-2">
              <icon-settings class="h-4 w-4 text-[var(--accent)]" />
              插件设置
            </div>
          </template>
          <ui-spin :loading="settings.loading.value">
            <ui-form layout="vertical" class="flex flex-col gap-4">
              <ui-form-item
                v-for="s in plugin.manifest.settings"
                :key="s.key"
                class="mb-0"
              >
                <template #label>
                  <span class="text-sm font-semibold text-slate-900">{{ s.label || s.key }}</span>
                  <ui-tag v-if="s.required" size="small" color="red" class="ml-2">必填</ui-tag>
                </template>
                <div class="flex flex-col gap-2">
                  <ui-input
                    v-if="isSecretSetting(s.key, s.secret)"
                    v-model="settingValues[s.key]"
                    type="password"
                    allow-clear
                    class="w-full"
                    :placeholder="s.required ? '必填' : '可选'"
                  />
                  <ui-input
                    v-else
                    v-model="settingValues[s.key]"
                    allow-clear
                    class="w-full"
                    :placeholder="s.required ? '必填' : '可选'"
                  />
                  <p v-if="s.description" class="text-sm leading-6 text-slate-500">{{ s.description }}</p>
                </div>
              </ui-form-item>
            </ui-form>
          </ui-spin>
          <template #footer>
            <ui-button
              type="primary"
              class="max-md:w-full max-md:justify-center"
              :loading="settings.saving.value"
              @click="saveSettings"
            >
              保存设置
            </ui-button>
          </template>
        </ui-card>
      </div>
    </div>
  </div>
</template>
