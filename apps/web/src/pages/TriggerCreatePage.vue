<script setup lang="ts">
import { computed, reactive, ref } from "vue";
import { useRouter } from "vue-router";
import { useJobDefinitions } from "@/composables/useJobs";
import { useMessage } from "@/composables";
import { useCreateTrigger } from "@/composables/useTriggers";
import { FormActionBar, FormPageLayout, PageHeader, SmartForm } from "@/components/index";
import type { FieldConfig } from "@/components/smart-form.types";
import { to } from "@/router/registry";

const router = useRouter();
const message = useMessage();
const { data: definitions } = useJobDefinitions();
const create = useCreateTrigger();

const formRef = ref<InstanceType<typeof SmartForm>>();
const formData = ref({
  key: "",
  name: "",
  job_definition_id: "",
  mode: "async",
});
const lastToken = ref("");
const copied = ref(false);

const formFields = computed<FieldConfig[]>(() => [
  {
    name: "key",
    label: "键名",
    type: "text",
    placeholder: "github-webhook",
    required: true,
  },
  {
    name: "name",
    label: "名称",
    type: "text",
    placeholder: "GitHub Webhook",
    required: true,
  },
  {
    name: "job_definition_id",
    label: "绑定任务定义",
    type: "select",
    placeholder: definitions.value.length ? "选择任务定义" : "暂无任务定义",
    required: true,
    options: definitions.value.map((item) => ({
      label: item.name,
      value: String(item.id),
    })),
  },
  {
    name: "mode",
    label: "执行模式",
    type: "select",
    required: true,
    options: [
      { label: "async（异步）", value: "async" },
      { label: "sync（同步等待）", value: "sync" },
    ],
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
    message.error("请检查表单填写是否正确");
    return;
  }
  try {
    const result = await create.execute({
      key: formData.value.key.trim(),
      name: formData.value.name.trim(),
      job_definition_id: Number(formData.value.job_definition_id),
      mode: formData.value.mode,
      default_input: {},
      enabled: true,
    });
    if (result && typeof result === "object" && "delivery_token" in result) {
      lastToken.value = result.delivery_token as string;
    }
    copied.value = false;
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
          :submit-disabled="!formData.key.trim() || !formData.job_definition_id"
          :submit-loading="create.loading.value"
          @cancel="router.push(to.triggers.list())"
          @submit="handleCreate"
        />
      </template>
    </FormPageLayout>
  </div>
</template>
