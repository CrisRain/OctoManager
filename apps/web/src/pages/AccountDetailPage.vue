<script setup lang="ts">
import { useAccount } from "@/composables/useAccounts";
import { useMessage } from "@/composables";
import { useExecuteAccount } from "@/composables/useAccounts";
import { usePlugins } from "@/composables/usePlugins";
import { useCreateJobDefinition, useEnqueueJobExecution } from "@/composables/useJobs";
import { useCreateAgent } from "@/composables/useAgents";
import { to } from "@/router/registry";
import type {
  AccountExecuteResult,
  PluginUIButton,
  PluginUIFormField,
} from "@/types";

const route = useRoute();
const router = useRouter();
const accountId = Number(route.params.id);
const message = useMessage();

const { data: account, loading } = useAccount(accountId);
const { data: plugins } = usePlugins();
const executeAccount = useExecuteAccount();
const createJobDefinition = useCreateJobDefinition();
const enqueueJobExecution = useEnqueueJobExecution();
const createAgent = useCreateAgent();

const plugin = computed(() => {
  if (!account.value?.account_type_key) return null;
  return plugins.value.find((p) => p.manifest.key === account.value!.account_type_key) ?? null;
});

// ── Spec display ──────────────────────────────────────────────────────────
const SECRET_KEYS = new Set(["token", "password", "secret", "access_token", "refresh_token", "client_secret", "api_key"]);

function isSecret(key: string): boolean {
  return SECRET_KEYS.has(key.toLowerCase());
}

function displaySpecValue(key: string, val: unknown): string {
  if (val === null || val === undefined) return "—";
  if (isSecret(key)) return "••••••••";
  if (typeof val === "object") return JSON.stringify(val);
  return String(val);
}

const specEntries = computed((): [string, unknown][] => {
  if (!account.value?.spec) return [];
  return Object.entries(account.value.spec);
});

// ── Tab state ─────────────────────────────────────────────────────────────
const activeTab = ref("info");

// Whether this plugin has custom UI tabs defined
const hasUITabs = computed(() =>
  (plugin.value?.manifest.ui?.tabs?.length ?? 0) > 0
);

// ── Custom UI execution ───────────────────────────────────────────────────
// Active UI sub-tab (within the Actions tab)
const activeUITab = ref<string>("");

const pluginActionTabs = computed(() =>
  (plugin.value?.manifest.ui?.tabs ?? []).filter(tab => tab.context !== "create")
);

watch(
  pluginActionTabs,
  (tabs) => {
    if (!tabs.length) {
      activeUITab.value = "";
      return;
    }
    if (!tabs.some((tab) => tab.key === activeUITab.value)) {
      activeUITab.value = tabs[0].key;
    }
  },
  { immediate: true }
);

// Per-button form state: Map<button type="button"Action, formValues>
const formValues = ref<Record<string, Record<string, string>>>({});
const execResults = ref<Record<string, AccountExecuteResult | null>>({});
const execLoading = ref<Record<string, boolean>>({});

function initForm(button: PluginUIButton) {
  if (!formValues.value[button.action]) {
    const vals: Record<string, string> = {};
    for (const f of button.form) {
      vals[f.name] = "";
    }
    formValues.value[button.action] = vals;
  }
}

function fieldValue(action: string, name: string): string {
  return formValues.value[action]?.[name] ?? "";
}

function setFieldValue(action: string, name: string, val: string) {
  if (!formValues.value[action]) formValues.value[action] = {};
  formValues.value[action][name] = val;
}

async function executeButton(button: PluginUIButton) {
  if (!account.value) return;
  execLoading.value[button.action] = true;
  execResults.value[button.action] = null;
  try {
    const params: Record<string, unknown> = {
      ...button.params,
      ...formValues.value[button.action],
    };

    if (button.mode === "job") {
      const jobKey = `${plugin.value!.manifest.key}-${button.action.toLowerCase()}-${Date.now()}`;
      const jobDef = await createJobDefinition.execute({
        key: jobKey,
        name: `${button.label} (${account.value.identifier})`,
        plugin_key: plugin.value!.manifest.key,
        action: button.action,
        input: { params, account: { id: account.value.id, identifier: account.value.identifier, spec: account.value.spec } },
      });
      const exec = await enqueueJobExecution.execute(jobDef.id);
      message.success(`已提交后台作业：${button.label}`);
      router.push(to.jobs.executionDetail(exec.id));
    } else if (button.mode === "agent") {
      const agent = await createAgent.execute({
        name: `${button.label} (${account.value.identifier})`,
        plugin_key: plugin.value!.manifest.key,
        action: button.action,
        input: { params, account: { id: account.value.id, identifier: account.value.identifier, spec: account.value.spec } },
      });
      message.success(`已创建常驻任务：${button.label}`);
      router.push(to.agents.detail(agent.id));
    } else {
      const res = await executeAccount.execute(accountId, button.action, params);
      execResults.value[button.action] = res;
      if (res.status === "ok") {
        message.success("执行成功");
      } else {
        message.warning(`执行失败: ${res.error_message ?? res.error_code}`);
      }
    }
  } catch (e) {
    message.error(e instanceof Error ? e.message : "执行失败");
  } finally {
    execLoading.value[button.action] = false;
  }
}

function isSecretField(field: PluginUIFormField): boolean {
  return SECRET_KEYS.has(field.name.toLowerCase()) || field.type === "password";
}

// ── Fallback: flat action list (no ui.tabs) ───────────────────────────────
const selectedAction = ref<string | null>(null);
const paramsJSON = ref("{}");
const execResult = ref<AccountExecuteResult | null>(null);
const paramsError = ref("");

function selectAction(key: string) {
  selectedAction.value = key;
  paramsJSON.value = "{}";
  execResult.value = null;
  paramsError.value = "";
}

async function handleExecute() {
  if (!account.value || !selectedAction.value) return;
  paramsError.value = "";
  let params: Record<string, unknown> = {};
  try {
    params = JSON.parse(paramsJSON.value) as Record<string, unknown>;
  } catch {
    paramsError.value = "参数必须是合法 JSON";
    return;
  }
  
  const actionMeta = plugin.value?.manifest.actions.find((a) => a.key === selectedAction.value);
  const modes = actionMeta?.modes || ["sync"];
  const mode = modes.includes("job") ? "job" : modes.includes("agent") ? "agent" : "sync";

  try {
    if (mode === "job") {
      const jobKey = `${plugin.value!.manifest.key}-${selectedAction.value.toLowerCase()}-${Date.now()}`;
      const jobDef = await createJobDefinition.execute({
        key: jobKey,
        name: `${selectedAction.value} (${account.value.identifier})`,
        plugin_key: plugin.value!.manifest.key,
        action: selectedAction.value,
        input: { params, account: { id: account.value.id, identifier: account.value.identifier, spec: account.value.spec } },
      });
      const exec = await enqueueJobExecution.execute(jobDef.id);
      message.success(`已提交后台作业：${selectedAction.value}`);
      router.push(to.jobs.executionDetail(exec.id));
    } else if (mode === "agent") {
      const agent = await createAgent.execute({
        name: `${selectedAction.value} (${account.value.identifier})`,
        plugin_key: plugin.value!.manifest.key,
        action: selectedAction.value,
        input: { params, account: { id: account.value.id, identifier: account.value.identifier, spec: account.value.spec } },
      });
      message.success(`已创建常驻任务：${selectedAction.value}`);
      router.push(to.agents.detail(agent.id));
    } else {
      const res = await executeAccount.execute(accountId, selectedAction.value, params);
      execResult.value = res;
      if (res.status === "ok") {
        message.success("执行成功");
      } else {
        message.warning(`执行失败: ${res.error_message ?? res.error_code}`);
      }
    }
  } catch (e) {
    message.error(e instanceof Error ? e.message : "执行失败");
  }
}

const statusColor: Record<string, string> = { active: "green", pending: "gray", inactive: "gray" };
const statusDot: Record<string, string> = { active: "bg-emerald-500 animate-pulse", pending: "bg-slate-400", inactive: "bg-slate-300" };
const actionCount = computed(() => plugin.value?.manifest.actions?.length ?? 0);
const tagCount = computed(() => account.value?.tags?.length ?? 0);

const pluginDisplayName = computed(() => {
  if (plugin.value?.manifest.name) return plugin.value.manifest.name;
  return account.value?.account_type_key || "未匹配插件";
});

const overviewDescription = computed(() => {
  if (!account.value) return "";
  if (account.value.account_type_key) {
    return `当前账号绑定到 ${pluginDisplayName.value}，这里集中展示状态、凭据配置和可执行操作。`;
  }
  return "这里集中展示状态、凭据配置和可执行操作。";
});

const overviewTagPreview = computed(() => {
  const tags = account.value?.tags ?? [];
  if (!tags.length) return "未设置标签";
  if (tags.length <= 2) return tags.join(" · ");
  return `${tags.slice(0, 2).join(" · ")} +${tags.length - 2}`;
});

function describeSpecValue(key: string, val: unknown): string {
  if (isSecret(key)) return "敏感字段已脱敏";
  if (Array.isArray(val)) return `数组 · ${val.length} 项`;
  if (val && typeof val === "object") return "对象结构";
  if (typeof val === "number") return "数值参数";
  if (typeof val === "boolean") return "布尔参数";
  if (typeof val === "string" && /^https?:\/\//i.test(val)) return "URL 地址";
  return "文本参数";
}

function countTabButtons(tab: { sections: Array<{ buttons: PluginUIButton[] }> }) {
  return tab.sections.reduce((total, section) => total + section.buttons.length, 0);
}

function describeButtonMode(button: PluginUIButton) {
  return button.mode === "job" ? "作业模式" : "即时模式";
}

function describeButtonForm(button: PluginUIButton) {
  if (!button.form.length) return "无需输入字段";
  return `${button.form.length} 个输入字段`;
}

const selectedActionMeta = computed(() =>
  plugin.value?.manifest.actions.find((action) => action.key === selectedAction.value) ?? null
);
</script>

<template>
  <div class="page-shell">
    <PageHeader
      :title="account ? account.identifier : '账号详情'"
      icon-bg="var(--accent-light)"
      icon-color="var(--accent)"
      :back-to="to.accounts.list()"
      back-label="返回账号列表"
    >
      <template #icon><icon-user /></template>
      <template #subtitle>
        <code v-if="account" class="inline-flex items-center rounded-md border border-slate-200 bg-slate-100 px-2 py-0.5 text-xs font-mono text-slate-600">账号类型: {{ account.account_type_key || "—" }}</code>
      </template>
      <template #actions>
        <div v-if="account" class="flex items-center gap-3 flex-wrap justify-end gap-2">
          <div class="inline-flex items-center gap-2">
            <span class="inline-block h-2 w-2 flex-shrink-0 rounded-full transition-colors" :class="statusDot[account.status] ?? 'neutral'" />
            <ui-tag :color="statusColor[account.status] ?? 'gray'">{{ account.status }}</ui-tag>
          </div>
          <ui-button type="primary" @click="router.push(to.accounts.edit(accountId))">
            <template #icon><icon-edit /></template>
            编辑账号
          </ui-button>
        </div>
      </template>
    </PageHeader>

    <div v-if="loading" class="flex flex-col items-center justify-center gap-4 py-20 text-center">
      <ui-spin size="2.25em" />
    </div>
    <div v-else-if="!account" class="flex flex-col items-center justify-center gap-4 py-20 text-center">
      <icon-user class="h-12 w-12 text-slate-400" />
      <p class="text-sm text-slate-500">未找到该账号。</p>
      <ui-button @click="router.push(to.accounts.list())">返回账号列表</ui-button>
    </div>

    <div v-else-if="account" class="flex flex-col gap-6">
      <section class="flex flex-col gap-6 rounded-xl border p-6 lg:flex-row border-slate-200 bg-slate-50 shadow-sm backdrop-blur-xl backdrop-saturate-150">
        <div class="flex flex-1 flex-col gap-3">
          <span class="text-xs font-semibold uppercase tracking-wider text-[var(--accent)]">账号资源概览</span>
          <h2 class="m-0 text-2xl font-bold text-slate-900">{{ account.identifier }}</h2>
          <p class="text-sm text-slate-500">{{ overviewDescription }}</p>

          <div class="flex flex-wrap items-center gap-2">
            <div class="inline-flex items-center gap-2 rounded-full border px-3 py-1.5 text-sm border-slate-200 bg-slate-50">
              <span class="inline-block h-2 w-2 flex-shrink-0 rounded-full transition-colors" :class="statusDot[account.status] ?? 'neutral'" />
              <ui-tag :color="statusColor[account.status] ?? 'gray'">
                {{ account.status }}
              </ui-tag>
            </div>

            <div class="inline-flex items-center gap-2 rounded-full border px-3 py-1.5 text-xs text-slate-500 border-slate-200 bg-slate-50">
              <span>账号类型</span>
              <code class="text-xs font-semibold text-slate-900 font-[var(--font-mono)]">{{ account.account_type_key || "—" }}</code>
            </div>

            <div class="inline-flex items-center gap-2 rounded-full border px-3 py-1.5 text-xs text-slate-500 border-slate-200 bg-slate-50">
              <span>标签预览</span>
              <span class="font-medium text-slate-900">{{ overviewTagPreview }}</span>
            </div>
          </div>
        </div>

        <div class="grid grid-cols-2 gap-3 lg:min-w-[240px]">
          <div class="flex flex-col gap-1 rounded-xl border p-3 border-slate-200 bg-slate-50 shadow-sm">
            <span class="text-xs text-slate-500">关联插件</span>
            <div class="flex items-center gap-1">
              <span class="truncate text-sm font-semibold text-slate-900">{{ pluginDisplayName }}</span>
            </div>
            <span class="text-xs text-slate-500">{{ account.account_type_key || "未匹配插件" }}</span>
          </div>

          <div class="flex flex-col gap-1 rounded-xl border p-3 border-slate-200 bg-slate-50 shadow-sm">
            <span class="text-xs text-slate-500">凭据字段</span>
            <div class="flex items-center gap-1">
              <span class="text-2xl font-bold text-slate-900">{{ specEntries.length }}</span>
            </div>
            <span class="text-xs text-slate-500">当前 Spec 项数量</span>
          </div>

          <div class="flex flex-col gap-1 rounded-xl border p-3 border-slate-200 bg-slate-50 shadow-sm">
            <span class="text-xs text-slate-500">可用动作</span>
            <div class="flex items-center gap-1">
              <span class="text-2xl font-bold text-slate-900">{{ actionCount }}</span>
            </div>
            <span class="text-xs text-slate-500">{{ hasUITabs ? "支持分组动作面板" : "使用基础动作面板" }}</span>
          </div>

          <div class="flex flex-col gap-1 rounded-xl border p-3 border-slate-200 bg-slate-50 shadow-sm">
            <span class="text-xs text-slate-500">标签数量</span>
            <div class="flex items-center gap-1">
              <span class="text-2xl font-bold text-slate-900">{{ tagCount }}</span>
            </div>
            <span class="text-xs text-slate-500">{{ overviewTagPreview }}</span>
          </div>
        </div>
      </section>

      <ui-tabs
        v-model:active-key="activeTab"
        :destroy-on-hide="false"
        class="rounded-xl border border-slate-200 bg-white p-4 shadow-sm"
      >

        <!-- ── Tab 1: Info ─────────────────────────────────────────────────── -->
        <ui-tab-pane key="info" title="基础运行信息">
          <div class="pt-4">
            <div class="flex flex-col gap-4 lg:flex-row">
              <ui-card class="min-w-0 flex-1">
                <template #title>
                  <div class="flex items-center gap-2">
                    <icon-info-circle class="h-4 w-4 text-[var(--accent)]" />
                    基本信息与状态
                  </div>
                </template>
                <div class="flex flex-col gap-5">
                  <section class="mb-5 flex flex-col gap-4 rounded-xl border p-5 md:flex-row md:items-center md:justify-between border-slate-200 bg-slate-50">
                    <div class="flex flex-1 flex-col gap-2">
                      <span class="text-xs font-semibold uppercase tracking-wider text-[var(--accent)]">Account Profile</span>
                      <h3 class="m-0 text-base font-bold text-slate-900">{{ account.identifier }}</h3>
                      <p class="text-xs leading-relaxed text-slate-500">
                        当前账号关联插件 <code class="rounded bg-slate-100 px-1.5 py-0.5 text-xs font-mono text-slate-600">{{ pluginDisplayName }}</code>，
                        可以在这里查看资源状态、标签和凭据配置。
                      </p>
                    </div>

                    <div class="flex flex-shrink-0 flex-col items-center gap-1 text-center">
                      <span class="text-xs text-slate-500">运行状态</span>
                      <div class="inline-flex items-center gap-2">
                        <span class="inline-block h-2 w-2 flex-shrink-0 rounded-full transition-colors" :class="statusDot[account.status] ?? 'neutral'" />
                        <ui-tag :color="statusColor[account.status] ?? 'gray'">{{ account.status }}</ui-tag>
                      </div>
                    </div>
                  </section>

                  <div class="grid grid-cols-2 gap-3 sm:grid-cols-4">
                    <article class="flex flex-col gap-1 rounded-xl border p-4 md:p-5 border-slate-200 bg-slate-50">
                      <span class="text-xs text-slate-500">数据库主键</span>
                      <span class="text-xl font-bold text-slate-900 font-mono">#{{ account.id }}</span>
                      <span class="text-xs text-slate-500">系统资源编号</span>
                    </article>

                    <article class="flex flex-col gap-1 rounded-xl border p-4 md:p-5 border-slate-200 bg-slate-50">
                      <span class="text-xs text-slate-500">账号类型</span>
                      <code class="inline-flex items-center rounded bg-slate-200 px-2 py-0.5 text-xs font-mono font-semibold text-slate-700">{{ account.account_type_key || "—" }}</code>
                      <span class="text-xs text-slate-500">当前绑定类型</span>
                    </article>

                    <article class="flex flex-col gap-1 rounded-xl border p-4 md:p-5 border-slate-200 bg-slate-50">
                      <span class="text-xs text-slate-500">标签数量</span>
                      <span class="text-xl font-bold text-slate-900">{{ tagCount }}</span>
                      <span class="text-xs text-slate-500">{{ overviewTagPreview }}</span>
                    </article>

                    <article class="flex flex-col gap-1 rounded-xl border p-4 md:p-5 border-slate-200 bg-slate-50">
                      <span class="text-xs text-slate-500">可用动作</span>
                      <span class="text-xl font-bold text-slate-900">{{ actionCount }}</span>
                      <span class="text-xs text-slate-500">{{ hasUITabs ? "分组动作面板" : "基础动作面板" }}</span>
                    </article>
                  </div>

                  <section class="mt-5 rounded-xl border p-5 border-slate-200 bg-slate-50">
                    <div class="flex items-baseline justify-between">
                      <span class="text-sm font-semibold text-slate-900">分类标签</span>
                      <span class="text-xs text-slate-500">Tags / Labels</span>
                    </div>

                    <div v-if="account.tags?.length" class="flex flex-wrap items-center gap-1 gap-2">
                      <ui-tag v-for="tag in account.tags" :key="tag">
                        {{ tag }}
                      </ui-tag>
                    </div>
                    <div v-else class="text-sm italic text-slate-400">
                      当前账号还没有设置任何标签
                    </div>
                  </section>
                </div>
              </ui-card>

              <ui-card v-if="specEntries.length" class="min-w-0 flex-1">
                <template #title>
                  <div class="flex items-center gap-2">
                    <icon-lock class="h-4 w-4 text-orange-600" />
                    凭据密钥与 Spec 结构
                  </div>
                </template>
                <div class="flex flex-col gap-4">
                  <section class="mb-5 flex flex-col gap-4 rounded-xl border p-5 md:flex-row md:items-center md:justify-between border-slate-200 bg-slate-50">
                    <div class="flex flex-1 flex-col gap-2">
                      <span class="text-xs font-semibold uppercase tracking-wider text-[var(--accent)]">Credential Payload</span>
                      <h3 class="m-0 text-base font-bold text-slate-900">Spec 结构映射</h3>
                      <p class="text-xs leading-relaxed text-slate-500">
                        每个字段拆成独立条目展示，敏感值自动脱敏，方便快速核对当前账号的配置结构。
                      </p>
                    </div>

                    <div class="flex flex-shrink-0 flex-col items-center gap-1 text-center">
                      <span class="text-xs text-slate-500">字段总数</span>
                      <span class="text-3xl font-bold text-slate-900">{{ specEntries.length }}</span>
                    </div>
                  </section>

                  <div class="flex flex-col gap-2">
                    <article
                      v-for="[key, val] in specEntries"
                      :key="key"
                      class="flex flex-col gap-2 rounded-lg border px-4 py-3 sm:flex-row sm:items-center sm:justify-between border-slate-200 bg-slate-50"
                      :class="{ 'border-amber-100 bg-amber-50/60': isSecret(key) }"
                    >
                      <div class="flex min-w-0 flex-1 flex-col gap-0.5">
                        <span class="text-xs text-slate-500">字段名</span>
                        <code class="text-xs font-mono font-semibold text-slate-900">{{ key }}</code>
                      </div>

                      <div class="flex min-w-0 flex-1 flex-col gap-0.5 flex-[2]">
                        <span class="text-xs text-slate-500">{{ describeSpecValue(key, val) }}</span>
                        <span class="break-all text-sm text-slate-900 font-mono" :class="{ 'tracking-widest text-amber-700': isSecret(key) }">
                          {{ displaySpecValue(key, val) }}
                        </span>
                      </div>
                    </article>
                  </div>
                </div>
              </ui-card>

              <ui-card v-else class="min-w-0 flex-1">
                <template #title>
                  <div class="flex items-center gap-2">
                    <icon-lock class="h-4 w-4 text-slate-500" />
                    凭据密钥与 Spec 结构
                  </div>
                </template>
                <div class="flex flex-col items-center gap-2 py-10 text-center">
                  <div class="flex h-12 w-12 items-center justify-center rounded-full bg-slate-100 text-xl font-bold text-slate-400">0</div>
                  <p class="text-sm font-semibold text-slate-900">当前账号未配置任何凭据字段</p>
                  <p class="text-xs text-slate-500">如果后续补充了 Spec 结构，这里会自动展示对应字段。</p>
                </div>
              </ui-card>
            </div>
          </div>
        </ui-tab-pane>

        <!-- ── Tab 2: Actions ─────────────────────────────────────────────── -->
        <ui-tab-pane key="actions" title="操作">
          <div class="pt-4">
            <div v-if="!plugin" class="flex flex-col items-center justify-center gap-4 py-20 text-center">
              <icon-apps class="h-12 w-12 text-slate-400" />
              <p class="text-sm text-slate-500">
                未找到插件 <code class="inline-flex items-center rounded-md border border-slate-200 bg-slate-100 px-2 py-0.5 text-xs font-mono text-slate-600">{{ account.account_type_key }}</code>，
                请确认 Worker 已启动并注册该插件。
              </p>
            </div>

            <template v-else>
              <section class="mb-5 flex flex-col items-start gap-4 rounded-xl border p-5 sm:flex-row border-slate-200 bg-slate-50 shadow-sm">
                <div class="flex flex-1 flex-col gap-2">
                  <span class="text-xs font-semibold uppercase tracking-wider text-[var(--accent)]">Action Workbench</span>
                  <h3 class="m-0 text-base font-bold text-slate-900">账号操作面板</h3>
                  <p class="text-xs leading-relaxed text-slate-500">
                    {{ hasUITabs
                      ? "按分组查看插件提供的操作，带表单的操作可以直接在当前页面配置并执行。"
                      : "当前插件提供基础操作列表，可在左侧选择操作，并在右侧填写 JSON 参数后执行。"
                    }}
                  </p>
                </div>

                <div class="flex flex-shrink-0 flex-wrap items-center gap-4">
                  <article class="flex flex-col items-center gap-0.5 px-3 text-center">
                    <span class="text-xs text-slate-500">总动作数</span>
                    <span class="text-xl font-bold text-slate-900">{{ actionCount }}</span>
                  </article>
                  <article class="flex flex-col items-center gap-0.5 px-3 text-center">
                    <span class="text-xs text-slate-500">分组数量</span>
                    <span class="text-xl font-bold text-slate-900">{{ hasUITabs ? pluginActionTabs.length : 1 }}</span>
                  </article>
                  <article class="flex flex-col items-center gap-0.5 px-3 text-center">
                    <span class="text-xs text-slate-500">展示方式</span>
                    <span class="text-xs font-medium text-slate-900">{{ hasUITabs ? "分组视图" : "基础视图" }}</span>
                  </article>
                </div>
              </section>

              <!-- ── Custom UI: ui.tabs ──────────────────────────────────────── -->
              <template v-if="hasUITabs">
                <div class="rounded-xl border p-4 md:p-5 border-slate-200 bg-slate-50">
                  <ui-tabs v-model:active-key="activeUITab" type="card" :destroy-on-hide="false" class="ui-tabs">
                    <ui-tab-pane
                      v-for="tab in pluginActionTabs"
                      :key="tab.key"
                      :title="tab.label"
                    >
                      <div class="p-5">
                        <section class="mb-5 flex flex-col items-start gap-4 rounded-xl border p-4 sm:flex-row border-slate-200 bg-slate-50">
                          <div class="flex flex-1 flex-col gap-2">
                            <span class="text-xs font-semibold uppercase tracking-wider text-[var(--accent)]">Action Group</span>
                            <h3 class="m-0 text-base font-bold text-slate-900">{{ tab.label }}</h3>
                            <p class="text-xs text-slate-500">
                              这个分组下共有 {{ countTabButtons(tab) }} 个操作，可直接在当前页面填写参数并执行。
                            </p>
                          </div>

                          <div class="flex flex-shrink-0 flex-col items-center gap-1 text-center">
                            <span class="text-xs text-slate-500">动作数量</span>
                            <span class="text-3xl font-bold text-slate-900">{{ countTabButtons(tab) }}</span>
                          </div>
                        </section>

                        <div class="flex flex-col gap-4">
                          <section
                            v-for="section in tab.sections"
                            :key="section.title"
                            class="rounded-xl border p-4 md:p-5 border-slate-200 bg-white"
                          >
                            <div class="flex items-center justify-between border-b border-slate-200 px-4 py-3">
                              <div class="flex flex-col gap-0.5">
                                <p class="text-sm font-semibold text-slate-900">{{ section.title || tab.label }}</p>
                                <p class="text-xs text-slate-500">{{ section.buttons.length }} 个操作</p>
                              </div>
                            </div>

                            <div class="grid grid-cols-1 gap-4 p-4 lg:grid-cols-2">
                              <article
                                v-for="button in section.buttons"
                                :key="button.action"
                                class="flex flex-col gap-4 rounded-lg border p-4 border-slate-200 bg-slate-50"
                              >
                                <div class="flex items-start justify-between gap-3">
                                  <div class="flex flex-1 flex-col gap-2">
                                    <code class="text-xs font-mono text-[var(--accent)]">{{ button.action }}</code>
                                    <h4 class="m-0 text-sm font-semibold text-slate-900">{{ button.label }}</h4>
                                    <div class="flex items-center gap-2">
                                      <ui-tag size="small" :color="button.mode === 'job' ? 'blue' : 'gray'">
                                        {{ describeButtonMode(button) }}
                                      </ui-tag>
                                      <span class="text-xs text-slate-500">{{ describeButtonForm(button) }}</span>
                                    </div>
                                  </div>
                                  <span class="flex-shrink-0 rounded border border-slate-200 bg-slate-100 px-1.5 py-0.5 text-xs font-mono font-semibold text-slate-500">{{ button.form.length ? "表单" : "直接执行" }}</span>
                                </div>

                                <ui-form
                                  v-if="button.form.length"
                                  layout="vertical"
                                  class="ui-button-form flex flex-col gap-3"
                                  :model="formValues[button.action] ?? {}"
                                  @vue:mounted="initForm(button)"
                                >
                                  <div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
                                    <ui-form-item
                                      v-for="field in button.form"
                                      :key="field.name"
                                      class="mb-0"
                                    >
                                      <template #label>
                                        <span>{{ field.name }}</span>
                                        <ui-tag v-if="field.required" size="small" color="red" class="ml-1">必填</ui-tag>
                                        <span v-if="field.description" class="ml-1 text-xs font-normal text-slate-400">— {{ field.description }}</span>
                                      </template>

                                      <ui-select
                                        v-if="field.choices?.length"
                                        :model-value="fieldValue(button.action, field.name)"
                                        allow-clear
                                        :placeholder="field.required ? '必填' : '可选'"
                                        @change="(v: string) => setFieldValue(button.action, field.name, v)"
                                      >
                                        <ui-option v-for="c in field.choices" :key="c" :value="c">{{ c }}</ui-option>
                                      </ui-select>
                                      <ui-input
                                        v-else-if="isSecretField(field)"
                                        :model-value="fieldValue(button.action, field.name)"
                                        type="password"
                                        allow-clear
                                        :placeholder="field.required ? '必填' : '可选'"
                                        @update:model-value="(v: string) => setFieldValue(button.action, field.name, v)"
                                      />
                                      <ui-input
                                        v-else
                                        :model-value="fieldValue(button.action, field.name)"
                                        allow-clear
                                        :placeholder="field.required ? '必填' : '可选'"
                                        @update:model-value="(v: string) => setFieldValue(button.action, field.name, v)"
                                      />
                                    </ui-form-item>
                                  </div>

                                  <ui-button
                                    type="primary"
                                    class="self-start"
                                    :loading="execLoading[button.action]"
                                    @click="executeButton(button)"
                                  >
                                    <template #icon><icon-play-arrow /></template>
                                    {{ button.label }}
                                  </ui-button>
                                </ui-form>

                                <div v-else class="flex flex-col gap-3">
                                  <p class="text-xs text-slate-500">此操作无需额外输入字段，可直接执行。</p>
                                  <ui-button
                                    :type="button.variant === 'primary' ? 'primary' : 'outline'"
                                    class="self-start w-full"
                                    :loading="execLoading[button.action]"
                                    @click="initForm(button); executeButton(button)"
                                  >
                                    <template #icon><icon-play-arrow /></template>
                                    {{ button.label }}
                                  </ui-button>
                                </div>

                                <div
                                  v-if="execResults[button.action]"
                                  class="flex flex-col gap-3 rounded-lg border p-4"
                                  :class="execResults[button.action]!.status === 'ok' ? 'bg-emerald-50/60 border-emerald-200' : 'bg-red-50/60 border-red-200'"
                                >
                                  <div class="flex items-center gap-2">
                                    <icon-check-circle
                                      v-if="execResults[button.action]!.status === 'ok'"
                                      class="h-4 w-4 flex-shrink-0 text-emerald-600"
                                    />
                                    <icon-close-circle v-else class="h-4 w-4 flex-shrink-0 text-red-600" />
                                    <span class="text-sm font-semibold">
                                      {{ execResults[button.action]!.status === "ok"
                                        ? "执行成功"
                                        : `错误: ${execResults[button.action]!.error_code}` }}
                                    </span>
                                  </div>
                                  <p v-if="execResults[button.action]!.error_message" class="m-0 text-sm text-red-700">
                                    {{ execResults[button.action]!.error_message }}
                                  </p>
                                  <pre v-if="execResults[button.action]!.result" class="overflow-auto rounded-lg border border-slate-200 bg-slate-50 shadow-sm p-4 text-xs leading-relaxed text-slate-600 font-mono whitespace-pre-wrap break-all">{{
                                    JSON.stringify(execResults[button.action]!.result, null, 2)
                                  }}</pre>
                                </div>
                              </article>
                            </div>
                          </section>
                        </div>
                      </div>
                    </ui-tab-pane>
                  </ui-tabs>
                </div>
              </template>

              <!-- ── Fallback: flat action list ─────────────────────────────── -->
              <div v-else class="rounded-xl border p-4 md:p-5 border-slate-200 bg-slate-50 flex min-h-[400px]">
                <aside class="flex flex-shrink-0 flex-col border-r border-slate-200 bg-slate-50">
                  <div class="flex flex-col gap-1 border-b border-slate-200 p-4">
                    <span class="text-xs font-semibold uppercase tracking-wider text-[var(--accent)]">Action Directory</span>
                    <h3 class="m-0 text-sm font-bold text-slate-900">动作目录</h3>
                    <p class="text-xs text-slate-500">选择一个操作，在右侧填写 JSON 参数并立即执行。</p>
                  </div>

                  <div class="flex flex-1 flex-col gap-1 overflow-y-auto p-2">
                    <button
                      v-for="act in plugin.manifest.actions"
                      :key="act.key"
                      type="button"
                      class="flex cursor-pointer flex-col gap-1 rounded-xl border border-transparent p-3 text-left bg-white/30 [transition-property:background-color,_border-color,_box-shadow,_transform] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--accent)]/50 hover:border-slate-200 hover:bg-white hover:-translate-y-px"
                      :class="{ 'border-blue-500/20 bg-slate-50 shadow-sm': selectedAction === act.key }"
                      @click="selectAction(act.key)"
                    >
                      <div class="flex items-center justify-between gap-2">
                        <code class="inline-flex items-center rounded-md border border-slate-200 bg-slate-100 px-2 py-0.5 text-xs font-mono text-slate-600">{{ act.key }}</code>
                        <span class="text-xs font-semibold text-slate-400">操作</span>
                      </div>
                      <p class="m-0 text-sm font-medium text-slate-900">{{ act.name }}</p>
                      <p v-if="act.description" class="m-0 text-xs text-slate-500">{{ act.description }}</p>
                    </button>
                    <ui-empty v-if="!plugin.manifest.actions.length" description="该插件未注册任何操作" />
                  </div>
                </aside>

                <div class="flex-1 overflow-y-auto p-5">
                  <div v-if="!selectedAction" class="flex h-full flex-col items-center justify-center py-16 text-center text-sm text-slate-500 gap-3">
                    <icon-thunderbolt class="mb-2 h-10 w-10 text-slate-400" />
                    <p>请在左侧选择要执行的操作</p>
                  </div>
                  <template v-else>
                    <ui-card class="min-w-0 flex-1 rounded-xl border overflow-hidden border-slate-200 bg-white shadow">
                      <template #title>
                        <div class="flex items-center gap-2">
                          <div class="flex h-7 w-7 flex-shrink-0 items-center justify-center rounded-lg bg-purple-50 text-purple-600"><icon-code-block /></div>
                          {{ selectedActionMeta?.name || "运行时调用" }}
                        </div>
                      </template>
                      <div class="flex flex-col gap-5">
                        <section class="flex items-start justify-between gap-3">
                          <div class="flex flex-1 flex-col gap-2">
                            <code class="text-xs font-mono text-[var(--accent)]">{{ selectedAction }}</code>
                            <p class="text-sm text-slate-500">
                              {{ selectedActionMeta?.description || "为该操作填写 JSON 参数，然后在当前页面直接执行。" }}
                            </p>
                          </div>
                          <span class="flex-shrink-0 rounded border border-slate-200 bg-slate-100 px-2 py-0.5 text-xs font-mono font-semibold text-slate-600">JSON</span>
                        </section>

                        <ui-form layout="vertical" :model="{}">
                          <ui-form-item label="执行参数 (JSON)">
                            <ui-textarea
                              v-model="paramsJSON"
                              :auto-size="{ minRows: 6, maxRows: 14 }"
                              class="font-mono text-sm"
                              placeholder="{}"
                            />
                            <p v-if="paramsError" class="mt-1 text-xs text-red-600">{{ paramsError }}</p>
                          </ui-form-item>
                          <ui-button
                            type="primary"
                            :loading="executeAccount.loading.value"
                            @click="handleExecute"
                          >
                            <template #icon><icon-play-arrow /></template>
                            执行操作
                          </ui-button>
                        </ui-form>

                        <div v-if="execResult" class="flex flex-col gap-3 rounded-lg border p-4 mt-5" :class="execResult.status === 'ok' ? 'bg-emerald-50/60 border-emerald-200' : 'bg-red-50/60 border-red-200'">
                          <div class="flex items-center gap-2">
                            <icon-check-circle v-if="execResult.status === 'ok'" class="h-4 w-4 flex-shrink-0 text-emerald-600" />
                            <icon-close-circle v-else class="h-4 w-4 flex-shrink-0 text-red-600" />
                            <span class="text-sm font-semibold">
                              {{ execResult.status === "ok" ? "指令下发并执行成功" : `执行异常: ${execResult.error_code}` }}
                            </span>
                          </div>
                          <p v-if="execResult.error_message" class="m-0 text-sm text-red-700">{{ execResult.error_message }}</p>
                          <pre v-if="execResult.result" class="overflow-auto rounded-lg border border-slate-200 bg-slate-50 shadow-sm p-4 text-xs leading-relaxed text-slate-600 font-mono whitespace-pre-wrap break-all">{{ JSON.stringify(execResult.result, null, 2) }}</pre>
                        </div>
                      </div>
                    </ui-card>
                  </template>
                </div>
              </div>
            </template>
          </div>
        </ui-tab-pane>

      </ui-tabs>
    </div>
  </div>
</template>
