<script setup lang="ts">
import { computed, reactive, ref, watch } from "vue";
import { useRouter } from "vue-router";
import { IconRobot } from "@/lib/icons";

import { FormActionBar, FormPageLayout, PageHeader, SmartForm } from "@/components/index";
import { useAccounts, useMessage, useErrorHandler, usePlugins } from "@/composables";
import { useCreateAgent } from "@/composables/useAgents";
import type { Account } from "@/types";
import type { FieldConfig } from "@/components/smart-form.types";
import { to } from "@/router/registry";
import { buildAgentInput, formatAccountOptionLabel, parseAgentParamsJSON } from "@/utils/agentForm";

const router = useRouter();
const message = useMessage();
const { withErrorHandler } = useErrorHandler();
const create = useCreateAgent();
const { data: accounts } = useAccounts();
const { data: plugins } = usePlugins();

// 表单引用
const formRef = ref<InstanceType<typeof SmartForm>>();

// 表单数据
const formData = ref({
  name: "",
  plugin_key: "",
  action: "",
  account_id: "",
  params_json: "{}",
});

const pluginOptions = computed(() =>
  plugins.value
    .filter((plugin) => plugin.healthy)
    .map((plugin) => ({
      label: plugin.manifest.name ? `${plugin.manifest.name} (${plugin.manifest.key})` : plugin.manifest.key,
      value: plugin.manifest.key,
    })),
);

const selectedPlugin = computed(() =>
  plugins.value.find((plugin) => plugin.manifest.key === formData.value.plugin_key) ?? null,
);

const actionOptions = computed(() =>
  (selectedPlugin.value?.manifest.actions ?? []).map((action) => ({
    label: action.name ? `${action.name} (${action.key})` : action.key,
    value: action.key,
  })),
);

const filteredAccounts = computed(() =>
  accounts.value.filter((account) => {
    if (!formData.value.plugin_key) {
      return true;
    }
    return account.account_type_key === formData.value.plugin_key;
  }),
);

const accountOptions = computed(() =>
  filteredAccounts.value.map((account) => ({
    label: formatAccountOptionLabel(account),
    value: String(account.id),
  })),
);

const selectedAccount = computed<Account | null>(() =>
  filteredAccounts.value.find((account) => String(account.id) === formData.value.account_id)
  ?? accounts.value.find((account) => String(account.id) === formData.value.account_id)
  ?? null,
);

watch(() => formData.value.plugin_key, () => {
  if (!actionOptions.value.some((option) => option.value === formData.value.action)) {
    formData.value.action = "";
  }
  if (!accountOptions.value.some((option) => option.value === formData.value.account_id)) {
    formData.value.account_id = "";
  }
});

const formFields = computed<FieldConfig[]>(() => [
  {
    name: "name",
    label: "Agent 名称",
    type: "text",
    placeholder: "例如: GitHub Watcher",
    required: true,
    description: "用于标识此 Agent 的名称",
  },
  {
    name: "plugin_key",
    label: "插件",
    type: "select",
    placeholder: pluginOptions.value.length ? "请选择插件" : "暂无可用插件",
    required: true,
    description: "只展示当前可用的插件",
    options: pluginOptions.value,
  },
  {
    name: "action",
    label: "动作名称",
    type: "select",
    placeholder: formData.value.plugin_key ? "请选择动作" : "请先选择插件",
    required: true,
    description: "动作列表来自插件 manifest",
    options: actionOptions.value,
  },
  {
    name: "account_id",
    label: "关联账号",
    type: "select",
    placeholder: accountOptions.value.length ? "从账号库中选择一个账号" : "当前插件下暂无账号",
    description: formData.value.plugin_key
      ? "提交时会自动写入 input.account"
      : "先选择插件，再从账号库中挑选对应账号",
    options: accountOptions.value,
  },
  {
    name: "params_json",
    label: "动作参数 (JSON)",
    type: "textarea",
    placeholder: '{"interval_seconds":60}',
    description: "将写入 input.params，必须是 JSON 对象",
    rows: 6,
  },
]);

// 提交创建
async function handleSubmit() {
  // 验证表单
  const isValid = formRef.value?.validate();
  if (!isValid) {
    return;
  }

  let params: Record<string, unknown> = {};
  try {
    params = parseAgentParamsJSON(formData.value.params_json);
  } catch (error) {
    message.error(error instanceof Error ? error.message : "动作参数格式错误，请检查 JSON");
    return;
  }

  await withErrorHandler(
    async () => {
      await create.execute({
        name: formData.value.name.trim(),
        plugin_key: formData.value.plugin_key.trim(),
        action: formData.value.action.trim(),
        input: buildAgentInput(selectedAccount.value, params),
      });
      message.success("Agent 已创建");
      router.push(to.agents.list());
    },
    { action: "创建 Agent", showSuccess: false }
  );
}

// 取消创建
function handleCancel() {
  router.push(to.agents.list());
}
</script>

<template>
  <div class="page-shell">
    <PageHeader
      title="创建 Agent"
      subtitle="创建一个持续运行的 Agent"
      icon-bg="linear-gradient(135deg, rgba(10,132,255,0.12), rgba(10,132,255,0.06))"
      icon-color="var(--icon-purple)"
      :back-to="to.agents.list()"
      back-label="返回 Agent 列表"
    >
      <template #icon><icon-robot /></template>
    </PageHeader>

    <FormPageLayout>
      <template #main>
        <ui-card class="min-w-0">
          <template #title>
            <div class="flex items-center gap-2">
              <icon-robot class="h-5 w-5 text-[var(--accent)]" />
              <span>基本信息</span>
            </div>
          </template>

          <SmartForm
            ref="formRef"
            v-model="formData"
            :fields="formFields"
          />
        </ui-card>
      </template>

      <template #aside>
        <ui-card class="min-w-0 lg:sticky lg:top-[var(--space-6)]">
          <template #title>
            <div class="flex items-center gap-2">
              <icon-info-circle class="h-5 w-5 text-[var(--accent)]" />
              <span>关于 Agent</span>
            </div>
          </template>

          <div class="flex flex-col gap-4">
            <div class="rounded-xl border p-4 border-slate-200 bg-slate-50 shadow-sm">
              <h4 class="mb-3 text-sm font-semibold text-slate-900">什么是 Agent？</h4>
              <p class="text-sm leading-6 text-slate-500">
                Agent 是持续运行的插件进程，可以持续执行特定任务，例如监控数据变化、定期同步信息等。
              </p>
            </div>

            <div class="rounded-xl border p-4 border-slate-200 bg-slate-50 shadow-sm">
              <h4 class="mb-3 text-sm font-semibold text-slate-900">任务生命周期</h4>
              <div class="flex flex-col gap-3">
                <div class="flex items-start gap-3 rounded-lg border p-3 border-slate-200 bg-white shadow-sm">
                  <span class="mt-1.5 h-2.5 w-2.5 flex-shrink-0 rounded-full bg-slate-300"></span>
                  <div class="flex flex-col gap-0.5">
                    <div class="text-sm font-semibold text-slate-900">已停止 (stopped)</div>
                    <div class="text-xs leading-5 text-slate-500">Agent 创建后的初始状态</div>
                  </div>
                </div>
                <div class="flex items-start gap-3 rounded-lg border p-3 border-slate-200 bg-white shadow-sm">
                  <span class="mt-1.5 h-2.5 w-2.5 flex-shrink-0 rounded-full bg-emerald-500"></span>
                  <div class="flex flex-col gap-0.5">
                    <div class="text-sm font-semibold text-slate-900">运行中 (running)</div>
                    <div class="text-xs leading-5 text-slate-500">Agent 正在运行，可持续接收事件</div>
                  </div>
                </div>
                <div class="flex items-start gap-3 rounded-lg border p-3 border-slate-200 bg-white shadow-sm">
                  <span class="mt-1.5 h-2.5 w-2.5 flex-shrink-0 rounded-full bg-sky-500"></span>
                  <div class="flex flex-col gap-0.5">
                    <div class="text-sm font-semibold text-slate-900">停止中 (stopping)</div>
                    <div class="text-xs leading-5 text-slate-500">正在优雅停止 Agent</div>
                  </div>
                </div>
                <div class="flex items-start gap-3 rounded-lg border p-3 border-slate-200 bg-white shadow-sm">
                  <span class="mt-1.5 h-2.5 w-2.5 flex-shrink-0 rounded-full bg-red-500"></span>
                  <div class="flex flex-col gap-0.5">
                    <div class="text-sm font-semibold text-slate-900">错误 (error)</div>
                    <div class="text-xs leading-5 text-slate-500">Agent 运行出错，需要检查配置</div>
                  </div>
                </div>
              </div>
            </div>

            <div class="rounded-xl border p-4 border-slate-200 bg-slate-50 shadow-sm">
              <h4 class="mb-3 text-sm font-semibold text-slate-900">使用场景</h4>
              <ul class="pl-5 text-sm leading-7 text-slate-600 list-disc">
                <li>持续监控 GitHub 仓库变化</li>
                <li>定期检查邮箱新邮件</li>
                <li>实时同步服务器状态</li>
                <li>定时数据采集和处理</li>
              </ul>
            </div>
          </div>
        </ui-card>
      </template>

      <template #actions>
        <FormActionBar
          cancel-text="取消"
          submit-text="创建 Agent"
          submit-loading-text="创建中…"
          :submit-loading="create.loading.value"
          @cancel="handleCancel"
          @submit="handleSubmit"
        />
      </template>
    </FormPageLayout>
  </div>
</template>
