<script setup lang="ts">
import { computed, useAttrs } from "vue";
import { cx } from "../utils";

interface Props {
  modelValue?: string | number | null;
  min?: number;
  max?: number;
  step?: number;
  placeholder?: string;
  disabled?: boolean;
}

const props = withDefaults(defineProps<Props>(), {
  modelValue: undefined,
  min: undefined,
  max: undefined,
  step: 1,
  placeholder: "",
  disabled: false,
});

const emit = defineEmits<{
  "update:modelValue": [value: number | undefined];
  change: [value: number | undefined];
  focus: [event: FocusEvent];
  blur: [event: FocusEvent];
}>();

const attrs = useAttrs();

const wrapperClass = computed(() =>
  cx(
    "ui-input-number flex items-center rounded-xl border border-slate-300 bg-white px-3 shadow-input transition-all",
    "focus-within:border-accent focus-within:shadow-input-focus",
    attrs.class as string,
  ),
);

function onInput(event: Event) {
  const raw = (event.target as HTMLInputElement).value;
  if (raw === "") { emit("update:modelValue", undefined); return; }
  const next = Number(raw);
  if (!Number.isNaN(next)) emit("update:modelValue", next);
}

function onChange(event: Event) {
  const raw = (event.target as HTMLInputElement).value;
  const next = raw === "" ? undefined : Number(raw);
  emit("change", Number.isNaN(next) ? undefined : next);
}
</script>

<template>
  <div v-bind="{ ...attrs, class: undefined }" :class="wrapperClass">
    <input
      class="h-9 w-full border-0 bg-transparent text-sm text-slate-900 outline-none placeholder:text-slate-500"
      type="number"
      :value="modelValue ?? ''"
      :min="min"
      :max="max"
      :step="step"
      :placeholder="placeholder"
      :disabled="disabled"
      @input="onInput"
      @change="onChange"
      @focus="emit('focus', $event)"
      @blur="emit('blur', $event)"
    />
  </div>
</template>
