<script setup lang="ts">
import { computed, reactive, ref } from "vue";
import { useRouter } from "vue-router";
import { useCreateAccountType } from "@/composables/useAccountTypes";
import { useMessage } from "@/composables";
import { FormActionBar, FormPageLayout, PageHeader, SmartForm } from "@/components/index";
import type { FieldConfig } from "@/components/smart-form.types";
import { to } from "@/router/registry";
import { Notification } from "@/lib/feedback";

const router = useRouter();
const message = useMessage();
const create = useCreateAccountType();

const formRef = ref<InstanceType<typeof SmartForm>>();
const formData = ref({
  key: "",
  name: "",
  category: "generic",
});

const formFields = computed<FieldConfig[]>(() => [
  {
    name: "key",
    label: "键名",
    type: "text",
    placeholder: "github",
    required: true,
  },
  {
    name: "name",
    label: "显示名称",
    type: "text",
    placeholder: "GitHub",
    required: true,
  },
  {
    name: "category",
    label: "分类",
    type: "select",
    required: true,
    options: [
      { label: "generic", value: "generic" },
      { label: "email", value: "email" },
      { label: "system", value: "system" },
    ],
  },
]);

async function handleCreate() {
  const isValid = formRef.value?.validate();
  if (!isValid) {
    return;
  }
  try {
    await create.execute({
      key: formData.value.key.trim(),
      name: formData.value.name.trim(),
      category: formData.value.category,
      schema: {},
      capabilities: {},
    });
    message.success("账号类型已创建");
    Notification.info({ title: "下一步", content: "为该账号类型添加账号，录入真实的凭证信息", duration: 6000 });
    router.push(to.accountTypes.list());
  } catch (e) {
    message.error(e instanceof Error ? e.message : "创建失败");
  }
}
</script>

<template>
  <div class="page-shell">
    <PageHeader
      title="创建账号类型"
      subtitle="创建一个新的账号类型"
      icon-bg="linear-gradient(135deg, rgba(2,132,199,0.12), rgba(14,165,233,0.12))"
      icon-color="#0284c7"
      :back-to="to.accountTypes.list()"
      back-label="返回账号类型列表"
    >
      <template #icon><icon-layers /></template>
    </PageHeader>

    <FormPageLayout>
      <template #main>
        <ui-card class="min-w-0">
          <template #title>
            <div class="flex items-center gap-2">
              <icon-layers class="h-5 w-5 text-[#0284c7]" />
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
              <icon-info-circle class="h-5 w-5 text-[#0284c7]" />
              <span>关于账号类型</span>
            </div>
          </template>
          <div class="flex flex-col gap-4">
            <div class="rounded-xl border p-4 border-slate-200 bg-slate-50 shadow-sm">
              <p class="text-sm leading-6 text-slate-500">
                账号类型定义了一组凭据规范，允许系统与不同的外部服务建立连接。
              </p>
            </div>
            <div class="rounded-xl border p-4 border-slate-200 bg-slate-50 shadow-sm">
              <h4 class="mb-3 text-sm font-semibold text-slate-900">注意事项</h4>
              <ul class="pl-5 text-sm leading-7 text-slate-600 list-disc">
                <li>键名 (Key) 必须唯一，且创建后无法修改。</li>
                <li>通常由插件自动注册，手动创建主要用于测试。</li>
              </ul>
            </div>
          </div>
        </ui-card>
      </template>

      <template #actions>
        <FormActionBar
          cancel-text="取消"
          submit-text="创建账号类型"
          submit-loading-text="创建中…"
          :submit-disabled="!formData.key.trim() || !formData.name.trim()"
          :submit-loading="create.loading.value"
          @cancel="router.push(to.accountTypes.list())"
          @submit="handleCreate"
        />
      </template>
    </FormPageLayout>
  </div>
</template>
