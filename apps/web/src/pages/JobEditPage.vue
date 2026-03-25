<script setup lang="ts">
import { ref, reactive, computed, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import { IconEdit, IconClockCircle } from "@/lib/icons";

import { FormActionBar, FormPageLayout, PageHeader, SmartForm } from "@/components/index";
import { useJobDefinitions, usePatchJobDefinition } from "@/composables/useJobs";
import { useMessage, useErrorHandler } from "@/composables";
import type { FieldConfig } from "@/components/smart-form.types";
import { to } from "@/router/registry";

const route = useRoute();
const router = useRouter();
const jobId = Number(route.params.id);

const message = useMessage();
const { withErrorHandler } = useErrorHandler();
const { data: definitions, loading } = useJobDefinitions();
const patch = usePatchJobDefinition();

const job = computed(() => definitions.value.find((j) => j.id === jobId));

const formRef = ref<InstanceType<typeof SmartForm>>();

const formData = ref({
  name: "",
  input: "{}",
  enabled: true,
  cron_expression: "",
  timezone: "UTC",
  schedule_enabled: false,
});

watch(job, (j) => {
  if (!j) return;
  formData.value.name = j.name;
  formData.value.input = JSON.stringify(j.input ?? {}, null, 2);
  formData.value.enabled = j.enabled;
  formData.value.cron_expression = j.schedule?.cron_expression ?? "";
  formData.value.timezone = j.schedule?.timezone ?? "UTC";
  formData.value.schedule_enabled = !!j.schedule?.cron_expression;
}, { immediate: true });

const formFields: FieldConfig[] = [
  {
    name: "name",
    label: "任务名称",
    type: "text",
    placeholder: "例如: GitHub 账号验证",
    required: true,
    description: "任务的显示名称，用于界面展示",
  },
  {
    name: "enabled",
    label: "启用任务",
    type: "switch",
    description: "关闭后任务将不会被自动调度执行",
  },
  {
    name: "input",
    label: "输入参数",
    type: "textarea",
    placeholder: '{"key":"value"}',
    description: "JSON 格式的输入参数，将传递给插件动作",
    rows: 5,
  },
  {
    name: "schedule_enabled",
    label: "启用定时调度",
    type: "switch",
    description: "开启后需要设置 Cron 表达式，任务将按计划自动执行",
  },
  {
    name: "cron_expression",
    label: "Cron 表达式",
    type: "text",
    placeholder: "0 * * * *",
    description: "格式：分 时 日 月 周。例如: 0 * * * * 表示每小时执行一次",
  },
  {
    name: "timezone",
    label: "时区",
    type: "select",
    defaultValue: "UTC",
    options: [
      { label: "UTC (协调世界时)", value: "UTC" },
      { label: "Asia/Shanghai (上海)", value: "Asia/Shanghai" },
      { label: "Asia/Tokyo (东京)", value: "Asia/Tokyo" },
      { label: "America/New_York (纽约)", value: "America/New_York" },
      { label: "Europe/London (伦敦)", value: "Europe/London" },
    ],
  },
];

const cronPresets = [
  { label: "每分钟", value: "* * * * *" },
  { label: "每小时", value: "0 * * * *" },
  { label: "每天 0 点", value: "0 0 * * *" },
  { label: "每周一 0 点", value: "0 0 * * 1" },
  { label: "每月 1 号 0 点", value: "0 0 1 * *" },
  { label: "工作日 9 点", value: "0 9 * * 1-5" },
];

function applyCronPreset(preset: string) {
  formData.value.cron_expression = preset;
}

async function handleSave() {
  const isValid = formRef.value?.validate();
  if (!isValid) {
    message.error("请检查表单填写是否正确");
    return;
  }

  let input: Record<string, unknown> = {};
  try {
    if (formData.value.input.trim()) {
      input = JSON.parse(formData.value.input) as Record<string, unknown>;
    }
  } catch {
    message.error("输入参数格式错误，请检查 JSON 格式");
    return;
  }

  await withErrorHandler(
    async () => {
      await patch.execute(jobId, {
        name: formData.value.name.trim(),
        input,
        enabled: formData.value.enabled,
        schedule: formData.value.schedule_enabled && formData.value.cron_expression.trim()
          ? {
              cron_expression: formData.value.cron_expression.trim(),
              timezone: formData.value.timezone,
              enabled: true,
            }
          : undefined,
      });
      message.success("任务已更新");
      router.push(to.jobs.detail(jobId));
    },
    { action: "更新任务", showSuccess: false }
  );
}
</script>

<template>
  <div class="page-shell">
    <PageHeader
      title="编辑任务"
      :subtitle="job ? `正在编辑 ${job.name}` : '任务详情加载中…'"
      icon-bg="linear-gradient(135deg, rgba(202,138,4,0.12), rgba(234,179,8,0.12))"
      icon-color="var(--icon-yellow)"
      :back-to="to.jobs.detail(jobId)"
      back-label="返回任务详情"
    >
      <template #icon><icon-edit /></template>
    </PageHeader>

    <FormPageLayout
      :loading="loading"
      :ready="!!job"
      empty-description="未找到该任务"
    >
      <template #empty-action>
        <ui-button type="primary" @click="router.push(to.jobs.list())">返回任务列表</ui-button>
      </template>

      <template #main>
        <ui-card class="min-w-0">
          <template #title>
            <div class="flex items-center gap-2">
              <icon-edit class="h-4 w-4 text-[var(--accent)]" />
              <span>编辑基本信息</span>
            </div>
          </template>

          <SmartForm ref="formRef" v-model="formData" :fields="formFields" />

          <div v-if="formData.schedule_enabled" class="mt-6 rounded-xl border border-dashed p-4 border-slate-200 bg-slate-50">
            <div class="text-xs font-semibold tracking-wider text-amber-700">快捷预设：</div>
            <div class="mt-3 flex flex-wrap gap-2">
              <ui-tag
                v-for="preset in cronPresets"
                :key="preset.value"
                class="cursor-pointer [transition-property:transform] hover:-translate-y-px"
                @click="applyCronPreset(preset.value)"
              >
                {{ preset.label }}
              </ui-tag>
            </div>
          </div>
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
              <span class="text-xs font-semibold tracking-wider text-slate-500">任务标识符</span>
              <code class="text-sm font-medium text-slate-700">{{ job?.key }}</code>
            </div>
            <div class="flex flex-col gap-2 rounded-xl border p-4 border-slate-200 bg-slate-50 shadow-sm">
              <span class="text-xs font-semibold tracking-wider text-slate-500">插件</span>
              <code class="text-sm font-medium text-slate-700">{{ job?.plugin_key }}</code>
            </div>
            <div class="flex flex-col gap-2 rounded-xl border p-4 border-slate-200 bg-slate-50 shadow-sm">
              <span class="text-xs font-semibold tracking-wider text-slate-500">动作</span>
              <span class="text-sm font-medium text-[var(--accent)]">{{ job?.action }}</span>
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
          @cancel="router.push(to.jobs.detail(jobId))"
          @submit="handleSave"
        />
      </template>
    </FormPageLayout>
  </div>
</template>
