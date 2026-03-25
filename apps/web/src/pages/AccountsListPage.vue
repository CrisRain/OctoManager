<script setup lang="ts">
/**
 * 账号列表页 - UX优化版本
 * 集成SmartListBar、RowActionsMenu
 */
import { computed, ref, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import { IconUser, IconCheck, IconSync, IconStop, IconPlayArrow } from "@/lib/icons";

import { useAccountTypes } from "@/composables/useAccountTypes";
import { useAccounts, usePatchAccount, useDeleteAccount } from "@/composables/useAccounts";
import { useMessage, useConfirm, useErrorHandler } from "@/composables";
import { to } from "@/router/registry";

const route = useRoute();
const router = useRouter();
const message = useMessage();
const confirm = useConfirm();
const { withErrorHandler } = useErrorHandler();

// 数据加载
const { data: types, loading: loadingTypes } = useAccountTypes();
const { data: accounts, loading: loadingAccounts, error: accountsError, refresh } = useAccounts();
watch(accounts, () => { selectedKeys.value = []; });
const patchAccount = usePatchAccount();
const deleteAccountOp = useDeleteAccount();

const selectedKeys = ref<string[]>([]);

// 类型筛选
const typeKey = computed(() => route.params.typeKey as string | undefined);
const statusFilter = ref("");
const searchKeyword = ref("");
const statusOptions = [
  { label: "全部", value: "" },
  { label: "活跃", value: "active" },
  { label: "停用", value: "inactive" },
  { label: "待定", value: "pending" },
];

function normalizeFilterValue(value: string | undefined) {
  return value?.trim().toLowerCase() ?? "";
}

const normalizedTypeKey = computed(() => normalizeFilterValue(typeKey.value));

// 获取当前类型信息
const currentType = computed(() => {
  if (!normalizedTypeKey.value) return null;
  return types.value.find(t => normalizeFilterValue(t.key) === normalizedTypeKey.value);
});

const typeKeyById = computed(() => {
  const map = new Map<number, string>();
  for (const type of types.value) {
    map.set(type.id, type.key);
  }
  return map;
});

function resolveAccountTypeKey(item: (typeof accounts.value)[number]) {
  if (item.account_type_key) {
    return normalizeFilterValue(item.account_type_key);
  }
  if (typeof item.account_type_id === "number") {
    return normalizeFilterValue(typeKeyById.value.get(item.account_type_id));
  }
  return "";
}

// 过滤账号
const filteredAccounts = computed(() => {
  let result = accounts.value;

  // 按类型筛选
  if (normalizedTypeKey.value) {
    result = result.filter(item => resolveAccountTypeKey(item) === normalizedTypeKey.value);
  }

  // 按状态筛选
  if (statusFilter.value) {
    result = result.filter(item => normalizeFilterValue(item.status) === statusFilter.value);
  }

  // 搜索过滤
  if (searchKeyword.value) {
    const keyword = searchKeyword.value.toLowerCase();
    result = result.filter(item =>
      item.identifier.toLowerCase().includes(keyword) ||
      item.tags?.some((tag: string) => tag.toLowerCase().includes(keyword))
    );
  }

  return result;
});

// 状态配置
const statusConfig: Record<string, { label: string; color: string; icon?: any }> = {
  active: { label: "活跃", color: "green", icon: IconCheck },
  inactive: { label: "停用", color: "gray", icon: IconStop },
  pending: { label: "待定", color: "orange", icon: IconSync },
};

// 快速操作
async function handleQuickAction(key: string, account: any) {
  switch (key) {
    case "view":
      router.push(to.accounts.detail(account.id));
      break;
    case "edit":
      router.push(to.accounts.edit(account.id));
      break;
    case "toggle":
      await toggleAccountStatus(account);
      break;
    case "delete":
      await deleteAccount(account);
      break;
    case "copy":
      await copyToClipboard(String(account.id));
      break;
  }
}

// 切换账号状态
async function toggleAccountStatus(account: any) {
  const newStatus = account.status === "active" ? "inactive" : "active";
  const action = newStatus === "active" ? "启用" : "停用";

  const confirmed = await confirm.confirm(
    `确定要${action}账号 "${account.identifier}" 吗？`
  );

  if (!confirmed) return;

  await withErrorHandler(
    async () => {
      await patchAccount.execute(account.id, { status: newStatus });
      await refresh();
    },
    { action, showSuccess: true }
  );
}

// 删除账号
async function deleteAccount(account: any) {
  const confirmed = await confirm.confirmDelete(account.identifier);
  if (!confirmed) return;

  await withErrorHandler(
    async () => {
      await deleteAccountOp.execute(account.id);
      message.success(`已删除账号: ${account.identifier}`);
      await refresh();
    },
    { action: "删除", showSuccess: false }
  );
}

// 复制到剪贴板
async function copyToClipboard(text: string) {
  try {
    await navigator.clipboard.writeText(text);
    message.success("已复制到剪贴板");
  } catch {
    message.error("复制失败");
  }
}

// 批量操作
async function handleBatchDelete(items: any[]) {
  const confirmed = await confirm.confirm(`确定要删除选中的 ${items.length} 个账号吗？`);
  if (!confirmed) return;

  await withErrorHandler(
    async () => {
      await Promise.all(items.map((item) => deleteAccountOp.execute(item.id)));
      message.success(`已删除 ${items.length} 个账号`);
      await refresh();
    },
    { action: "批量删除", showSuccess: false }
  );
}

// 批量导出
async function handleBatchExport(items: any[]) {
  const data = JSON.stringify(items, null, 2);
  const blob = new Blob([data], { type: "application/json" });
  const url = URL.createObjectURL(blob);
  const link = document.createElement("a");
  link.href = url;
  link.download = `accounts-${Date.now()}.json`;
  link.click();
  URL.revokeObjectURL(url);
  message.success(`已导出 ${items.length} 个账号`);
}

// 批量启用/停用
async function handleBatchToggle(selectedItems: any[], enable: boolean) {
  const action = enable ? "启用" : "停用";
  const confirmed = await confirm.confirm(`确定要${action}选中的 ${selectedItems.length} 个账号吗？`);
  if (!confirmed) return;

  await withErrorHandler(
    async () => {
      const newStatus = enable ? "active" : "inactive";
      await Promise.all(selectedItems.map((item) => patchAccount.execute(item.id, { status: newStatus })));
      await refresh();
    },
    { action: `批量${action}`, showSuccess: true }
  );
}
</script>

<template>
  <div class="page-shell accounts-list-page">
    <PageHeader
      :title="currentType?.name || '账号管理'"
      :subtitle="((currentType?.schema as Record<string, unknown> | undefined)?.description as string | undefined) || '集中管理各类型账号，支持按类型筛选和批量操作'"
      icon-bg="var(--accent-light)"
      icon-color="var(--accent)"
    >
      <template #icon><icon-user /></template>
      <template #actions>
        <ui-button type="primary" @click="router.push(to.accounts.create())">
          <template #icon><icon-plus /></template>
          新增账号
        </ui-button>
      </template>
    </PageHeader>

    <!-- 智能工具栏 -->
    <SmartListBar
      :data="filteredAccounts"
      :loading="loadingAccounts || loadingTypes"
      v-model:search="searchKeyword"
      v-model:selectedKeys="selectedKeys"
      @refresh="refresh"
      @batch-delete="handleBatchDelete"
      @batch-export="handleBatchExport"
    >
      <template #filters>
        <!-- 类型筛选 -->
        <div class="flex flex-wrap items-center gap-3">
          <span class="text-sm font-medium text-slate-600">类型：</span>
          <div class="flex flex-wrap items-center gap-2">
            <button type="button"
              class="filter-chip"
              :class="{ active: !typeKey }"
              @click="router.push(to.accounts.list())"
            >
              全部
            </button>
            <button type="button"
              v-for="type in types?.filter(t => t.category === 'generic')"
              :key="type.key"
              class="filter-chip"
              :class="{ active: normalizedTypeKey === normalizeFilterValue(type.key) }"
              @click="router.push(to.accounts.byType(type.key))"
            >
              {{ type.name }}
            </button>
          </div>
        </div>

        <!-- 状态筛选 -->
        <div class="flex flex-wrap items-center gap-3 mt-3">
          <span class="text-sm font-medium text-slate-600">状态：</span>
          <div class="flex flex-wrap items-center gap-2">
            <button type="button"
              v-for="option in statusOptions"
              :key="option.value"
              class="filter-chip"
              :class="{ active: statusFilter === option.value }"
              @click="statusFilter = option.value"
            >
              {{ option.label }}
            </button>
          </div>
        </div>
      </template>

      <template #extra-actions>
        <ui-button @click="router.push(to.accountTypes.list())">
          管理类型
        </ui-button>
      </template>
    </SmartListBar>

    <!-- 数据表格 -->
    <ui-card class="mb-4 hidden lg:block">
      <ui-table
        :data="filteredAccounts"
        :loading="loadingAccounts || loadingTypes"
        :pagination="{
          showTotal: true,
          pageSizeOptions: [10, 20, 50],
          defaultPageSize: 20,
        }"
        :bordered="false"
        row-key="id"
        :row-selection="{ type: 'checkbox' }"
        v-model:selectedKeys="selectedKeys"
      >
        <template #columns>
          <!-- 账号标识 -->
          <ui-table-column title="账号标识" data-index="identifier">
            <template #cell="{ record }">
              <div class="flex items-center gap-3">
                <div class="flex h-9 w-9 flex-shrink-0 items-center justify-center rounded-lg border border-blue-200 bg-blue-50 text-sm text-blue-600 shadow-sm">
                  <icon-user />
                </div>
                <div class="flex min-w-0 flex-col gap-0.5">
                  <div class="truncate text-[14px] font-medium text-slate-900">{{ record.identifier }}</div>
                  <code v-if="record.id" class="text-xs text-slate-500 mono">#{{ record.id }}</code>
                </div>
              </div>
            </template>
          </ui-table-column>

          <!-- 类型 -->
          <ui-table-column title="类型" data-index="account_type_key">
            <template #cell="{ record }">
              <code v-if="resolveAccountTypeKey(record)" class="inline-flex items-center rounded border border-slate-200 bg-slate-100 px-1.5 py-0.5 text-xs font-mono text-slate-600">
                {{ resolveAccountTypeKey(record) }}
              </code>
              <span v-else class="text-sm text-slate-400">—</span>
            </template>
          </ui-table-column>

          <!-- 状态 -->
          <ui-table-column title="状态" data-index="status">
            <template #cell="{ record }">
              <StatusTag :status="record.status" />
            </template>
          </ui-table-column>

          <!-- 标签 -->
          <ui-table-column title="标签" data-index="tags">
            <template #cell="{ record }">
              <div v-if="record.tags?.length" class="flex flex-wrap items-center gap-1">
                <ui-tag v-for="tag in record.tags.slice(0, 3)" :key="tag" size="small">
                  {{ tag }}
                </ui-tag>
                <ui-tag v-if="record.tags.length > 3" size="small" color="gray">
                  +{{ record.tags.length - 3 }}
                </ui-tag>
              </div>
              <span v-else class="text-sm text-slate-400">—</span>
            </template>
          </ui-table-column>

          <!-- 快速操作 -->
          <ui-table-column title="操作" align="right">
            <template #cell="{ record }">
              <RowActionsMenu
                :item="record"
                :actions="[
                  { key: 'view', label: '查看详情', icon: 'IconEye' },
                  { key: 'edit', label: '编辑', icon: 'IconEdit' },
                  { key: 'toggle', label: record.status === 'active' ? '停用' : '启用', icon: record.status === 'active' ? 'IconStop' : 'IconPlayArrow' },
                  { key: 'copy', label: '复制ID', icon: 'IconCopy' },
                  { key: 'delete-divider', divider: true },
                  { key: 'delete', label: '删除', icon: 'IconDelete', danger: true },
                ]"
                @action="handleQuickAction"
              />
            </template>
          </ui-table-column>
        </template>

        <!-- 空状态 -->
        <template #empty>
          <ui-empty
            :description="accountsError ? `加载失败: ${accountsError}` : (typeKey ? `暂无 [${typeKey}] 类型的账号` : '暂无账号')"
          >
            <ui-button v-if="accountsError" @click="refresh">重新加载</ui-button>
            <ui-button v-else type="primary" @click="router.push(to.accounts.create())">
              创建第一个账号
            </ui-button>
          </ui-empty>
        </template>
      </ui-table>
    </ui-card>

    <div class="flex flex-col gap-3 lg:hidden">
      <ui-card
        v-for="record in filteredAccounts"
        :key="record.id"
        v-memo="[record.id, record.identifier, record.status, record.account_type_key, record.account_type_id, record.tags?.length]"
        class="rounded-xl border p-5 border-slate-200 bg-white shadow-sm"
      >
        <div class="mb-3 flex items-center gap-3">
          <div class="flex h-9 w-9 flex-shrink-0 items-center justify-center rounded-lg border border-blue-200 bg-blue-50 text-sm text-blue-600 shadow-sm">
            <icon-user />
          </div>
          <div class="flex min-w-0 flex-1 flex-col gap-0.5">
            <div class="truncate text-sm font-semibold text-slate-900">{{ record.identifier }}</div>
            <div class="text-xs text-slate-500">
              <code v-if="record.id" class="text-xs text-slate-500 mono">#{{ record.id }}</code>
            </div>
          </div>
          <RowActionsMenu
            :item="record"
            :actions="[
              { key: 'view', label: '查看详情', icon: 'IconEye' },
              { key: 'edit', label: '编辑', icon: 'IconEdit' },
              { key: 'toggle', label: record.status === 'active' ? '停用' : '启用', icon: record.status === 'active' ? 'IconStop' : 'IconPlayArrow' },
              { key: 'copy', label: '复制ID', icon: 'IconCopy' },
              { key: 'delete-divider', divider: true },
              { key: 'delete', label: '删除', icon: 'IconDelete', danger: true },
            ]"
            @action="handleQuickAction"
          />
        </div>

        <div class="flex flex-col gap-2">
          <div class="flex items-center justify-between gap-2">
            <span class="w-12 flex-shrink-0 text-xs font-medium text-slate-500">类型</span>
            <div class="flex flex-wrap items-center gap-1">
              <code v-if="resolveAccountTypeKey(record)" class="inline-flex items-center rounded border border-slate-200 bg-slate-100 px-1.5 py-0.5 text-xs font-mono text-slate-600">
                {{ resolveAccountTypeKey(record) }}
              </code>
              <span v-else class="text-sm text-slate-400">—</span>
            </div>
          </div>

          <div class="flex items-center justify-between gap-2">
            <span class="w-12 flex-shrink-0 text-xs font-medium text-slate-500">状态</span>
            <div class="flex flex-wrap items-center gap-1">
              <StatusTag :status="record.status" />
            </div>
          </div>

          <div class="flex items-center justify-between gap-2">
            <span class="w-12 flex-shrink-0 text-xs font-medium text-slate-500">标签</span>
            <div class="flex flex-wrap items-center gap-1">
              <template v-if="record.tags?.length">
                <ui-tag v-for="tag in record.tags.slice(0, 3)" :key="tag" size="small">
                  {{ tag }}
                </ui-tag>
                <ui-tag v-if="record.tags.length > 3" size="small" color="gray">
                  +{{ record.tags.length - 3 }}
                </ui-tag>
              </template>
              <span v-else class="text-sm text-slate-400">—</span>
            </div>
          </div>
        </div>
      </ui-card>

      <ui-card
        v-if="!loadingAccounts && !loadingTypes && !filteredAccounts.length"
        class="col-span-full empty-state-block"
      >
        <ui-empty
          :description="accountsError ? `加载失败: ${accountsError}` : (typeKey ? `暂无 [${typeKey}] 类型的账号` : '暂无账号')"
        >
          <ui-button v-if="accountsError" @click="refresh">重新加载</ui-button>
          <ui-button v-else type="primary" @click="router.push(to.accounts.create())">
            创建第一个账号
          </ui-button>
        </ui-empty>
      </ui-card>
    </div>

  </div>
</template>
