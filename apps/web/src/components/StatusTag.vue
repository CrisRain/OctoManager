<script setup lang="ts">
import { computed } from "vue";
import { getStatusLabel, getStatusColor, getStatusDotClass } from "@/shared/utils/status";

const props = defineProps<{ status?: string }>();

const label = computed(() => getStatusLabel(props.status ?? ""));
const color = computed(() => getStatusColor(props.status ?? ""));
const dotClass = computed(() => getStatusDotClass(props.status ?? ""));
</script>

<template>
  <div class="inline-flex items-center gap-2">
    <span class="inline-block h-2 w-2 flex-shrink-0 rounded-full bg-slate-300 [&.offline]:bg-red-500 [&.neutral]:bg-slate-400 motion-safe:[&.online]:animate-[pulse-dot_2s_ease-in-out_infinite] motion-safe:[&.online]:bg-emerald-500" :class="dotClass" />
    <ui-tag :color="color" class="status-tag-pill status-chip-label">{{ label }}</ui-tag>
  </div>
</template>
