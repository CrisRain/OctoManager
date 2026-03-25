<script setup lang="ts">
import { ref, reactive, computed, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import { IconEdit } from "@/lib/icons";

import { FormActionBar, FormPageLayout, PageHeader, SmartForm } from "@/components/index";
import { useAccountTypes, usePatchAccountType } from "@/composables/useAccountTypes";
import { useMessage, useErrorHandler } from "@/composables";
import type { FieldConfig } from "@/components/smart-form.types";
import { to } from "@/router/registry";

const route = useRoute();
const router = useRouter();
const typeKey = route.params.id as string;

const message = useMessage();
const { withErrorHandler } = useErrorHandler();
const { data: accountTypes, loading } = useAccountTypes();
const patch = usePatchAccountType();

const accountType = computed(() => accountTypes.value.find((t) => t.key === typeKey));

const formRef = ref<InstanceType<typeof SmartForm>>();

const formData = ref({
  name: "",
  category: "",
});

watch(accountType, (t) => {
  if (!t) return;
  formData.value.name = t.name;
  formData.value.category = t.category;
}, { immediate: true });

const formFields: FieldConfig[] = [
  {
    name: "name",
    label: "账号类型名称",
    type: "text",
    placeholder: "例如: GitHub",
    required: true,
    description: "账号类型的显示名称",
  },
  {
    name: "category",
    label: "分类",
    type: "text",
    placeholder: "例如: vcs",
    description: "账号类型的分类标识",
  },
];

async function handleSave() {
  const isValid = formRef.value?.validate();
  if (!isValid) {
    message.error("请检查表单填写是否正确");
    return;
  }

  await withErrorHandler(
    async () => {
      await patch.execute(typeKey, {
        name: formData.value.name.trim(),
        category: formData.value.category.trim(),
      });
      message.success("账号类型已更新");
      router.push(to.accountTypes.list());
    },
    { action: "更新账号类型", showSuccess: false }
  );
}
</script>

<template>
  <div class="page-shell">
    <PageHeader
      title="编辑账号类型"
      :subtitle="accountType ? `正在编辑 ${accountType.name}` : '账号类型详情加载中…'"
      icon-bg="linear-gradient(135deg, rgba(2,132,199,0.12), rgba(14,165,233,0.12))"
      icon-color="#0284c7"
      :back-to="to.accountTypes.list()"
      back-label="返回账号类型列表"
    >
      <template #icon><icon-layers /></template>
    </PageHeader>

    <FormPageLayout
      :loading="loading"
      :ready="!!accountType"
      empty-description="未找到该账号类型"
    >
      <template #empty-action>
        <ui-button type="primary" @click="router.push(to.accountTypes.list())">返回账号类型列表</ui-button>
      </template>

      <template #main>
        <ui-card class="min-w-0">
          <template #title>
            <div class="flex items-center gap-2">
              <icon-edit class="h-4 w-4 text-[var(--accent)]" />
              <span>编辑账号类型信息</span>
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
              <span class="text-xs font-semibold tracking-wider text-slate-500">账号类型标识符</span>
              <code class="text-sm font-medium text-slate-700">{{ accountType?.key }}</code>
            </div>
            <div v-if="accountType?.capabilities && Object.keys(accountType.capabilities).length > 0" class="flex flex-col gap-2 rounded-xl border p-4 border-slate-200 bg-slate-50 shadow-sm">
              <span class="text-xs font-semibold tracking-wider text-slate-500">能力列表</span>
              <div class="mt-1 flex flex-wrap gap-2">
                <ui-tag v-for="cap in Object.keys(accountType.capabilities)" :key="cap">{{ cap }}</ui-tag>
              </div>
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
          @cancel="router.push(to.accountTypes.list())"
          @submit="handleSave"
        />
      </template>
    </FormPageLayout>
  </div>
</template>
