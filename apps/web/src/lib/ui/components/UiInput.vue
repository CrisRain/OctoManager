<script setup lang="ts">
defineOptions({
  inheritAttrs: false,
});

import { computed, useAttrs } from "vue";
import { cx, INPUT_SIZE_CLASS } from "../utils";

interface Props {
  modelValue?: string | number | null;
  type?: string;
  placeholder?: string;
  disabled?: boolean;
  allowClear?: boolean;
  size?: "mini" | "small" | "medium" | "large";
  status?: "error" | "warning" | "success";
}

const props = withDefaults(defineProps<Props>(), {
  modelValue: "",
  type: "text",
  placeholder: "",
  disabled: false,
  allowClear: false,
  size: "medium",
  status: undefined,
});

const emit = defineEmits<{
  "update:modelValue": [value: string];
  change: [value: string];
  focus: [event: FocusEvent];
  blur: [event: FocusEvent];
  clear: [];
}>();

const attrs = useAttrs();

const wrapperClass = computed(() =>
  cx(
    "ui-input-wrapper relative inline-flex items-center w-full transition-all duration-200 rounded-lg",
    "bg-white border border-slate-200 shadow-sm",
    "focus-within:border-accent focus-within:ring-2 focus-within:ring-accent/20 focus-within:shadow-none",
    props.disabled && "opacity-60 cursor-not-allowed bg-slate-50",
    props.status === "error" && "border-red-400 focus-within:border-red-500 focus-within:ring-red-500/20",
    props.status === "warning" && "border-amber-400 focus-within:border-amber-500 focus-within:ring-amber-500/20",
    props.status === "success" && "border-emerald-400 focus-within:border-emerald-500 focus-within:ring-emerald-500/20",
    attrs.class as string,
  ),
);

const inputClass = computed(() =>
  cx(
    "ui-input w-full bg-transparent border-none outline-none text-slate-900 placeholder:text-slate-500",
    INPUT_SIZE_CLASS[props.size] ?? INPUT_SIZE_CLASS.medium,
    props.disabled && "cursor-not-allowed",
  ),
);

const showClear = computed(
  () => props.allowClear && String(props.modelValue ?? "").length > 0 && !props.disabled,
);

function onInput(event: Event) {
  emit("update:modelValue", (event.target as HTMLInputElement).value);
}

function onChange(event: Event) {
  emit("change", (event.target as HTMLInputElement).value);
}

function onClear(event: MouseEvent) {
  event.preventDefault();
  event.stopPropagation();
  emit("update:modelValue", "");
  emit("clear");
}
</script>

<template>
  <div v-bind="{ ...attrs, class: undefined }" :class="wrapperClass">
    <div v-if="$slots.prefix" class="ui-input-prefix flex items-center justify-center pl-3 pr-1 text-slate-400">
      <slot name="prefix" />
    </div>

    <input
      :type="type"
      :value="modelValue ?? ''"
      :placeholder="placeholder"
      :disabled="disabled"
      :class="[inputClass, { 'pl-3': !$slots.prefix, 'pr-3': !showClear && !$slots.suffix }]"
      @input="onInput"
      @change="onChange"
      @focus="emit('focus', $event)"
      @blur="emit('blur', $event)"
    />

    <button
      v-if="showClear"
      type="button"
      class="ui-input-clear mx-1 flex h-5 w-5 items-center justify-center rounded-full text-slate-400 hover:bg-slate-100 hover:text-slate-600 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent/40"
      @click="onClear"
    >
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="h-3.5 w-3.5"><line x1="18" y1="6" x2="6" y2="18"></line><line x1="6" y1="6" x2="18" y2="18"></line></svg>
    </button>

    <div v-if="$slots.suffix" class="ui-input-suffix flex items-center justify-center pr-3 pl-1 text-slate-400">
      <slot name="suffix" />
    </div>
  </div>
</template>
