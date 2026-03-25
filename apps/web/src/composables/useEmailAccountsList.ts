import { computed, ref, watch } from "vue";
import { useRouter } from "vue-router";
import { useBulkImportEmailAccounts, useDeleteEmailAccount, useEmailAccounts } from "@/composables/useEmailAccounts";
import { useConfirm, useErrorHandler, useMessage } from "@/composables";
import { normalizeListResponse } from "@/utils/normalizeListResponse";
import type { EmailAccount, EmailBulkImportLineResult } from "@/types";
import { to } from "@/router/registry";

export function useEmailAccountsList() {
  const router = useRouter();
  const message = useMessage();
  const confirm = useConfirm();
  const { withErrorHandler } = useErrorHandler();

  const { data: accounts, loading, refresh, error } = useEmailAccounts();
  const deleteAccountAction = useDeleteEmailAccount();
  const bulkImport = useBulkImportEmailAccounts();
  const bulkImportLoading = bulkImport.loading;

  watch(error, (err) => {
    if (err) {
      message.error(`获取邮箱账号失败: ${err}`);
    }
  });

  const statusFilter = ref<string>("");
  const providerFilter = ref<string>("");
  const searchKeyword = ref("");

  const filteredAccounts = computed(() => {
    let result = accounts.value;

    if (statusFilter.value) {
      result = result.filter((item) => item.status === statusFilter.value);
    }

    if (providerFilter.value) {
      result = result.filter((item) => item.provider === providerFilter.value);
    }

    if (searchKeyword.value) {
      const keyword = searchKeyword.value.toLowerCase();
      result = result.filter((item) => item.address.toLowerCase().includes(keyword));
    }

    return result;
  });

  const providers = computed(() => {
    const uniqueProviders = new Set(accounts.value.map((item) => item.provider));
    return Array.from(uniqueProviders).sort();
  });

  const showBulkImport = ref(false);
  const bulkText = ref("");
  const bulkResult = ref<EmailBulkImportLineResult[] | null>(null);

  function openBulkImport() {
    bulkText.value = "";
    bulkResult.value = null;
    showBulkImport.value = true;
  }

  async function handleBulkImport() {
    const lines = bulkText.value.split("\n").map((l) => l.trim()).filter(Boolean);
    if (!lines.length) {
      message.warning("请输入至少一行数据");
      return;
    }

    await withErrorHandler(
      async () => {
        const res = await bulkImport.execute(lines);
        bulkResult.value = normalizeListResponse<EmailBulkImportLineResult>(res);
        message.success(`导入完成：成功 ${res.success}，失败 ${res.failed}`);
        if (res.success > 0) await refresh();
      },
      { action: "批量导入" },
    );
  }

  async function handleQuickAction(key: string, account: EmailAccount) {
    switch (key) {
      case "view":
        router.push(to.emailAccounts.preview(account.id));
        break;
      case "edit":
        router.push(to.emailAccounts.edit(account.id));
        break;
      case "delete":
        await deleteAccount(account);
        break;
    }
  }

  async function deleteAccount(account: EmailAccount) {
    const confirmed = await confirm.confirm(
      `删除后，依赖此邮箱的任务可能会失败。确定删除邮箱 "${account.address}" 吗？`,
    );
    if (!confirmed) return;

  await withErrorHandler(
    async () => {
      await deleteAccountAction.execute(account.id);
      message.success("已删除邮箱账号");
      await refresh();
    },
    { action: "删除", showSuccess: false },
  );
  }

  async function handleBatchDelete(items: EmailAccount[]) {
    const confirmed = await confirm.confirm(`确定要删除选中的 ${items.length} 个邮箱账号吗？`);
    if (!confirmed) return;

  await withErrorHandler(
    async () => {
      await Promise.all(items.map((item) => deleteAccountAction.execute(item.id)));
      message.success(`已删除 ${items.length} 个邮箱账号`);
      await refresh();
    },
    { action: "批量删除", showSuccess: false },
  );
}

  async function handleBatchExport(items: EmailAccount[]) {
    const data = JSON.stringify(items, null, 2);
    const blob = new Blob([data], { type: "application/json" });
    const url = URL.createObjectURL(blob);
    const link = document.createElement("a");
    link.href = url;
    link.download = `email-accounts-${Date.now()}.json`;
    link.click();
    URL.revokeObjectURL(url);
    message.success(`已导出 ${items.length} 个邮箱账号`);
  }

  return {
    accounts,
    loading,
    refresh,
    filteredAccounts,
    providers,
    statusFilter,
    providerFilter,
    searchKeyword,
    showBulkImport,
    bulkText,
    bulkResult,
    bulkImportLoading,
    openBulkImport,
    handleBulkImport,
    handleQuickAction,
    handleBatchDelete,
    handleBatchExport,
  };
}
