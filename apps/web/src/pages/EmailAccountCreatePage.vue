<script setup lang="ts">
import { reactive, ref } from "vue";
import { useRouter } from "vue-router";
import { IconEmail, IconPlus } from "@/lib/icons";

import { FormActionBar, FormPageLayout, PageHeader, SmartForm } from "@/components/index";
import { useCreateEmailAccount } from "@/composables/useEmailAccounts";
import { useMessage, useErrorHandler } from "@/composables";
import type { FieldConfig } from "@/components/smart-form.types";
import { to } from "@/router/registry";

const router = useRouter();
const message = useMessage();
const { withErrorHandler } = useErrorHandler();
const create = useCreateEmailAccount();

// 表单引用
const formRef = ref<InstanceType<typeof SmartForm>>();

// 表单数据
const formData = ref({
  address: "",
  provider: "outlook",
  status: "active",
  // Outlook 配置
  tenant: "common",
  redirect_uri: "http://localhost:5173/oauth/callback",
  mailbox: "Inbox",
});

// 表单字段配置
const formFields: FieldConfig[] = [
  {
    name: "address",
    label: "邮箱地址",
    type: "text",
    placeholder: "robot@example.com",
    required: true,
    description: "完整的邮箱地址",
  },
  {
    name: "provider",
    label: "服务商",
    type: "select",
    required: true,
    options: [
      { label: "Outlook", value: "outlook" },
      { label: "Gmail", value: "gmail" },
      { label: "IMAP", value: "imap" },
    ],
    description: "邮箱服务提供商",
  },
  {
    name: "status",
    label: "状态",
    type: "select",
    defaultValue: "active",
    options: [
      { label: "已激活", value: "active" },
      { label: "待验证", value: "pending" },
      { label: "已停用", value: "inactive" },
    ],
  },
  {
    name: "tenant",
    label: "租户ID (Tenant)",
    type: "text",
    placeholder: "common",
    description: "Outlook 租户ID，通常为 'common' 或组织ID",
  },
  {
    name: "redirect_uri",
    label: "回调地址",
    type: "text",
    placeholder: "http://localhost:5173/oauth/callback",
    description: "OAuth 授权完成后的回调地址",
  },
  {
    name: "mailbox",
    label: "邮箱文件夹",
    type: "text",
    placeholder: "Inbox",
    description: "默认监听的邮箱文件夹",
  },
];

// 提交创建
async function handleSubmit() {
  // 验证表单
  const isValid = formRef.value?.validate();
  if (!isValid) {
    message.error("请检查表单填写是否正确");
    return;
  }

  await withErrorHandler(
    async () => {
      await create.execute({
        address: formData.value.address.trim(),
        provider: formData.value.provider,
        status: formData.value.status,
        config: {
          tenant: formData.value.tenant,
          redirect_uri: formData.value.redirect_uri,
          mailbox: formData.value.mailbox,
          scope: [
            "offline_access",
            "openid",
            "profile",
            "email",
            "https://graph.microsoft.com/Mail.Read",
          ],
        },
      });
      message.success("邮箱账号已创建");
      router.push(to.emailAccounts.list());
    },
    { action: "创建邮箱账号", showSuccess: true }
  );
}

// 取消创建
function handleCancel() {
  router.push(to.emailAccounts.list());
}
</script>

<template>
  <div class="page-shell">
    <PageHeader
      title="创建邮箱账号"
      subtitle="创建一个新的邮箱账号"
      icon-bg="linear-gradient(135deg, rgba(234,88,12,0.12), rgba(249,115,22,0.12))"
      icon-color="var(--icon-orange)"
      :back-to="to.emailAccounts.list()"
      back-label="返回邮箱账号列表"
    >
      <template #icon><icon-email /></template>
    </PageHeader>

    <FormPageLayout>
      <template #main>
        <ui-card class="min-w-0">
          <template #title>
            <div class="flex items-center gap-2">
              <icon-email class="h-5 w-5 text-[var(--accent)]" />
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
              <icon-info-circle class="h-5 w-5 text-[var(--accent)]" />
              <span>配置说明</span>
            </div>
          </template>

          <div class="flex flex-col gap-4">
            <h4 class="text-sm font-semibold text-slate-900">支持的邮箱服务商</h4>
            <div class="flex flex-col gap-3">
              <div class="flex items-start gap-3 rounded-xl border p-4 border-slate-200 bg-slate-50 shadow-sm flex-col">
                <div class="flex h-10 w-10 items-center justify-center rounded-xl border text-sm font-bold text-slate-700 border-slate-200 bg-slate-50 shadow-sm">O</div>
                <div>
                  <div class="text-sm font-semibold text-slate-900">Outlook</div>
                  <div class="text-xs leading-5 text-slate-500">Microsoft Outlook / Office 365</div>
                </div>
              </div>
              <div class="flex items-start gap-3 rounded-xl border p-4 border-slate-200 bg-slate-50 shadow-sm flex-col">
                <div class="flex h-10 w-10 items-center justify-center rounded-xl border text-sm font-bold text-slate-700 border-slate-200 bg-slate-50 shadow-sm">G</div>
                <div>
                  <div class="text-sm font-semibold text-slate-900">Gmail</div>
                  <div class="text-xs leading-5 text-slate-500">Google Gmail (即将支持)</div>
                </div>
              </div>
              <div class="flex items-start gap-3 rounded-xl border p-4 border-slate-200 bg-slate-50 shadow-sm flex-col">
                <div class="flex h-10 w-10 items-center justify-center rounded-xl border text-sm font-bold text-slate-700 border-slate-200 bg-slate-50 shadow-sm">I</div>
                <div>
                  <div class="text-sm font-semibold text-slate-900">IMAP</div>
                  <div class="text-xs leading-5 text-slate-500">通用 IMAP 协议 (即将支持)</div>
                </div>
              </div>
            </div>

            <div class="rounded-xl border p-4 border-slate-200 bg-slate-50 shadow-sm">
              <h4 class="mb-3 text-sm font-semibold text-slate-900">配置步骤</h4>
              <ol class="pl-5 text-sm leading-7 text-slate-600 list-decimal">
                <li>填写邮箱地址和选择服务商</li>
                <li>设置初始状态（通常为"待验证"）</li>
                <li>配置 OAuth 回调地址</li>
                <li>保存后进行 OAuth 授权</li>
                <li>授权完成后状态自动更新为"已激活"</li>
              </ol>
            </div>

            <div class="rounded-xl border p-4 border-slate-200 bg-slate-50 shadow-sm">
              <h4 class="mb-3 text-sm font-semibold text-slate-900">注意事项</h4>
              <ul class="pl-5 text-sm leading-7 text-slate-600 list-disc">
                <li>回调地址必须与在邮箱服务商处注册的地址一致</li>
                <li>租户ID (Tenant) 通常使用 "common" 即可</li>
                <li>默认邮箱文件夹为 "Inbox"</li>
                <li>需要先在 Azure AD 中注册应用程序</li>
              </ul>
            </div>
          </div>
        </ui-card>
      </template>

      <template #actions>
        <FormActionBar
          cancel-text="取消"
          submit-text="创建邮箱账号"
          submit-loading-text="创建中…"
          :submit-loading="create.loading.value"
          @cancel="handleCancel"
          @submit="handleSubmit"
        />
      </template>
    </FormPageLayout>
  </div>
</template>
