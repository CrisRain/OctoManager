<script setup lang="ts">
import { computed, reactive, ref, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import { IconEdit } from "@/lib/icons";

import { FormActionBar, FormPageLayout, PageHeader, SmartForm } from "@/components/index";
import { useAgent, usePatchAgent } from "@/composables/useAgents";
import { useAccounts, useMessage, useErrorHandler } from "@/composables";
import type { Account } from "@/types";
import type { FieldConfig } from "@/components/smart-form.types";
import { to } from "@/router/registry";
import {
  buildAgentInput,
  formatAccountOptionLabel,
  parseAgentParamsJSON,
  splitAgentInput,
  stringifyAgentParams,
} from "@/utils/agentForm";

const route = useRoute();
const router = useRouter();
const agentId = Number(route.params.id);

const message = useMessage();
const { withErrorHandler } = useErrorHandler();
const { data: agent, loading } = useAgent(agentId);
const patch = usePatchAgent();
const { data: accounts } = useAccounts();

const formRef = ref<InstanceType<typeof SmartForm>>();

const formData = ref({
  name: "",
  account_id: "",
  params_json: "{}",
});

watch(agent, (a) => {
  if (!a) return;
  const { accountId, params } = splitAgentInput(a.input);
  formData.value.name = a.name;
  formData.value.account_id = accountId;
  formData.value.params_json = stringifyAgentParams(params);
}, { immediate: true });

const filteredAccounts = computed(() =>
  accounts.value.filter((account) => {
    if (!agent.value?.plugin_key) {
      return true;
    }
    return account.account_type_key === agent.value.plugin_key;
  }),
);

const accountOptions = computed(() => {
  const options = filteredAccounts.value.map((account) => ({
    label: formatAccountOptionLabel(account),
    value: String(account.id),
  }));

  if (formData.value.account_id && !options.some((option) => option.value === formData.value.account_id)) {
    options.unshift({
      label: `当前已保存账号 #${formData.value.account_id}（账号库中未找到）`,
      value: formData.value.account_id,
    });
  }

  return options;
});

const selectedAccount = computed<Account | null>(() =>
  filteredAccounts.value.find((account) => String(account.id) === formData.value.account_id)
  ?? accounts.value.find((account) => String(account.id) === formData.value.account_id)
  ?? null,
);

const formFields = computed<FieldConfig[]>(() => [
  {
    name: "name",
    label: "Agent 名称",
    type: "text",
    placeholder: "例如: GitHub 监控",
    required: true,
    description: "Agent 的显示名称",
  },
  {
    name: "account_id",
    label: "关联账号",
    type: "select",
    placeholder: accountOptions.value.length ? "从账号库中选择一个账号" : "当前插件下暂无账号",
    description: "保存后会覆盖 input.account",
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

async function handleSave() {
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
      await patch.execute(agentId, {
        name: formData.value.name.trim(),
        input: buildAgentInput(selectedAccount.value, params),
      });
      message.success("Agent 已更新");
      router.push(to.agents.detail(agentId));
    },
    { action: "更新 Agent", showSuccess: false }
  );
}
</script>

<template>
  <div class="page-shell">
    <PageHeader
      title="编辑 Agent"
      :subtitle="agent ? `正在编辑 ${agent.name}` : 'Agent 详情加载中…'"
      icon-bg="linear-gradient(135deg, rgba(10,132,255,0.12), rgba(10,132,255,0.06))"
      icon-color="var(--icon-purple)"
      :back-to="to.agents.detail(agentId)"
      back-label="返回 Agent 详情"
    >
      <template #icon><icon-robot /></template>
    </PageHeader>

    <FormPageLayout
      :loading="loading"
      :ready="!!agent"
      empty-description="未找到该 Agent"
    >
      <template #empty-action>
        <ui-button type="primary" @click="router.push(to.agents.list())">返回 Agent 列表</ui-button>
      </template>

      <template #main>
        <ui-card class="min-w-0">
          <template #title>
            <div class="flex items-center gap-2">
              <icon-edit class="h-4 w-4 text-[var(--accent)]" />
              <span>编辑 Agent 配置</span>
            </div>
          </template>

          <SmartForm ref="formRef" v-model="formData" :fields="formFields" />
        </ui-card>
      </template>

      <template #aside>
        <ui-card class="min-w-0 lg:sticky lg:top-[var(--space-6)]">
          <template #title>
            <div class="flex items-center gap-2">
              <icon-info-circle class="h-4 w-4 text-[var(--accent)]" />
              <span>不可修改项</span>
            </div>
          </template>

          <div class="flex flex-col gap-3">
            <div class="flex flex-col gap-2 rounded-xl border p-4 border-slate-200 bg-slate-50 shadow-sm">
              <span class="text-xs font-semibold tracking-wider text-slate-500">插件</span>
              <code class="text-sm font-medium text-slate-700">{{ agent?.plugin_key }}</code>
            </div>
            <div class="flex flex-col gap-2 rounded-xl border p-4 border-slate-200 bg-slate-50 shadow-sm">
              <span class="text-xs font-semibold tracking-wider text-slate-500">动作</span>
              <span class="text-sm font-medium text-[var(--accent)]">{{ agent?.action }}</span>
            </div>
            <div class="flex flex-col gap-2 rounded-xl border p-4 border-slate-200 bg-slate-50 shadow-sm">
              <span class="text-xs font-semibold tracking-wider text-slate-500">期望状态</span>
              <span class="text-sm font-medium text-slate-700">{{ agent?.desired_state }}</span>
            </div>
          </div>
        </ui-card>
      </template>

      <template #actions>
        <FormActionBar
          cancel-text="取消"
          submit-text="保存修改"
          submit-loading-text="保存中…"
          :submit-loading="patch.loading.value"
          @cancel="router.push(to.agents.detail(agentId))"
          @submit="handleSave"
        />
      </template>
    </FormPageLayout>
  </div>
</template>
