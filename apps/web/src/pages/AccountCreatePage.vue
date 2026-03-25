<script setup lang="ts">
import { ref, computed, watch } from "vue";
import { useRouter } from "vue-router";
import { IconUser } from "@/lib/icons";
import { useAccountTypes } from "@/composables/useAccountTypes";
import { useCreateAccount } from "@/composables/useAccounts";
import { useMessage } from "@/composables";
import { FormActionBar, FormPageLayout, PageHeader, SmartForm } from "@/components/index";
import { to } from "@/router/registry";
import type { FieldConfig } from "@/components/smart-form.types";
import { Notification } from "@/lib/feedback";

const router = useRouter();
const message = useMessage();
const { data: accountTypes } = useAccountTypes();
const create = useCreateAccount();

const manualFormRef = ref<InstanceType<typeof SmartForm>>();
const formData = ref({
  account_type_id: "",
  identifier: "",
  status: "active",
  tags: "",
});
const spec = ref<Record<string, string>>({});

// Find selected account type
const selectedType = computed(() =>
  accountTypes.value.find((t) => t.id === Number(formData.value.account_type_id)) ?? null
);

// Validate if all required spec fields are filled
const isSpecValid = computed(() => {
  for (const field of specFields.value) {
    if (field.required) {
      const val = spec.value[field.key];
      if (val === undefined || val === null || val === "") return false;
    }
  }
  return true;
});

const isFormValid = computed(() => {
  return !!formData.value.account_type_id && !!formData.value.identifier?.trim() && isSpecValid.value;
});

// Extract spec field definitions from JSON Schema properties
interface SpecField {
  key: string;
  title: string;
  type: string;
  isSecret: boolean;
  defaultValue: string;
  required: boolean;
  choices?: string[];
}

const SECRET_KEYS = new Set(["token", "password", "secret", "access_token", "refresh_token", "client_secret", "api_key"]);

const specFields = computed((): SpecField[] => {
  const schema = selectedType.value?.schema as Record<string, unknown> | undefined;
  if (!schema) return [];
  const props = schema.properties as Record<string, Record<string, unknown>> | undefined;
  if (!props) return [];
  const required = new Set((schema.required as string[]) ?? []);
  return Object.entries(props).map(([key, def]) => ({
    key,
    title: (def.title as string) || key,
    type: (def.type as string) || "string",
    isSecret: SECRET_KEYS.has(key.toLowerCase()) || def.secret === true,
    defaultValue: def.default != null ? String(def.default) : "",
    required: required.has(key),
    choices: Array.isArray(def.enum) ? (def.enum as string[]) : undefined,
  }));
});

// Reset spec when account type changes, seeding defaults
watch(selectedType, () => {
  const next: Record<string, string> = {};
  for (const f of specFields.value) {
    next[f.key] = f.defaultValue;
  }
  spec.value = next;
});

const accountTypeOptions = computed(() =>
  accountTypes.value.map((item) => ({
    label: `${item.name} (${item.key})`,
    value: String(item.id),
  })),
);

const accountTypeFields = computed<FieldConfig[]>(() => [
  {
    name: "account_type_id",
    label: "账号类型",
    type: "select",
    placeholder: accountTypeOptions.value.length ? "选择账号类型" : "暂无账号类型",
    required: true,
    options: accountTypeOptions.value,
  },
]);

const manualFormFields = computed<FieldConfig[]>(() => [
  ...accountTypeFields.value,
  {
    name: "identifier",
    label: "标识符",
    type: "text",
    placeholder: "唯一 ID 或用户名",
    required: true,
  },
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

async function handleCreate() {
  const isValid = manualFormRef.value?.validate();
  if (!isValid) {
    message.error("请检查表单填写是否正确");
    return;
  }
  // Build spec — omit empty-string optional fields
  const builtSpec: Record<string, unknown> = {};
  for (const f of specFields.value) {
    const val = spec.value[f.key] ?? "";
    if (val !== "" || f.required) {
      builtSpec[f.key] = f.type === "integer" ? Number(val) : val;
    }
  }

  try {
    await create.execute({
      account_type_id: Number(formData.value.account_type_id),
      identifier: formData.value.identifier.trim(),
      status: formData.value.status,
      tags: formData.value.tags.split(",").map((t) => t.trim()).filter(Boolean),
      spec: builtSpec,
    });
    message.success("账号已创建");
    Notification.info({ title: "下一步", content: "创建一个任务来对该账号执行自动化操作", duration: 6000 });
    router.push(to.accounts.list());
  } catch (e) {
    message.error(e instanceof Error ? e.message : "创建失败");
  }
}
</script>

<template>
  <div class="page-shell">
    <PageHeader
      title="创建账号"
      subtitle="手动配置并创建一个新的账号"
      icon-bg="var(--accent-light)"
      icon-color="var(--accent)"
      :back-to="to.accounts.list()"
      back-label="返回账号列表"
    >
      <template #icon><icon-user /></template>
    </PageHeader>

    <FormPageLayout>
      <template #main>
        <ui-card class="min-w-0">
          <template #title>
            <div class="flex items-center gap-2">
              <icon-user class="h-5 w-5 text-[var(--accent)]" />
              <span>基本信息</span>
            </div>
          </template>
          <SmartForm
            ref="manualFormRef"
            v-model="formData"
            :fields="manualFormFields"
          />

          <ui-form v-if="specFields.length" layout="vertical" class="mt-4">
            <ui-divider orientation="left" class="my-4">
              <span class="text-xs font-semibold uppercase tracking-wide text-slate-500">凭据 / Spec</span>
            </ui-divider>

            <ui-form-item
              v-for="field in specFields"
              :key="field.key"
              :label="field.title"
            >
              <template #label>
                <span>{{ field.title }}</span>
                <ui-tag v-if="field.required" size="small" color="red" class="ml-1">必填</ui-tag>
                <code class="ml-1 text-xs font-mono text-slate-400">{{ field.key }}</code>
              </template>

              <ui-select
                v-if="field.choices?.length"
                v-model="spec[field.key]"
                allow-clear
                :placeholder="field.required ? '必填' : '可选'"
              >
                <ui-option v-for="c in field.choices" :key="c" :value="c">{{ c }}</ui-option>
              </ui-select>
              <ui-input-number
                v-else-if="field.type === 'integer'"
                v-model="(spec[field.key] as unknown as number)"
                :placeholder="field.defaultValue || field.key"
                class="w-full"
              />
              <ui-input
                v-else-if="field.isSecret"
                v-model="spec[field.key]"
                type="password"
                allow-clear
                :placeholder="field.required ? '必填' : '可选'"
              />
              <ui-input
                v-else
                v-model="spec[field.key]"
                allow-clear
                :placeholder="field.defaultValue || (field.required ? '必填' : '可选')"
              />
            </ui-form-item>
          </ui-form>
        </ui-card>
      </template>

      <template #aside>
        <ui-card class="min-w-0 lg:sticky lg:top-[var(--space-6)]">
          <template #title>
            <div class="flex items-center gap-2">
              <icon-info-circle class="h-5 w-5 text-[var(--accent)]" />
              <span>关于账号</span>
            </div>
          </template>
          <div class="flex flex-col gap-4">
            <div class="rounded-xl border p-4 border-slate-200 bg-slate-50 shadow-sm">
              <p class="text-sm leading-6 text-slate-500">
                账号是用于连接第三方服务或系统的凭据。创建账号后，你可以将其分配给 Agent 使用。
              </p>
            </div>
            <div class="rounded-xl border p-4 border-slate-200 bg-slate-50 shadow-sm">
              <h4 class="mb-3 text-sm font-semibold text-slate-900">注意事项</h4>
              <ul class="pl-5 text-sm leading-7 text-slate-600 list-disc">
                <li>不同的账号类型需要填写不同的凭据信息。</li>
                <li>标识符是账号的唯一标识，用于区分不同的账号。</li>
                <li>凭据信息将加密存储。</li>
              </ul>
            </div>
          </div>
        </ui-card>
      </template>

      <template #actions>
        <FormActionBar
          cancel-text="取消"
          submit-text="创建账号"
          submit-loading-text="创建中…"
          :submit-disabled="!isFormValid"
          :submit-loading="create.loading.value"
          @cancel="router.push(to.accounts.list())"
          @submit="handleCreate"
        />
      </template>
    </FormPageLayout>
  </div>
</template>
