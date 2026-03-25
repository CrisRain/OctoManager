<script setup lang="ts">
import { computed, ref, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import { IconEdit } from "@/lib/icons";

import { FormActionBar, FormPageLayout, PageHeader, SmartForm } from "@/components/index";
import { useJobDefinitions } from "@/composables/useJobs";
import { useTriggers, usePatchTrigger } from "@/composables/useTriggers";
import { useMessage, useErrorHandler } from "@/composables";
import type { FieldConfig } from "@/components/smart-form.types";
import { to } from "@/router/registry";
import {
  formatJobDefinitionOptionLabel,
  parseTriggerDefaultInput,
  stringifyTriggerDefaultInput,
} from "@/utils/triggerForm";

const route = useRoute();
const router = useRouter();
const triggerId = Number(route.params.id);

const message = useMessage();
const { withErrorHandler } = useErrorHandler();
const { data: triggers, loading } = useTriggers();
const { data: definitions } = useJobDefinitions();
const patch = usePatchTrigger();

const trigger = computed(() => triggers.value.find((t) => t.id === triggerId));

const formRef = ref<InstanceType<typeof SmartForm>>();

const formData = ref({
  name: "",
  job_definition_id: "",
  mode: "async",
  default_input_json: "{}",
  enabled: true,
});

watch(trigger, (t) => {
  if (!t) return;
  formData.value.name = t.name;
  formData.value.job_definition_id = String(t.job_definition_id);
  formData.value.mode = t.mode;
  formData.value.default_input_json = stringifyTriggerDefaultInput(t.default_input);
  formData.value.enabled = t.enabled;
}, { immediate: true });

const jobOptions = computed(() => {
  const options = definitions.value.map((item) => ({
    label: formatJobDefinitionOptionLabel(item),
    value: String(item.id),
  }));

  if (
    trigger.value
    && formData.value.job_definition_id
    && !options.some((option) => option.value === formData.value.job_definition_id)
  ) {
    options.unshift({
      label: `当前已绑定任务 #${formData.value.job_definition_id}（任务列表中未找到）`,
      value: formData.value.job_definition_id,
    });
  }

  return options;
});

const selectedDefinition = computed(() =>
  definitions.value.find((item) => String(item.id) === formData.value.job_definition_id) ?? null,
);

const formFields = computed<FieldConfig[]>(() => [
  {
    name: "name",
    label: "触发器名称",
    type: "text",
    placeholder: "例如: GitHub Webhook",
    required: true,
    description: "触发器的显示名称",
  },
  {
    name: "job_definition_id",
    label: "绑定任务定义",
    type: "select",
    placeholder: jobOptions.value.length ? "选择要触发的任务定义" : "暂无任务定义",
    required: true,
    description: selectedDefinition.value
      ? `当前将触发 ${selectedDefinition.value.plugin_key}:${selectedDefinition.value.action}`
      : "可以切换到新的任务定义",
    options: jobOptions.value,
  },
  {
    name: "mode",
    label: "执行模式",
    type: "select",
    options: [
      { label: "异步（async）— 立即返回，后台执行", value: "async" },
      { label: "同步（sync）— 等待执行完成后返回结果", value: "sync" },
    ],
    description: "sync 模式下请求会阻塞直到任务完成",
  },
  {
    name: "default_input_json",
    label: "默认输入 (JSON)",
    type: "textarea",
    placeholder: "{\"source\":\"github\"}",
    description: "Webhook 请求体会与这里的默认输入合并；必须是 JSON 对象",
    rows: 6,
  },
  {
    name: "enabled",
    label: "启用触发器",
    type: "switch",
    description: "关闭后此触发器将拒绝所有 Webhook 请求",
  },
]);

async function handleSave() {
  const isValid = formRef.value?.validate();
  if (!isValid) {
    return;
  }

  let defaultInput: Record<string, unknown> = {};
  try {
    defaultInput = parseTriggerDefaultInput(formData.value.default_input_json);
  } catch (error) {
    message.error(error instanceof Error ? error.message : "默认输入格式错误");
    return;
  }

  await withErrorHandler(
    async () => {
      await patch.execute(triggerId, {
        name: formData.value.name.trim(),
        job_definition_id: Number(formData.value.job_definition_id),
        mode: formData.value.mode,
        default_input: defaultInput,
        enabled: formData.value.enabled,
      });
      message.success("触发器已更新");
      router.push(to.triggers.list());
    },
    { action: "更新触发器", showSuccess: false }
  );
}
</script>

<template>
  <div class="page-shell">
    <PageHeader
      title="编辑触发器"
      :subtitle="trigger ? `正在编辑 ${trigger.name}` : '触发器详情加载中…'"
      icon-bg="linear-gradient(135deg, rgba(234,179,8,0.12), rgba(202,138,4,0.12))"
      icon-color="var(--icon-yellow)"
      :back-to="to.triggers.list()"
      back-label="返回触发器列表"
    >
      <template #icon><icon-thunderbolt /></template>
    </PageHeader>

    <FormPageLayout
      :loading="loading"
      :ready="!!trigger"
      empty-description="未找到该触发器"
    >
      <template #empty-action>
        <ui-button type="primary" @click="router.push(to.triggers.list())">返回触发器列表</ui-button>
      </template>

      <template #main>
        <ui-card class="min-w-0">
          <template #title>
            <div class="flex items-center gap-2">
              <icon-edit class="h-4 w-4 text-[var(--accent)]" />
              <span>编辑触发器配置</span>
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
              <span class="text-xs font-semibold tracking-wider text-slate-500">触发器标识符</span>
              <code class="text-sm font-medium text-slate-700">{{ trigger?.key }}</code>
            </div>
            <div class="flex flex-col gap-2 rounded-xl border p-4 border-slate-200 bg-slate-50 shadow-sm">
              <span class="text-xs font-semibold tracking-wider text-slate-500">关联任务定义 ID</span>
              <span class="text-sm font-medium text-slate-700">#{{ trigger?.job_definition_id }}</span>
            </div>
            <div v-if="selectedDefinition" class="flex flex-col gap-2 rounded-xl border p-4 border-slate-200 bg-slate-50 shadow-sm">
              <span class="text-xs font-semibold tracking-wider text-slate-500">当前目标任务</span>
              <span class="text-sm font-medium text-slate-700">{{ selectedDefinition.name }}</span>
              <code class="text-xs text-slate-500">{{ selectedDefinition.plugin_key }}:{{ selectedDefinition.action }}</code>
            </div>
            <div class="flex flex-col gap-2 rounded-xl border p-4 border-slate-200 bg-slate-50 shadow-sm">
              <span class="text-xs font-semibold tracking-wider text-slate-500">Token 前缀</span>
              <code class="text-sm font-medium text-slate-700">{{ trigger?.token_prefix }}...</code>
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
          @cancel="router.push(to.triggers.list())"
          @submit="handleSave"
        />
      </template>
    </FormPageLayout>
  </div>
</template>
