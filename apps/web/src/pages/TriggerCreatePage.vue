<script setup lang="ts">
import { computed, ref } from "vue";
import { useRouter } from "vue-router";
import { useJobDefinitions } from "@/composables/useJobs";
import { useMessage } from "@/composables";
import { useCreateTrigger } from "@/composables/useTriggers";
import { FormActionBar, FormPageLayout, PageHeader, SmartForm } from "@/components/index";
import type { FieldConfig } from "@/components/smart-form.types";
import { to } from "@/router/registry";
import {
  formatJobDefinitionOptionLabel,
  parseTriggerDefaultInput,
} from "@/utils/triggerForm";

const router = useRouter();
const message = useMessage();
const { data: definitions, loading: loadingDefinitions, error: errorDefinitions } = useJobDefinitions();
const create = useCreateTrigger();

const formRef = ref<InstanceType<typeof SmartForm>>();
const formData = ref({
  key: "",
  name: "",
  job_definition_id: "",
  mode: "async",
  default_input_json: "{}",
  enabled: true,
});
const lastToken = ref("");
const copied = ref(false);

const jobOptions = computed(() =>
  definitions.value.map((item) => ({
    label: formatJobDefinitionOptionLabel(item),
    value: String(item.id),
  })),
);

const selectedDefinition = computed(() =>
  definitions.value.find((item) => String(item.id) === formData.value.job_definition_id) ?? null,
);

const webhookPath = computed(() => {
  const key = formData.value.key.trim();
  return `/api/v2/webhooks/${key ? encodeURIComponent(key) : ":key"}`;
});

const webhookURL = computed(() => {
  if (typeof window === "undefined") {
    return webhookPath.value;
  }
  return `${window.location.origin}${webhookPath.value}`;
});

const formFields = computed<FieldConfig[]>(() => [
  {
    name: "key",
    label: "键名",
    type: "text",
    placeholder: "github-webhook",
    required: true,
    description: "Webhook 路径的一部分，建议使用小写字母、数字和连字符",
  },
  {
    name: "name",
    label: "名称",
    type: "text",
    placeholder: "GitHub Webhook",
    required: true,
    description: "用于界面展示的触发器名称",
  },
  {
    name: "job_definition_id",
    label: "绑定任务定义",
    type: "select",
    placeholder: loadingDefinitions.value
      ? "任务定义加载中…"
      : definitions.value.length
        ? "选择要触发的任务定义"
        : "暂无任务定义",
    required: true,
    description: selectedDefinition.value
      ? `当前将触发 ${selectedDefinition.value.plugin_key}:${selectedDefinition.value.action}`
      : "触发器收到 Webhook 后，会执行这里选中的任务定义",
    options: jobOptions.value,
  },
  {
    name: "mode",
    label: "执行模式",
    type: "select",
    required: true,
    description: "异步模式适合耗时任务；同步模式会等待执行结果返回",
    options: [
      { label: "async（异步）", value: "async" },
      { label: "sync（同步等待）", value: "sync" },
    ],
  },
  {
    name: "enabled",
    label: "创建后立即启用",
    type: "switch",
    description: "关闭后 Webhook 会创建成功，但不会接受请求，直到你手动启用",
  },
  {
    name: "default_input_json",
    label: "默认输入 (JSON)",
    type: "textarea",
    rows: 6,
    placeholder: "{\"source\":\"github\"}",
    description: "Webhook 请求体会与这里的默认输入合并；必须是 JSON 对象",
  },
]);

async function copyToken() {
  try {
    await navigator.clipboard.writeText(lastToken.value);
    copied.value = true;
    setTimeout(() => { copied.value = false; }, 2000);
  } catch { /* ignore */ }
}

async function handleCreate() {
  const isValid = formRef.value?.validate();
  if (!isValid) {
    return;
  }

  if (!formData.value.job_definition_id) {
    message.error("请选择要绑定的任务定义");
    return;
  }

  let defaultInput: Record<string, unknown> = {};
  try {
    defaultInput = parseTriggerDefaultInput(formData.value.default_input_json);
  } catch (error) {
    message.error(error instanceof Error ? error.message : "默认输入格式错误");
    return;
  }

  try {
    const result = await create.execute({
      key: formData.value.key.trim(),
      name: formData.value.name.trim(),
      job_definition_id: Number(formData.value.job_definition_id),
      mode: formData.value.mode,
      default_input: defaultInput,
      enabled: formData.value.enabled,
    });
    if (result && typeof result === "object" && "delivery_token" in result) {
      lastToken.value = result.delivery_token as string;
    }
    copied.value = false;
    message.success("触发器已创建");
  } catch (e) {
    message.error(e instanceof Error ? e.message : "创建失败");
  }
}
</script>

<template>
  <div class="page-shell">
    <PageHeader
      title="创建触发器"
      subtitle="创建一个新的 Webhook 触发器"
      icon-bg="linear-gradient(135deg, rgba(234,179,8,0.12), rgba(202,138,4,0.12))"
      icon-color="var(--icon-yellow)"
      :back-to="to.triggers.list()"
      back-label="返回触发器列表"
    >
      <template #icon><icon-thunderbolt /></template>
    </PageHeader>

    <FormPageLayout>
      <template #main>
        <ui-card class="min-w-0">
          <template #title>
            <div class="flex items-center gap-2">
              <icon-thunderbolt class="h-5 w-5 text-[var(--accent)]" />
              <span>基本信息</span>
            </div>
          </template>

          <div
            v-if="errorDefinitions"
            class="mb-6 rounded-xl border border-red-200 bg-red-50 px-4 py-3 text-sm leading-6 text-red-700"
          >
            任务定义加载失败：{{ errorDefinitions }}
          </div>

          <div
            v-else-if="!loadingDefinitions && !definitions.length"
            class="mb-6 flex flex-col gap-3 rounded-xl border border-amber-200 bg-amber-50 px-4 py-4 text-sm leading-6 text-amber-800"
          >
            <div>当前还没有可绑定的任务定义，所以下拉列表会为空。</div>
            <div>请先创建一个任务，再回来创建触发器。</div>
            <div>
              <ui-button size="small" type="outline" @click="router.push(to.jobs.create())">
                去创建任务
              </ui-button>
            </div>
          </div>

          <SmartForm
            ref="formRef"
            v-model="formData"
            :fields="formFields"
          />

          <div v-if="lastToken" class="mt-6 rounded-xl border p-5 border-slate-200 bg-slate-50 shadow-sm">
            <div class="mb-3 flex items-start justify-between gap-3">
              <div class="flex items-center gap-2 text-sm font-semibold text-emerald-800">
                <span class="inline-block flex-shrink-0 rounded-full bg-slate-400 h-2 w-2 [@media(prefers-reduced-motion:no-preference)]:[&.online]:bg-emerald-500 [@media(prefers-reduced-motion:no-preference)]:[&.online]:animate-[pulse-dot_2s_ease-in-out_infinite] [&.offline]:bg-red-500 [&.neutral]:bg-slate-400 online" />
                <span>创建成功！请保存您的 Delivery Token</span>
              </div>
              <ui-button size="mini" type="text" @click="copyToken">
                <template #icon><icon-copy /></template>
                {{ copied ? "已复制" : "复制" }}
              </ui-button>
            </div>
            <code class="block w-full overflow-auto rounded-xl border px-4 py-3 text-[13px] text-slate-900 border-slate-200 bg-white/70">{{ lastToken }}</code>
            <p class="mt-3 text-xs text-emerald-700">注意：Token 只会显示一次，请妥善保管。</p>
          </div>
        </ui-card>
      </template>

      <template #aside>
        <ui-card class="min-w-0 lg:sticky lg:top-[var(--space-6)]">
          <template #title>
            <div class="flex items-center gap-2">
              <icon-info-circle class="h-5 w-5 text-[var(--accent)]" />
              <span>关于触发器</span>
            </div>
          </template>
          <div class="flex flex-col gap-4">
            <div class="rounded-xl border p-4 border-slate-200 bg-slate-50 shadow-sm">
              <p class="text-sm leading-6 text-slate-500">
                触发器允许你通过 Webhook 从外部系统触发任务定义的执行。
              </p>
            </div>

            <div class="rounded-xl border p-4 border-slate-200 bg-slate-50 shadow-sm">
              <h4 class="mb-3 text-sm font-semibold text-slate-900">Webhook 地址</h4>
              <code class="block overflow-auto rounded-lg border border-slate-200 bg-white px-3 py-2 text-xs text-slate-700">{{ webhookURL }}</code>
              <p class="mt-3 text-xs leading-5 text-slate-500">请求时需要携带 `X-Trigger-Token` 请求头。</p>
            </div>

            <div v-if="selectedDefinition" class="rounded-xl border p-4 border-slate-200 bg-slate-50 shadow-sm">
              <h4 class="mb-3 text-sm font-semibold text-slate-900">目标任务</h4>
              <div class="flex flex-col gap-3 text-sm">
                <div class="flex items-center justify-between gap-4">
                  <span class="text-slate-500">任务名称</span>
                  <span class="font-medium text-slate-900">{{ selectedDefinition.name }}</span>
                </div>
                <div class="flex items-center justify-between gap-4">
                  <span class="text-slate-500">任务标识</span>
                  <code class="text-xs text-slate-700">{{ selectedDefinition.key }}</code>
                </div>
                <div class="flex items-center justify-between gap-4">
                  <span class="text-slate-500">插件动作</span>
                  <code class="text-xs text-slate-700">{{ selectedDefinition.plugin_key }}:{{ selectedDefinition.action }}</code>
                </div>
                <div class="flex items-center justify-between gap-4">
                  <span class="text-slate-500">调度状态</span>
                  <span class="text-slate-700">{{ selectedDefinition.schedule?.enabled ? "已配置定时调度" : "仅手动/触发器执行" }}</span>
                </div>
              </div>
            </div>

            <div class="rounded-xl border p-4 border-slate-200 bg-slate-50 shadow-sm">
              <h4 class="mb-3 text-sm font-semibold text-slate-900">执行模式</h4>
              <div class="flex flex-col gap-3">
                <div class="flex items-start gap-3 rounded-lg border p-3 border-slate-200 bg-white shadow-sm">
                  <div class="flex flex-col gap-0.5">
                    <div class="text-sm font-semibold text-slate-900">异步 (async)</div>
                    <div class="text-xs leading-5 text-slate-500">请求立即返回受理成功，任务在后台排队执行。适用于耗时较长的任务。</div>
                  </div>
                </div>
                <div class="flex items-start gap-3 rounded-lg border p-3 border-slate-200 bg-white shadow-sm">
                  <div class="flex flex-col gap-0.5">
                    <div class="text-sm font-semibold text-slate-900">同步等待 (sync)</div>
                    <div class="text-xs leading-5 text-slate-500">请求会阻塞直到任务执行完成，并返回任务执行的结果。适用于需要即时反馈的短任务。</div>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </ui-card>
      </template>

      <template #actions>
        <FormActionBar
          :cancel-text="lastToken ? '返回触发器列表' : '取消'"
          submit-text="创建触发器"
          submit-loading-text="创建中…"
          :submit-visible="!lastToken"
          :submit-disabled="!formData.key.trim() || !formData.job_definition_id || !definitions.length"
          :submit-loading="create.loading.value"
          @cancel="router.push(to.triggers.list())"
          @submit="handleCreate"
        />
      </template>
    </FormPageLayout>
  </div>
</template>
