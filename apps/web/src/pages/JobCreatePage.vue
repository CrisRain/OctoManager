<script setup lang="ts">
import { computed, ref, watch } from "vue";
import { useRouter } from "vue-router";
import { IconClockCircle, IconPlus } from "@/lib/icons";

import { FormActionBar, FormPageLayout, PageHeader, SmartForm } from "@/components/index";
import { useCreateJobDefinition } from "@/composables/useJobs";
import { useErrorHandler, useMessage, usePlugins } from "@/composables";
import type { FieldConfig } from "@/components/smart-form.types";
import { to } from "@/router/registry";
import { Notification } from "@/lib/feedback";

const router = useRouter();
const message = useMessage();
const { withErrorHandler } = useErrorHandler();
const create = useCreateJobDefinition();
const { data: plugins } = usePlugins();

// 表单引用
const formRef = ref<InstanceType<typeof SmartForm>>();

// 表单数据
const formData = ref({
  key: "",
  name: "",
  plugin_key: "",
  action: "",
  input: "{}",
  cron_expression: "",
  timezone: "UTC",
  schedule_enabled: false,
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

watch(() => formData.value.plugin_key, () => {
  if (!actionOptions.value.some((option) => option.value === formData.value.action)) {
    formData.value.action = "";
  }
});

const timezoneOptions = [
  { label: "UTC (协调世界时)", value: "UTC" },
  { label: "Asia/Shanghai (上海)", value: "Asia/Shanghai" },
  { label: "Asia/Tokyo (东京)", value: "Asia/Tokyo" },
  { label: "America/New_York (纽约)", value: "America/New_York" },
  { label: "Europe/London (伦敦)", value: "Europe/London" },
];

// 表单字段配置
const formFields = computed<FieldConfig[]>(() => [
  {
    name: "key",
    label: "任务标识符",
    type: "text",
    placeholder: "例如: github-verify",
    required: true,
    description: "用于唯一标识此任务定义，只能包含小写字母、数字和连字符",
  },
  {
    name: "name",
    label: "任务名称",
    type: "text",
    placeholder: "例如: GitHub 账号验证",
    required: true,
    description: "任务的显示名称，用于界面展示",
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
    name: "input",
    label: "输入参数",
    type: "textarea",
    placeholder: '{"username":"octocat"}',
    description: "JSON 格式的输入参数，将传递给插件动作",
    rows: 4,
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
    options: timezoneOptions,
  },
]);

// 预设 Cron 表达式选项
const cronPresets = [
  { label: "每分钟", value: "* * * * *" },
  { label: "每小时", value: "0 * * * *" },
  { label: "每天 0 点", value: "0 0 * * *" },
  { label: "每周一 0 点", value: "0 0 * * 1" },
  { label: "每月 1 号 0 点", value: "0 0 1 * *" },
  { label: "工作日 9 点", value: "0 9 * * 1-5" },
];

// 应用预设 Cron 表达式
function applyCronPreset(preset: string) {
  formData.value.cron_expression = preset;
}

// 提交创建
async function handleCreate() {
  // 验证表单
  const isValid = formRef.value?.validate();
  if (!isValid) {
    return;
  }

  // 解析输入参数
  let input: Record<string, unknown> = {};
  try {
    if (formData.value.input.trim()) {
      input = JSON.parse(formData.value.input) as Record<string, unknown>;
    }
  } catch (e) {
    message.error("输入参数格式错误，请检查 JSON 格式");
    return;
  }

  await withErrorHandler(
    async () => {
      await create.execute({
        key: formData.value.key.trim(),
        name: formData.value.name.trim(),
        plugin_key: formData.value.plugin_key.trim(),
        action: formData.value.action.trim(),
        input,
        schedule: formData.value.schedule_enabled && formData.value.cron_expression.trim()
          ? {
              cron_expression: formData.value.cron_expression.trim(),
              timezone: formData.value.timezone,
              enabled: true,
            }
          : undefined,
      });
      message.success("任务定义已创建");
      Notification.info({ title: "下一步", content: "点击「立即执行」手动触发一次，或在任务详情中配置触发器", duration: 7000 });
      router.push(to.jobs.list());
    },
    { action: "创建任务", showSuccess: false }
  );
}

// 取消创建
function handleCancel() {
  router.push(to.jobs.list());
}
</script>

<template>
  <div class="page-shell job-create-page">
    <PageHeader
      title="创建任务"
      subtitle="定义一个新的自动化任务，支持定时调度和手动触发"
      icon-bg="linear-gradient(135deg, rgba(202,138,4,0.12), rgba(234,179,8,0.12))"
      icon-color="var(--icon-yellow)"
      :back-to="to.jobs.list()"
      back-label="返回任务列表"
    >
      <template #icon><icon-clock-circle /></template>
    </PageHeader>

    <FormPageLayout>
      <template #main>
        <ui-card class="min-w-0">
          <template #title>
            <div class="flex items-center gap-2">
              <icon-plus class="h-5 w-5 text-[var(--accent)]" />
              <span>基本信息</span>
            </div>
          </template>

          <SmartForm
            ref="formRef"
            v-model="formData"
            :fields="formFields"
          />

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
              <icon-info-circle class="h-5 w-5 text-[var(--accent)]" />
              <span>Cron 表达式说明</span>
            </div>
          </template>

          <div class="flex flex-col gap-4">
            <div class="rounded-xl border p-5 border-slate-200 bg-slate-50 shadow-sm">
              <code class="inline-flex rounded-lg px-3 py-1 text-sm font-semibold text-slate-700 bg-slate-100">* * * * *</code>
              <div class="mt-4 flex flex-col gap-1 font-mono text-xs text-slate-500">
                <span>│ │ │ │ │</span>
                <span>│ │ │ │ └─ 星期几 (0-6, 0=周日)</span>
                <span>│ │ │ └─── 月份 (1-12)</span>
                <span>│ │ └───── 日期 (1-31)</span>
                <span>│ └─────── 小时 (0-23)</span>
                <span>└───────── 分钟 (0-59)</span>
              </div>
            </div>

            <div>
              <h4 class="mb-3 text-sm font-semibold text-slate-900">常用示例</h4>
              <div class="flex flex-col gap-3">
                <div class="flex items-start gap-3 rounded-xl border p-4 border-slate-200 bg-slate-50 shadow-sm flex-col">
                  <code class="text-slate-700 font-semibold">0 * * * *</code>
                  <span class="text-slate-500 text-sm">每小时执行一次</span>
                </div>
                <div class="flex items-start gap-3 rounded-xl border p-4 border-slate-200 bg-slate-50 shadow-sm flex-col">
                  <code class="text-slate-700 font-semibold">0 0 * * *</code>
                  <span class="text-slate-500 text-sm">每天 0 点执行</span>
                </div>
                <div class="flex items-start gap-3 rounded-xl border p-4 border-slate-200 bg-slate-50 shadow-sm flex-col">
                  <code class="text-slate-700 font-semibold">0 9 * * 1-5</code>
                  <span class="text-slate-500 text-sm">工作日上午 9 点执行</span>
                </div>
                <div class="flex items-start gap-3 rounded-xl border p-4 border-slate-200 bg-slate-50 shadow-sm flex-col">
                  <code class="text-slate-700 font-semibold">*/30 * * * *</code>
                  <span class="text-slate-500 text-sm">每 30 分钟执行一次</span>
                </div>
                <div class="flex items-start gap-3 rounded-xl border p-4 border-slate-200 bg-slate-50 shadow-sm flex-col">
                  <code class="text-slate-700 font-semibold">0 0 1 * *</code>
                  <span class="text-slate-500 text-sm">每月 1 号 0 点执行</span>
                </div>
              </div>
            </div>
          </div>
        </ui-card>
      </template>

      <template #actions>
        <FormActionBar
          cancel-text="取消"
          submit-text="创建任务"
          submit-loading-text="创建中…"
          :submit-loading="create.loading.value"
          @cancel="handleCancel"
          @submit="handleCreate"
        />
      </template>
    </FormPageLayout>
  </div>
</template>
