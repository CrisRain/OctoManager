<script setup lang="ts">
import { computed, ref, watch, watchEffect, useSlots, useAttrs, type VNode } from "vue";
import { flattenNodes, getFromPath, cx } from "../utils";
import UiSpin from "./UiSpin.vue";
import UiEmpty from "./UiEmpty.vue";
import UiButton from "./UiButton.vue";

interface ParsedColumn {
  key: string;
  title?: string;
  dataIndex?: string;
  align?: "left" | "center" | "right";
  slotName?: string;
  cell?: (args: { record: unknown; rowIndex: number; column: ParsedColumn }) => VNode[];
}

interface Props {
  data?: unknown[];
  columns?: Array<Record<string, unknown>>;
  loading?: boolean;
  pagination?: boolean | Record<string, unknown>;
  rowKey?: string | ((record: unknown) => string | number);
  rowSelection?: { type: "checkbox" } | null;
}

const props = withDefaults(defineProps<Props>(), {
  data: () => [],
  columns: undefined,
  loading: false,
  pagination: false,
  rowKey: "id",
  rowSelection: null,
});

const selectedKeys = defineModel<string[]>("selectedKeys", { default: () => [] });

const slots = useSlots();
const attrs = useAttrs();

const currentPage = ref(1);

const pageSize = computed(() => {
  if (props.pagination && typeof props.pagination === "object") {
    const size = Number(
      (props.pagination as Record<string, unknown>).pageSize ??
        (props.pagination as Record<string, unknown>).defaultPageSize ??
        20,
    );
    return Number.isFinite(size) && size > 0 ? size : 20;
  }
  return 20;
});

const paginationEnabled = computed(() => Boolean(props.pagination));
const allRows = computed(() => (Array.isArray(props.data) ? props.data : []));
const pagedRows = computed(() => {
  if (!paginationEnabled.value) return allRows.value;
  const start = (currentPage.value - 1) * pageSize.value;
  return allRows.value.slice(start, start + pageSize.value);
});
const totalPages = computed(() => Math.max(1, Math.ceil(allRows.value.length / pageSize.value)));

watch([allRows, totalPages], () => {
  if (currentPage.value > totalPages.value) currentPage.value = totalPages.value;
});

const parsedColumns = computed<ParsedColumn[]>(() => {
  if (Array.isArray(props.columns) && props.columns.length > 0) {
    return props.columns.map((col, i) => ({
      key: String(col.key ?? col.dataIndex ?? i),
      title: String(col.title ?? ""),
      dataIndex: typeof col.dataIndex === "string" ? col.dataIndex : undefined,
      align: (col.align as ParsedColumn["align"]) ?? "left",
      slotName: typeof col.slotName === "string" ? col.slotName : undefined,
    }));
  }

  const source = flattenNodes(slots.columns?.() ?? slots.default?.() ?? []);
  return source
    .filter((node) => {
      const type = node.type as { name?: string; __name?: string };
      return type?.name === "UiTableColumn" || type?.__name === "UiTableColumn";
    })
    .map((node, index) => {
      const np = (node.props ?? {}) as Record<string, unknown>;
      const children = node.children as Record<string, (...args: unknown[]) => VNode[]> | null;
      return {
        key: String(node.key ?? np.dataIndex ?? index),
        title: typeof np.title === "string" ? np.title : "",
        dataIndex: typeof np.dataIndex === "string" ? np.dataIndex : undefined,
        align: (np.align as ParsedColumn["align"]) ?? "left",
        cell: children?.cell ? (args) => children.cell?.(args) ?? [] : undefined,
      } satisfies ParsedColumn;
    });
});

function renderCell(column: ParsedColumn, record: unknown, rowIndex: number) {
  if (column.slotName && slots[column.slotName]) {
    return slots[column.slotName]?.({ record, rowIndex, column });
  }
  if (column.cell) return column.cell({ record, rowIndex, column });
  const value = getFromPath(record, column.dataIndex);
  if (value == null || value === "") return "-";
  return String(value);
}

function keyForRow(record: unknown, index: number): string {
  if (typeof props.rowKey === "function") return String(props.rowKey(record));
  if (record && typeof record === "object") {
    const maybe = (record as Record<string, unknown>)[props.rowKey as string];
    if (maybe != null) return String(maybe);
  }
  return String(index);
}

// Row selection
const headerCheckboxRef = ref<HTMLInputElement | null>(null);

const isAllSelected = computed(() =>
  allRows.value.length > 0 &&
  allRows.value.every((r, i) => selectedKeys.value.includes(keyForRow(r, i)))
);

const isIndeterminate = computed(() =>
  selectedKeys.value.length > 0 && !isAllSelected.value
);

watchEffect(() => {
  if (headerCheckboxRef.value) {
    headerCheckboxRef.value.indeterminate = isIndeterminate.value;
  }
});

function toggleSelectAll() {
  if (isAllSelected.value) {
    selectedKeys.value = [];
  } else {
    selectedKeys.value = allRows.value.map((r, i) => keyForRow(r, i));
  }
}

function toggleRowSelected(key: string) {
  if (selectedKeys.value.includes(key)) {
    selectedKeys.value = selectedKeys.value.filter(k => k !== key);
  } else {
    selectedKeys.value = [...selectedKeys.value, key];
  }
}
</script>

<template>
  <div
    v-if="loading"
    v-bind="{ ...attrs, class: undefined }"
    :class="cx('ui-table-container flex min-h-[12em] items-center justify-center', attrs.class as string)"
  >
    <UiSpin :loading="true" tip="加载中..." />
  </div>

  <div v-else v-bind="{ ...attrs, class: undefined }" :class="cx('ui-table-container ui-table-content overflow-x-auto', attrs.class as string)">
    <table class="ui-table ui-table-element min-w-full border-collapse text-sm">
      <thead v-if="parsedColumns.length">
        <tr>
          <th
            v-if="rowSelection"
            class="ui-table-th border-b border-slate-200 bg-slate-50/50 w-10 px-3 py-3"
          >
            <input
              ref="headerCheckboxRef"
              type="checkbox"
              :checked="isAllSelected"
              class="h-4 w-4 cursor-pointer rounded border-slate-300 accent-[var(--accent)]"
              @change="toggleSelectAll"
            />
          </th>
          <th
            v-for="col in parsedColumns"
            :key="col.key"
            class="ui-table-th ui-table-column border-b border-slate-200 bg-slate-50/50 px-5 py-3 text-left text-[12px] font-medium uppercase tracking-wider text-slate-500"
            :style="{ textAlign: col.align ?? 'left' }"
          >
            {{ col.title }}
          </th>
        </tr>
      </thead>
      <tbody>
        <tr
          v-for="(record, rowIndex) in pagedRows"
          :key="keyForRow(record, rowIndex)"
          class="ui-table-tr transition-colors duration-150 hover:bg-slate-50/50"
        >
          <td
            v-if="rowSelection"
            class="ui-table-td border-b border-slate-200 w-10 px-3 py-3.5 align-top"
          >
            <input
              type="checkbox"
              :checked="selectedKeys.includes(keyForRow(record, rowIndex))"
              class="h-4 w-4 cursor-pointer rounded border-slate-300 accent-[var(--accent)]"
              @change="toggleRowSelected(keyForRow(record, rowIndex))"
            />
          </td>
          <td
            v-for="col in parsedColumns"
            :key="`${col.key}-${rowIndex}`"
            class="ui-table-td ui-table-column border-b border-slate-200 px-5 py-3.5 align-top text-[14px] text-slate-700"
            :style="{ textAlign: col.align ?? 'left' }"
          >
            <component :is="() => renderCell(col, record, rowIndex)" />
          </td>
        </tr>
        <tr v-if="pagedRows.length === 0">
          <td class="ui-table-td px-4 py-10" :colspan="Math.max(parsedColumns.length, 1) + (rowSelection ? 1 : 0)">
            <slot name="empty">
              <UiEmpty description="暂无数据" />
            </slot>
          </td>
        </tr>
      </tbody>
    </table>

    <div v-if="paginationEnabled && totalPages > 1" class="ui-pagination flex items-center justify-end gap-2 px-4 py-3">
      <UiButton size="small" :disabled="currentPage <= 1" @click="currentPage = Math.max(1, currentPage - 1)">
        上一页
      </UiButton>
      <span class="text-xs text-slate-500">{{ currentPage }} / {{ totalPages }}</span>
      <UiButton size="small" :disabled="currentPage >= totalPages" @click="currentPage = Math.min(totalPages, currentPage + 1)">
        下一页
      </UiButton>
    </div>
  </div>
</template>
