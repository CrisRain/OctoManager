<script setup lang="ts">
import { useRouter } from "vue-router";
import {
  IconEmail, IconImport, IconEye, IconEdit, IconDelete, IconPlus
} from "@/lib/icons";

import { PageHeader, SmartListBar, RowActionsMenu, StatusTag } from "@/components/index";
import { useEmailAccountsList } from "@/composables/useEmailAccountsList";
import { to } from "@/router/registry";

const router = useRouter();

const columns = [
  { title: "ID", width: "80", slotName: "id" },
  { title: "邮箱地址", dataIndex: "address", slotName: "address" },
  { title: "服务商", dataIndex: "provider", slotName: "provider" },
  { title: "状态", dataIndex: "status", slotName: "status" },
  { title: "操作", align: "right", slotName: "actions" },
];

const {
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
} = useEmailAccountsList();
</script>

<template>
  <div class="page-shell email-accounts-list-page">
    <PageHeader
      title="邮箱账号列表"
      subtitle="管理已接入的邮箱账号"
      icon-bg="linear-gradient(135deg, rgba(234,88,12,0.12), rgba(249,115,22,0.12))"
      icon-color="var(--icon-orange)"
    >
      <template #icon><icon-email /></template>
      <template #actions>
        <ui-button @click="openBulkImport">
          <template #icon><icon-import /></template>
          批量导入
        </ui-button>
        <ui-button type="primary" @click="router.push(to.emailAccounts.create())">
          <template #icon><icon-plus /></template>
          创建邮箱账号
        </ui-button>
      </template>
    </PageHeader>

    <!-- 智能工具栏 -->
    <SmartListBar
      :data="filteredAccounts"
      :loading="loading"
      v-model:search="searchKeyword"
      @refresh="refresh"
      @batch-delete="handleBatchDelete"
      @batch-export="handleBatchExport"
    >
      <template #filters>
        <!-- 状态筛选 -->
        <div class="flex flex-wrap items-center gap-2">
          <span class="text-xs font-medium text-slate-500">状态：</span>
          <div class="flex flex-wrap items-center gap-1">
            <button type="button"
              class="filter-chip"
              :class="{ active: !statusFilter }"
              @click="statusFilter = ''"
            >
              全部
            </button>
            <button type="button"
              class="filter-chip"
              :class="{ active: statusFilter === 'active' }"
              @click="statusFilter = 'active'"
            >
              已激活
            </button>
            <button type="button"
              class="filter-chip"
              :class="{ active: statusFilter === 'pending' }"
              @click="statusFilter = 'pending'"
            >
              待验证
            </button>
            <button type="button"
              class="filter-chip"
              :class="{ active: statusFilter === 'inactive' }"
              @click="statusFilter = 'inactive'"
            >
              已停用
            </button>
          </div>
        </div>

        <!-- 服务商筛选 -->
        <div class="flex flex-wrap items-center gap-2" v-if="providers.length">
          <span class="text-xs font-medium text-slate-500">服务商：</span>
          <div class="flex flex-wrap items-center gap-1">
            <button type="button"
              class="filter-chip"
              :class="{ active: !providerFilter }"
              @click="providerFilter = ''"
            >
              全部
            </button>
            <button type="button"
              v-for="provider in providers"
              :key="provider"
              class="filter-chip"
              :class="{ active: providerFilter === provider }"
              @click="providerFilter = provider"
            >
              {{ provider }}
            </button>
          </div>
        </div>
      </template>
    </SmartListBar>

    <!-- 数据表格 -->
    <ui-card class="mb-4 hidden lg:block">
      <ui-table
        :data="filteredAccounts"
        :loading="loading"
        :columns="columns"
        :pagination="{
          showTotal: true,
          pageSizeOptions: [10, 20, 50],
          defaultPageSize: 20,
        }"
        :bordered="false"
        row-key="id"
        :row-selection="{ type: 'checkbox' }"
      >
        <!-- ID -->
        <template #id="{ record }">
          <code class="text-xs font-mono font-semibold text-slate-500">#{{ record.id }}</code>
        </template>

        <!-- 邮箱地址 -->
        <template #address="{ record }">
          <div class="flex items-center gap-3">
            <div class="flex h-9 w-9 flex-shrink-0 items-center justify-center rounded-lg border border-orange-200 bg-orange-50 text-sm text-orange-600 shadow-sm">
              <icon-email />
            </div>
            <span class="truncate text-[14px] font-medium text-slate-900">{{ record.address }}</span>
          </div>
        </template>

        <!-- 服务商 -->
        <template #provider="{ record }">
          <code class="inline-flex items-center rounded border border-slate-200 bg-slate-100 px-1.5 py-0.5 text-xs font-mono text-slate-600">{{ record.provider }}</code>
        </template>

        <!-- 状态 -->
        <template #status="{ record }">
          <StatusTag :status="record.status" />
        </template>

        <!-- 快速操作 -->
        <template #actions="{ record }">
          <RowActionsMenu
            :item="record"
            :actions="[
              { key: 'view', label: '预览邮件', icon: IconEye },
              { key: 'edit', label: '配置', icon: IconEdit },
              { key: 'delete-divider', divider: true },
              { key: 'delete', label: '删除', icon: IconDelete, danger: true },
            ]"
            @action="handleQuickAction"
          />
        </template>

        <!-- 空状态 -->
        <template #empty>
          <ui-empty description="暂无邮箱账号">
            <ui-button type="primary" @click="router.push(to.emailAccounts.create())">
              创建第一个邮箱账号
            </ui-button>
          </ui-empty>
        </template>
      </ui-table>
    </ui-card>

    <div class="flex flex-col gap-3 lg:hidden">
      <ui-card
        v-for="record in filteredAccounts"
        :key="record.id"
        class="rounded-xl border p-5 border-slate-200 bg-white shadow-sm"
      >
        <div class="mb-3 flex items-center gap-3">
          <div class="flex h-9 w-9 flex-shrink-0 items-center justify-center rounded-lg border border-orange-200 bg-orange-50 text-sm text-orange-600 shadow-sm">
            <icon-email />
          </div>
          <div class="flex min-w-0 flex-1 flex-col gap-0.5">
            <div class="truncate text-sm font-semibold text-slate-900">{{ record.address }}</div>
            <div class="text-xs text-slate-500">
              <code class="text-xs text-slate-500 mono">#{{ record.id }} · {{ record.provider }}</code>
            </div>
          </div>
          <RowActionsMenu
            :item="record"
            :actions="[
              { key: 'view', label: '预览邮件', icon: IconEye },
              { key: 'edit', label: '配置', icon: IconEdit },
              { key: 'delete-divider', divider: true },
              { key: 'delete', label: '删除', icon: IconDelete, danger: true },
            ]"
            @action="handleQuickAction"
          />
        </div>

        <div class="flex flex-col gap-2">
          <div class="flex items-center justify-between gap-2">
            <span class="w-12 flex-shrink-0 text-xs font-medium text-slate-500">状态</span>
            <div class="flex flex-wrap items-center gap-1">
              <StatusTag :status="record.status" />
            </div>
          </div>
        </div>
      </ui-card>

      <ui-card
        v-if="!loading && !filteredAccounts.length"
        class="col-span-full empty-state-block"
      >
        <ui-empty description="暂无邮箱账号">
          <ui-button type="primary" @click="router.push(to.emailAccounts.create())">
            添加第一个邮箱
          </ui-button>
        </ui-empty>
      </ui-card>
    </div>

    <!-- 批量导入对话框 -->
    <ui-modal
      v-model:visible="showBulkImport"
      title="批量导入邮箱"
      :footer="bulkResult === null"
      ok-text="开始导入"
      :ok-loading="bulkImportLoading"
      @ok="handleBulkImport"
    >
      <div v-if="bulkResult === null">
        <p class="mb-3 text-sm text-slate-500">
          每行一条，格式：<code class="rounded border border-slate-200 bg-slate-100 px-1.5 py-0.5 text-xs font-mono text-slate-700">邮箱----密码----clientid----refresh_token</code>
        </p>
        <ui-textarea
          v-model="bulkText"
          :auto-size="{ minRows: 8, maxRows: 16 }"
          placeholder="user@example.com----password----xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx----0.AXXXXX..."
          class="font-mono"
        />
      </div>
      <div v-else>
        <div class="mb-4 flex items-center gap-3 text-sm">
          <span class="text-slate-600">共 {{ bulkResult.length }} 条</span>
          <span class="font-medium text-emerald-600">成功 {{ bulkResult.filter(r => r.ok).length }}</span>
          <span class="font-medium text-red-600">失败 {{ bulkResult.filter(r => !r.ok).length }}</span>
        </div>
        <div class="flex max-h-64 flex-col gap-2 overflow-y-auto rounded-xl border border-slate-200 p-3">
          <div
            v-for="(item, i) in bulkResult"
            :key="i"
            class="flex items-start gap-2 rounded-lg border px-3 py-2 text-sm"
            :class="item.ok ? 'border-emerald-200 bg-emerald-50/60' : 'border-red-200 bg-red-50/60'"
          >
            <span
              class="mt-1.5 h-1.5 w-1.5 flex-shrink-0 rounded-full"
              :class="item.ok ? 'bg-emerald-500' : 'bg-red-500'"
            />
            <div class="flex flex-col gap-0.5">
              <span class="font-mono text-xs text-slate-700">{{ item.address || item.line }}</span>
              <span v-if="!item.ok" class="text-xs text-red-600">{{ item.error }}</span>
            </div>
          </div>
        </div>
        <div class="mt-4 flex justify-end gap-2">
          <ui-button @click="() => { bulkResult = null; bulkText = ''; }">继续导入</ui-button>
          <ui-button type="primary" @click="showBulkImport = false">关闭</ui-button>
        </div>
      </div>
    </ui-modal>
  </div>
</template>
