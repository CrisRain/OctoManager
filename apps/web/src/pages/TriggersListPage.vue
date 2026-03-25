<script setup lang="ts">
import { ref, computed } from "vue";
import { useRouter } from "vue-router";
import { useTriggers, useDeleteTrigger, useFireTrigger } from "@/composables/useTriggers";
import { useMessage, useConfirm, useErrorHandler } from "@/composables";
import { PageHeader, SmartListBar, RowActionsMenu } from "@/components/index";
import { to } from "@/router/registry";

const router = useRouter();
const message = useMessage();
const confirm = useConfirm();
const { withErrorHandler } = useErrorHandler();
const { data: items, loading, refresh } = useTriggers();
const deleteTrigger = useDeleteTrigger();
const fireTrigger = useFireTrigger();

const searchKeyword = ref("");
const payloadText = ref("{}");
const resultText = ref("");

const filteredItems = computed(() => {
  let result = items.value;
  if (searchKeyword.value) {
    const keyword = searchKeyword.value.toLowerCase();
    result = result.filter(item =>
      item.name?.toLowerCase().includes(keyword) ||
      item.key?.toLowerCase().includes(keyword)
    );
  }
  return result;
});

function safePayload(): Record<string, unknown> {
  try {
    const p = JSON.parse(payloadText.value);
    if (p && typeof p === "object" && !Array.isArray(p)) return p as Record<string, unknown>;
    return {};
  } catch { return {}; }
}

async function handleQuickAction(key: string, trigger: any) {
  switch (key) {
    case "fire":
      await handleFire(trigger.id);
      break;
    case "edit":
      router.push(to.triggers.edit(trigger.id));
      break;
    case "delete":
      await handleDelete(trigger);
      break;
  }
}

async function handleFire(id: number) {
  try {
    const result = await fireTrigger.execute(id, safePayload());
    resultText.value = JSON.stringify(result, null, 2);
    message.success("触发成功");
  } catch (e) {
    message.error(e instanceof Error ? e.message : "触发失败");
  }
}

async function handleDelete(trigger: any) {
  const confirmed = await confirm.confirmDelete(trigger.name || trigger.key);
  if (!confirmed) return;

  await withErrorHandler(
    async () => {
      await deleteTrigger.execute(trigger.id);
      message.success("已删除");
      await refresh();
    },
    { action: "删除", showSuccess: true }
  );
}

async function handleBatchDelete(selectedItems: any[]) {
  const confirmed = await confirm.confirm(`确定要删除选中的 ${selectedItems.length} 个触发器吗？`);
  if (!confirmed) return;

  await withErrorHandler(
    async () => {
      await Promise.all(selectedItems.map((item) => deleteTrigger.execute(item.id)));
      message.success(`已删除 ${selectedItems.length} 个触发器`);
      await refresh();
    },
    { action: "批量删除", showSuccess: true }
  );
}

async function handleBatchExport(selectedItems: any[]) {
  const data = JSON.stringify(selectedItems, null, 2);
  const blob = new Blob([data], { type: "application/json" });
  const url = URL.createObjectURL(blob);
  const link = document.createElement("a");
  link.href = url;
  link.download = `triggers-${Date.now()}.json`;
  link.click();
  URL.revokeObjectURL(url);
  message.success(`已导出 ${selectedItems.length} 个触发器`);
}
</script>

<template>
  <div class="page-shell">
    <PageHeader
      title="触发器"
      subtitle="通过 HTTP 调用触发任务执行，支持 Webhook 接入。"
      icon-bg="linear-gradient(135deg, rgba(234,179,8,0.12), rgba(202,138,4,0.12))"
      icon-color="var(--icon-yellow)"
    >
      <template #icon><icon-thunderbolt /></template>
      <template #actions>
        <ui-button type="primary" @click="router.push(to.triggers.create())">
          <template #icon><icon-plus /></template>
          新建触发器
        </ui-button>
      </template>
    </PageHeader>

    <!-- 智能工具栏 -->
    <SmartListBar
      :data="filteredItems"
      :loading="loading"
      v-model:search="searchKeyword"
      @refresh="refresh"
      @batch-delete="handleBatchDelete"
      @batch-export="handleBatchExport"
    />

    <!-- 数据表格 -->
    <ui-card class="mb-4 hidden lg:block">
      <ui-table
        :data="filteredItems"
        :loading="loading"
        :pagination="{
          showTotal: true,
          pageSizeOptions: [10, 20, 50],
          defaultPageSize: 20,
        }"
        :bordered="false"
        row-key="id"
        :row-selection="{ type: 'checkbox' }"
      >
        <template #columns>
          <!-- ID -->
          <ui-table-column title="ID" width="80">
            <template #cell="{ record }">
              <code class="text-xs font-mono font-semibold text-slate-500">#{{ record.id }}</code>
            </template>
          </ui-table-column>

          <ui-table-column title="名称 / Key">
            <template #cell="{ record }">
              <div class="truncate text-[14px] font-medium text-slate-900">{{ record.name }}</div>
              <code class="inline-flex items-center rounded-md border border-slate-200 bg-slate-100 px-2 py-0.5 text-xs font-mono text-slate-600 mt-1">{{ record.key }}</code>
            </template>
          </ui-table-column>

          <ui-table-column title="执行模式">
            <template #cell="{ record }">
              <span class="inline-flex items-center rounded-full border px-3 py-1 text-xs font-semibold shadow-sm [&.sync]:border-sky-200 [&.sync]:bg-sky-50 [&.sync]:text-sky-700 [&.async]:border-[var(--accent)]/20 [&.async]:bg-[var(--accent)]/8 [&.async]:text-[var(--accent)]" :class="record.mode === 'sync' ? 'sync' : 'async'">
                {{ record.mode }}
              </span>
            </template>
          </ui-table-column>

          <ui-table-column title="Token 前缀">
            <template #cell="{ record }">
              <span class="font-mono text-sm text-slate-400">{{ record.token_prefix }}••••••</span>
            </template>
          </ui-table-column>

          <ui-table-column title="操作" align="right">
            <template #cell="{ record }">
              <RowActionsMenu
                :item="record"
                :actions="[
                  { key: 'fire', label: '测试触发', icon: 'IconThunderbolt' },
                  { key: 'edit', label: '编辑', icon: 'IconEdit' },
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
          <ui-empty description="暂无触发器">
            <ui-button type="primary" @click="router.push(to.triggers.create())">
              新建触发器
            </ui-button>
          </ui-empty>
        </template>
      </ui-table>
    </ui-card>

    <div class="flex flex-col gap-3 lg:hidden">
      <ui-card
        v-for="record in filteredItems"
        :key="record.id"
        class="rounded-xl border p-5 border-slate-200 bg-white shadow-sm"
      >
        <div class="mb-3 flex items-center gap-3">
          <div class="flex h-9 w-9 flex-shrink-0 items-center justify-center rounded-lg border border-amber-200 bg-amber-50 text-sm text-amber-600 shadow-sm">
            <icon-thunderbolt />
          </div>
          <div class="flex min-w-0 flex-1 flex-col gap-0.5">
            <div class="truncate text-sm font-semibold text-slate-900">{{ record.name }}</div>
            <div class="text-xs text-slate-500">
              <code class="text-xs text-slate-500 mono">#{{ record.id }} · {{ record.key }}</code>
            </div>
          </div>
          <RowActionsMenu
            :item="record"
            :actions="[
              { key: 'fire', label: '测试触发', icon: 'IconThunderbolt' },
              { key: 'edit', label: '编辑', icon: 'IconEdit' },
              { key: 'delete-divider', divider: true },
              { key: 'delete', label: '删除', icon: 'IconDelete', danger: true },
            ]"
            @action="handleQuickAction"
          />
        </div>

        <div class="flex flex-col gap-2">
          <div class="flex items-center justify-between gap-2">
            <span class="w-16 flex-shrink-0 text-xs font-medium text-slate-500">模式</span>
            <span class="inline-flex items-center rounded-full border px-3 py-1 text-xs font-semibold shadow-sm [&.sync]:border-sky-200 [&.sync]:bg-sky-50 [&.sync]:text-sky-700 [&.async]:border-[var(--accent)]/20 [&.async]:bg-[var(--accent)]/8 [&.async]:text-[var(--accent)]" :class="record.mode === 'sync' ? 'sync' : 'async'">
              {{ record.mode }}
            </span>
          </div>
          <div class="flex items-center justify-between gap-2">
            <span class="w-16 flex-shrink-0 text-xs font-medium text-slate-500">Token</span>
            <span class="font-mono text-xs text-slate-400">{{ record.token_prefix }}••••••</span>
          </div>
        </div>
      </ui-card>

      <ui-card
        v-if="!loading && !filteredItems.length"
        class="col-span-full empty-state-block"
      >
        <ui-empty description="暂无触发器">
          <ui-button type="primary" @click="router.push(to.triggers.create())">
            新建触发器
          </ui-button>
        </ui-empty>
      </ui-card>
    </div>

    <!-- Test panel (only shown when there are triggers) -->
    <ui-card v-if="items.length" class="min-w-0 flex-1 rounded-xl border overflow-hidden border-slate-200 bg-white shadow">
      <template #title>
        <div class="flex items-center gap-2">
          <icon-thunderbolt class="h-4 w-4 text-amber-600" />
          测试发送
        </div>
      </template>
      <ui-form layout="vertical">
        <ui-form-item label="请求参数（JSON）">
          <ui-textarea
            v-model="payloadText"
            placeholder="{}"
            :auto-size="{ minRows: 3 }"
            class="font-mono text-sm"
          />
          <div class="text-sm leading-6 text-slate-500">点击列表中的「测试」按钮时，此参数会一起发送。</div>
        </ui-form-item>
        <ui-form-item v-if="resultText" label="返回结果">
          <ui-textarea :model-value="resultText" :auto-size="{ minRows: 4 }" readonly class="font-mono text-sm" />
        </ui-form-item>
      </ui-form>
    </ui-card>
  </div>
</template>
