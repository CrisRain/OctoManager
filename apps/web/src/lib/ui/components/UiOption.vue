<script setup lang="ts">
import { inject, ref, computed, onMounted, onBeforeUnmount, watch } from 'vue';
import { cx } from '../utils';

interface Props {
  value?: unknown;
  disabled?: boolean;
}

const props = withDefaults(defineProps<Props>(), {
  value: undefined,
  disabled: false,
});

const context = inject<any>('ui-select-context', null);
const optionRef = ref<HTMLElement | null>(null);

const isSelected = computed(() => {
  if (!context) return false;
  if (context.multiple) {
    return Array.isArray(context.modelValue.value) && context.modelValue.value.includes(props.value);
  }
  return context.modelValue.value === props.value;
});

function updateRegistration() {
  if (context && optionRef.value) {
    const label = optionRef.value.textContent?.trim() || String(props.value);
    context.registerOption(props.value, label);
  }
}

onMounted(() => {
  updateRegistration();
});

watch(() => props.value, (newVal, oldVal) => {
  if (context) {
    if (oldVal !== undefined) {
      context.unregisterOption(oldVal);
    }
    updateRegistration();
  }
});

onBeforeUnmount(() => {
  if (context) {
    context.unregisterOption(props.value);
  }
});

function onClick(e: Event) {
  e.stopPropagation();
  if (props.disabled || !context) return;
  context.selectOption(props.value);
}
</script>

<template>
  <div
    ref="optionRef"
    :class="cx(
      'flex cursor-pointer items-center px-3 py-2 text-sm transition-colors',
      disabled ? 'cursor-not-allowed opacity-50 bg-slate-50' : 'hover:bg-slate-100',
      isSelected ? 'bg-slate-50 font-semibold text-[var(--accent)]' : 'text-slate-700'
    )"
    @click="onClick"
  >
    <slot />
    <span v-if="isSelected && context?.multiple" class="ml-auto text-[var(--accent)]">✓</span>
  </div>
</template>
