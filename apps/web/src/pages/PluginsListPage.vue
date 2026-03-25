<script setup lang="ts">
import { ref, computed, watch } from "vue";
import { useRouter } from "vue-router";
import { IconApps, IconSync, IconEye, IconRefresh } from "@/lib/icons";

import { PageHeader, SmartListBar } from "@/components/index";
import { usePlugins, useSyncPlugins } from "@/composables/usePlugins";
import { useMessage, useErrorHandler } from "@/composables";
import { to } from "@/router/registry";

const router = useRouter();
const message = useMessage();
const { withErrorHandler } = useErrorHandler();

const { data: plugins, loading, refresh } = usePlugins();
const selectedKeys = ref<string[]>([]);
watch(plugins, () => { selectedKeys.value = []; });
const sync = useSyncPlugins();

// 筛选
const healthFilter = ref<string>();
const searchKeyword = ref("");

// 同步结果
const syncResult = ref<{ synced: number; failed: number; errors: string[] } | null>(null);

// 过滤插件
const filteredPlugins = computed(() => {
  let result = plugins.value;

  // 按健康状态筛选
  if (healthFilter.value === "healthy") {
    result = result.filter(item => item.healthy);
  } else if (healthFilter.value === "unhealthy") {
    result = result.filter(item => !item.healthy);
  }

  // 搜索过滤
  if (searchKeyword.value) {
    const keyword = searchKeyword.value.toLowerCase();
    result = result.filter(item =>
      item.manifest.key.toLowerCase().includes(keyword) ||
      item.manifest.name.toLowerCase().includes(keyword)
    );
  }

  return result;
});

// 同步插件
async function handleSync() {
  syncResult.value = null;

  await withErrorHandler(
    async () => {
      const result = await sync.execute();
      syncResult.value = result;
      await refresh();
    },
    { action: "同步插件" }
  );
}

// 查看详情
function viewDetail(plugin: any) {
  router.push(to.plugins.detail(plugin.manifest.key));
}

// 批量操作
async function handleBatchExport(items: any[]) {
  const data = JSON.stringify(items, null, 2);
  const blob = new Blob([data], { type: "application/json" });
  const url = URL.createObjectURL(blob);
  const link = document.createElement("a");
  link.href = url;
  link.download = `plugins-${Date.now()}.json`;
  link.click();
  URL.revokeObjectURL(url);
  message.success(`已导出 ${items.length} 个插件`);
}

// 关闭同步结果
function closeSyncResult() {
  syncResult.value = null;
}
</script>

<template>
  <div class="page-shell plugins-list-page">
    <PageHeader
      title="插件管理"
      subtitle="查看和管理已加载的系统插件"
      icon-bg="linear-gradient(135deg, rgba(236,72,153,0.12), rgba(219,39,119,0.12))"
      icon-color="var(--icon-pink)"
    >
      <template #icon><icon-apps /></template>
      <template #actions>
        <ui-button type="primary" :loading="sync.loading.value" @click="handleSync">
          <template #icon><icon-sync /></template>
          同步插件
        </ui-button>
      </template>
    </PageHeader>

    <!-- 同步结果横幅 -->
    <div
      v-if="syncResult"
      class="mb-5 rounded-xl border p-4 shadow-sm"
      :class="syncResult.failed ? 'border-amber-200 bg-amber-50/80 text-amber-800' : 'border-emerald-200 bg-emerald-50/80 text-emerald-800'"
    >
      <div class="flex items-center gap-3">
        <icon-check-circle v-if="!syncResult.failed" class="h-5 w-5 flex-shrink-0 text-emerald-600" />
        <icon-exclamation-circle v-else class="h-5 w-5 flex-shrink-0 text-amber-600" />
        <span>
          同步完成：<strong>{{ syncResult.synced }}</strong> 个成功
          <template v-if="syncResult.failed">，<strong>{{ syncResult.failed }}</strong> 个失败</template>
        </span>
        <ui-button size="mini" type="text" @click="closeSyncResult">
          <template #icon><icon-close /></template>
        </ui-button>
      </div>
      <ul v-if="syncResult.errors.length" class="mt-3 pl-5 text-sm leading-6">
        <li v-for="(msg, i) in syncResult.errors" :key="i">{{ msg }}</li>
      </ul>
    </div>

    <!-- 智能工具栏 -->
    <SmartListBar
      :data="filteredPlugins"
      :loading="loading"
      v-model:search="searchKeyword"
      v-model:selectedKeys="selectedKeys"
      :row-key="(r: any) => r.manifest.key"
      @refresh="refresh"
      @batch-export="handleBatchExport"
    >
      <template #filters>
        <!-- 健康状态筛选 -->
        <div class="flex flex-wrap items-center gap-2">
          <span class="text-xs font-medium text-slate-500">状态：</span>
          <div class="flex flex-wrap items-center gap-1">
            <button type="button"
              class="filter-chip"
              :class="{ active: !healthFilter }"
              @click="healthFilter = ''"
            >
              全部
            </button>
            <button type="button"
              class="filter-chip"
              :class="{ active: healthFilter === 'healthy' }"
              @click="healthFilter = 'healthy'"
            >
              正常
            </button>
            <button type="button"
              class="filter-chip"
              :class="{ active: healthFilter === 'unhealthy' }"
              @click="healthFilter = 'unhealthy'"
            >
              异常
            </button>
          </div>
        </div>
      </template>
    </SmartListBar>

    <!-- 数据表格 -->
    <ui-card class="mb-4 hidden lg:block">
      <ui-table
        :data="filteredPlugins"
        :loading="loading"
        :pagination="{
          showTotal: true,
          pageSizeOptions: [10, 20, 50],
          defaultPageSize: 20,
        }"
        :bordered="false"
        :row-key="(r: any) => r.manifest.key"
        :row-selection="{ type: 'checkbox' }"
        v-model:selectedKeys="selectedKeys"
      >
        <template #columns>
          <!-- 插件 Key -->
          <ui-table-column title="插件标识" data-index="manifest.key">
            <template #cell="{ record }">
              <div class="flex flex-col gap-0.5">
                <code class="inline-flex items-center rounded-full border px-2.5 py-1 text-xs font-mono font-semibold text-slate-700 border-slate-200 bg-white/65">@{{ record.manifest.key }}</code>
              </div>
            </template>
          </ui-table-column>

          <!-- 名称 -->
          <ui-table-column title="名称" data-index="manifest.name">
            <template #cell="{ record }">
              <span class="text-sm font-semibold text-slate-900">{{ record.manifest.name }}</span>
            </template>
          </ui-table-column>

          <!-- 版本 -->
          <ui-table-column title="版本" data-index="manifest.version">
            <template #cell="{ record }">
              <code class="inline-flex items-center rounded-full border px-2.5 py-1 text-xs font-mono font-semibold text-sky-700 border-slate-200 bg-slate-50">{{ record.manifest.version || 'Unmarked' }}</code>
            </template>
          </ui-table-column>

          <!-- 操作数 -->
          <ui-table-column title="可用操作">
            <template #cell="{ record }">
              <span class="text-sm font-medium text-slate-700">{{ record.manifest.actions?.length || 0 }} 个</span>
            </template>
          </ui-table-column>

          <!-- 状态 -->
          <ui-table-column title="状态">
            <template #cell="{ record }">
              <span
                class="inline-flex items-center gap-2 rounded-full border px-3 py-1 text-xs font-semibold"
                :class="record.healthy ? 'border-emerald-200 bg-emerald-50 text-emerald-700' : 'border-red-200 bg-red-50 text-red-700'"
              >
                <span
                  class="inline-block h-2 w-2 flex-shrink-0 rounded-full"
                  :class="record.healthy ? 'bg-emerald-500 animate-pulse' : 'bg-red-500'"
                ></span>
                {{ record.healthy ? '正常' : '异常' }}
              </span>
            </template>
          </ui-table-column>

          <!-- 快速操作 -->
          <ui-table-column title="操作" align="right">
            <template #cell="{ record }">
              <ui-button size="small" type="text" @click="viewDetail(record)">
                <template #icon><icon-eye /></template>
                查看详情
              </ui-button>
            </template>
          </ui-table-column>
        </template>

        <!-- 空状态 -->
        <template #empty>
          <ui-empty description="暂无插件">
            <p class="text-sm leading-6 text-slate-500">请将插件放入 plugins/modules 目录后重启服务，或点击「同步插件」</p>
            <ui-button type="primary" @click="handleSync">
              <template #icon><icon-sync /></template>
              同步插件
            </ui-button>
          </ui-empty>
        </template>
      </ui-table>
    </ui-card>

    <div class="flex flex-col gap-3 lg:hidden">
      <ui-card
        v-for="record in filteredPlugins"
        :key="record.manifest.key"
        class="rounded-xl border p-5 border-slate-200 bg-white shadow-sm"
      >
        <div class="mb-3 flex items-center gap-3">
          <div class="flex h-9 w-9 flex-shrink-0 items-center justify-center rounded-lg text-sm text-white icon-pink">
            <icon-apps />
          </div>
          <div class="flex min-w-0 flex-1 flex-col gap-0.5">
            <div class="truncate text-sm font-semibold text-slate-900">{{ record.manifest.name }}</div>
            <div class="text-xs text-slate-500">
              <code class="inline-flex items-center rounded-full border px-2.5 py-1 text-xs font-mono font-semibold text-slate-700 border-slate-200 bg-white/65">@{{ record.manifest.key }}</code>
            </div>
          </div>
          <ui-button size="small" type="text" @click="viewDetail(record)">
            <template #icon><icon-eye /></template>
            查看
          </ui-button>
        </div>

        <div class="flex flex-col gap-2">
          <div class="flex items-center justify-between gap-2">
            <span class="w-12 flex-shrink-0 text-xs font-medium text-slate-500">版本</span>
            <div class="flex flex-wrap items-center gap-1">
              <code class="inline-flex items-center rounded-full border px-2.5 py-1 text-xs font-mono font-semibold text-sky-700 border-slate-200 bg-slate-50">{{ record.manifest.version || 'Unmarked' }}</code>
            </div>
          </div>

          <div class="flex items-center justify-between gap-2">
            <span class="w-12 flex-shrink-0 text-xs font-medium text-slate-500">操作数</span>
            <div class="flex flex-wrap items-center gap-1">
              <span class="text-sm font-medium text-slate-700">{{ record.manifest.actions?.length || 0 }} 个</span>
            </div>
          </div>

          <div class="flex items-center justify-between gap-2">
            <span class="w-12 flex-shrink-0 text-xs font-medium text-slate-500">状态</span>
            <div class="flex flex-wrap items-center gap-1">
              <span
                class="inline-flex items-center gap-2 rounded-full border px-3 py-1 text-xs font-semibold"
                :class="record.healthy ? 'border-emerald-200 bg-emerald-50 text-emerald-700' : 'border-red-200 bg-red-50 text-red-700'"
              >
                <span
                  class="inline-block h-2 w-2 flex-shrink-0 rounded-full"
                  :class="record.healthy ? 'bg-emerald-500 animate-pulse' : 'bg-red-500'"
                ></span>
                {{ record.healthy ? '正常' : '异常' }}
              </span>
            </div>
          </div>
        </div>
      </ui-card>

      <ui-card
        v-if="!loading && !filteredPlugins.length"
        class="col-span-full empty-state-block"
      >
        <ui-empty description="暂无插件">
          <p class="text-sm leading-6 text-slate-500">请将插件放入 plugins/modules 目录后重启服务，或点击「同步插件」</p>
          <ui-button type="primary" @click="handleSync">
            <template #icon><icon-sync /></template>
            同步插件
          </ui-button>
        </ui-empty>
      </ui-card>
    </div>
  </div>
</template>
