<script setup lang="ts">
import { computed, useSlots } from "vue";

interface Props {
  loading?: boolean;
  ready?: boolean;
  emptyDescription?: string;
}

const props = withDefaults(defineProps<Props>(), {
  loading: false,
  ready: true,
  emptyDescription: "未找到相关数据",
});

const slots = useSlots();

const hasAside = computed(() => Boolean(slots.aside));
const hasActions = computed(() => Boolean(slots.actions));
const contentClass = computed(() =>
  hasAside.value
    ? "grid grid-cols-1 items-start gap-6 pb-24 lg:grid-cols-[minmax(0,_1.45fr)_minmax(16em,_0.85fr)]"
    : "flex flex-col gap-6 pb-24"
);
</script>

<template>
  <div v-if="loading" class="empty-state-block">
    <ui-spin size="2.25em" />
  </div>
  <ui-card v-else-if="!ready" class="empty-state-block">
    <ui-empty :description="emptyDescription">
      <slot name="empty-action" />
    </ui-empty>
  </ui-card>
  <template v-else>
    <div :class="contentClass">
      <div class="flex min-w-0 flex-col gap-6">
        <slot name="main" />
      </div>
      <div v-if="hasAside" class="flex min-w-0 flex-col gap-6">
        <slot name="aside" />
      </div>
    </div>
    <slot v-if="hasActions" name="actions" />
  </template>
</template>
