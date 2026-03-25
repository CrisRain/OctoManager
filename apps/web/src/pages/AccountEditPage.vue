<script setup lang="ts">
import { ref, computed, watch, reactive } from "vue";
import { useRoute, useRouter } from "vue-router";
import { useAccounts, usePatchAccount } from "@/composables/useAccounts";
import { useMessage } from "@/composables";
import { useAccountTypes } from "@/composables/useAccountTypes";
import { FormActionBar, FormPageLayout, PageHeader, SmartForm } from "@/components/index";
import { to } from "@/router/registry";
import type { FieldConfig } from "@/components/smart-form.types";

const route = useRoute();
const router = useRouter();
const accountId = Number(route.params.id);
const message = useMessage();

const { data: accounts, loading, refresh } = useAccounts();
const { data: accountTypes } = useAccountTypes();
const patch = usePatchAccount();

const account = computed(() => accounts.value.find((a) => a.id === accountId) ?? null);

// Find matching account type for spec fields
const accountType = computed(() =>
  accountTypes.value.find((t) => t.key === account.value?.account_type_key) ?? null
);

// ── Form state ──────────────────────────────────────────────────────────────
const formRef = ref<InstanceType<typeof SmartForm>>();
const formData = ref({
  status: "active",
  tags: "",
});
const spec = ref<Record<string, string>>({});

const SECRET_KEYS = new Set(["token", "password", "secret", "access_token", "refresh_token", "client_secret", "api_key"]);

interface SpecField {
  key: string;
  title: string;
  type: string;
  isSecret: boolean;
  defaultValue: string;
  required: boolean;
}

const specFields = computed((): SpecField[] => {
  const schema = accountType.value?.schema as Record<string, unknown> | undefined;
  if (!schema) return [];
  const props = schema.properties as Record<string, Record<string, unknown>> | undefined;
  if (!props) return [];
  const required = new Set((schema.required as string[]) ?? []);
  return Object.entries(props).map(([key, def]) => ({
    key,
    title: (def.title as string) || key,
    type: (def.type as string) || "string",
    isSecret: SECRET_KEYS.has(key.toLowerCase()),
    defaultValue: def.default != null ? String(def.default) : "",
    required: required.has(key),
  }));
});

// Seed form when account loads
watch(account, (acc) => {
  if (!acc) return;
  formData.value.status = acc.status;
  formData.value.tags = (acc.tags ?? []).join(", ");
}, { immediate: true });

watch([account, specFields], ([acc, fields]) => {
  if (!acc || !fields.length) return;
  const next: Record<string, string> = {};
  for (const f of fields) {
    const existing = acc.spec?.[f.key];
    next[f.key] = existing != null ? String(existing) : f.defaultValue;
  }
  spec.value = next;
}, { immediate: true });

async function handleSave() {
  const isValid = formRef.value?.validate();
  if (!isValid) {
    message.error("请检查表单填写是否正确");
    return;
  }

  const builtSpec: Record<string, unknown> = {};
  for (const f of specFields.value) {
    const val = spec.value[f.key] ?? "";
    if (val !== "" || f.required) {
      builtSpec[f.key] = f.type === "integer" ? Number(val) : val;
    }
  }
  // If no schema fields, pass existing spec unchanged
  const specPayload = specFields.value.length > 0 ? builtSpec : account.value?.spec ?? {};

  try {
    await patch.execute(accountId, {
      status: formData.value.status,
      tags: formData.value.tags.split(",").map((t) => t.trim()).filter(Boolean),
      spec: specPayload,
    });
    await refresh();
    message.success("账号已更新");
    router.push(to.accounts.detail(accountId));
  } catch (e) {
    message.error(e instanceof Error ? e.message : "保存失败");
  }
}

const formFields = computed<FieldConfig[]>(() => [
  {
    name: "status",
    label: "状态",
    type: "select",
    required: true,
    options: [
      { label: "已激活", value: "active" },
      { label: "待验证", value: "pending" },
      { label: "已停用", value: "inactive" },
    ],
  },
  {
    name: "tags",
    label: "标签",
    type: "text",
    placeholder: "用逗号分隔，如 demo, test",
  },
]);
</script>

<template>
  <div class="page-shell">
    <PageHeader
      title="编辑账号"
      :subtitle="account ? `正在编辑 ${account.identifier}` : '账号详情加载中…'"
      icon-bg="var(--accent-light)"
      icon-color="var(--accent)"
      :back-to="to.accounts.detail(accountId)"
      back-label="返回账号详情"
    >
      <template #icon><icon-user /></template>
    </PageHeader>

    <FormPageLayout
      :loading="loading"
      :ready="!!account"
      empty-description="未找到该账号"
    >
      <template #empty-action>
        <ui-button type="primary" @click="router.push(to.accounts.list())">返回账号列表</ui-button>
      </template>

      <template #main>
        <ui-card class="min-w-0">
          <template #title>
            <div class="flex items-center gap-2">
              <icon-edit class="h-4 w-4 text-[var(--accent)]" />
              <span>编辑账号信息</span>
            </div>
          </template>
          <SmartForm
            ref="formRef"
            v-model="formData"
            :fields="formFields"
          />

          <ui-form layout="vertical" class="mt-4">
            <template v-if="specFields.length">
              <ui-divider orientation="left" class="my-4">
                <span class="text-xs font-semibold uppercase tracking-wide text-slate-500">凭据 / Spec</span>
              </ui-divider>

              <ui-form-item
                v-for="field in specFields"
                :key="field.key"
              >
                <template #label>
                  <span>{{ field.title }}</span>
                  <ui-tag v-if="field.required" size="small" color="red" class="ml-1">必填</ui-tag>
                  <code class="ml-1 text-xs font-mono text-slate-400">{{ field.key }}</code>
                </template>

                <ui-input-number
                  v-if="field.type === 'integer'"
                  v-model="(spec[field.key] as unknown as number)"
                  :placeholder="field.defaultValue || field.key"
                  class="w-full"
                />
                <ui-input
                  v-else-if="field.isSecret"
                  v-model="spec[field.key]"
                  type="password"
                  allow-clear
                  :placeholder="field.required ? '必填' : '可选（留空保持不变）'"
                />
                <ui-input
                  v-else
                  v-model="spec[field.key]"
                  allow-clear
                  :placeholder="field.defaultValue || (field.required ? '必填' : '可选')"
                />
              </ui-form-item>
            </template>

            <template v-else>
              <ui-divider orientation="left" class="my-4">
                <span class="text-xs font-semibold uppercase tracking-wide text-slate-500">凭据 / Spec（JSON）</span>
              </ui-divider>
              <ui-form-item label="Spec（原始 JSON）">
                <ui-textarea
                  :model-value="JSON.stringify(account?.spec, null, 2)"
                  disabled
                  :auto-size="{ minRows: 3, maxRows: 10 }"
                  class="font-mono text-sm"
                />
                <p class="text-sm leading-6 text-slate-500">该账号类型未找到 Schema，Spec 不可编辑。</p>
              </ui-form-item>
            </template>
          </ui-form>
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
              <span class="text-xs font-semibold tracking-wider text-slate-500">账号类型</span>
              <code class="text-sm font-medium text-slate-700">{{ account?.account_type_key || "—" }}</code>
            </div>
            <div class="flex flex-col gap-2 rounded-xl border p-4 border-slate-200 bg-slate-50 shadow-sm">
              <span class="text-xs font-semibold tracking-wider text-slate-500">标识符</span>
              <span class="text-sm font-medium text-slate-700">{{ account?.identifier }}</span>
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
          @cancel="router.push(to.accounts.detail(accountId))"
          @submit="handleSave"
        />
      </template>
    </FormPageLayout>
  </div>
</template>
