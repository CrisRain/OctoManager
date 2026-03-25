<script setup lang="ts">
import { computed, useAttrs, ref, provide, onMounted, onBeforeUnmount, reactive } from "vue";
import { cx, optionValue } from "../utils";

interface Props {
  modelValue?: unknown;
  placeholder?: string;
  disabled?: boolean;
  allowClear?: boolean;
  multiple?: boolean;
}

const props = withDefaults(defineProps<Props>(), {
  modelValue: undefined,
  placeholder: "",
  disabled: false,
  allowClear: false,
  multiple: false,
});

const emit = defineEmits<{
  "update:modelValue": [value: unknown];
  change: [value: unknown];
  focus: [event: FocusEvent];
  blur: [event: FocusEvent];
  clear: [];
}>();

const attrs = useAttrs();
const isOpen = ref(false);
const selectRef = ref<HTMLElement | null>(null);
const dropdownStyle = ref({});

function updateDropdownPosition() {
  if (!selectRef.value) return;
  const rect = selectRef.value.getBoundingClientRect();
  dropdownStyle.value = {
    top: `${rect.bottom + 4}px`,
    left: `${rect.left}px`,
    width: `${rect.width}px`
  };
}

const wrapperClass = computed(() =>
  cx(
    "ui-select-view relative flex items-center rounded-lg border border-slate-200 bg-white px-3 shadow-sm transition-all hover:border-slate-300",
    isOpen.value && "ring-2 ring-slate-400/20 border-[var(--accent)] shadow-input-focus",
    !props.multiple && "ui-select-view-single",
    props.disabled && "bg-white/50 opacity-60 cursor-not-allowed",
    attrs.class as string,
  ),
);

const hasRegisteredOptions = computed(() => optionsMap.size > 0);

// Map of value -> label
const optionsMap = reactive(new Map<unknown, string>());

provide('ui-select-context', {
  modelValue: computed(() => props.modelValue),
  multiple: props.multiple,
  registerOption: (value: unknown, label: string) => {
    optionsMap.set(value, label);
  },
  unregisterOption: (value: unknown) => {
    optionsMap.delete(value);
  },
  selectOption: (value: unknown) => {
    if (props.multiple) {
      const current = Array.isArray(props.modelValue) ? [...props.modelValue] : [];
      const idx = current.indexOf(value);
      if (idx >= 0) current.splice(idx, 1);
      else current.push(value);
      emit("update:modelValue", current);
      emit("change", current);
    } else {
      emit("update:modelValue", value);
      emit("change", value);
      isOpen.value = false;
    }
  }
});

const displayLabel = computed(() => {
  if (props.multiple) {
    const vals = Array.isArray(props.modelValue) ? props.modelValue : [];
    if (vals.length === 0) return props.placeholder;
    return vals.map(v => optionsMap.get(v) || String(v)).join(', ');
  }
  if (props.modelValue === '' || props.modelValue == null) return props.placeholder;
  return optionsMap.get(props.modelValue) || String(props.modelValue);
});

const showClear = computed(() => props.allowClear && !props.multiple && props.modelValue);

function toggleOpen(e: Event) {
  if (props.disabled) return;
  // If the target is the clear button, don't toggle
  if ((e.target as HTMLElement).closest('.clear-btn')) return;
  isOpen.value = !isOpen.value;
  if (isOpen.value) {
    updateDropdownPosition();
    emit('focus', e as FocusEvent);
  } else {
    emit('blur', e as FocusEvent);
  }
}

function onClear(event: MouseEvent) {
  event.preventDefault();
  event.stopPropagation();
  emit("update:modelValue", "");
  emit("change", "");
  emit("clear");
  isOpen.value = false;
}

function handleClickOutside(event: MouseEvent) {
  if (selectRef.value && !selectRef.value.contains(event.target as Node)) {
    if (isOpen.value) {
      isOpen.value = false;
      emit('blur', event as any);
    }
  }
}

onMounted(() => {
  document.addEventListener('click', handleClickOutside);
  window.addEventListener('resize', updateDropdownPosition);
  window.addEventListener('scroll', updateDropdownPosition, true);
});

onBeforeUnmount(() => {
  document.removeEventListener('click', handleClickOutside);
  window.removeEventListener('resize', updateDropdownPosition);
  window.removeEventListener('scroll', updateDropdownPosition, true);
});
</script>

<template>
  <div ref="selectRef" v-bind="{ ...attrs, class: undefined }" :class="wrapperClass" @click="toggleOpen">
    <div class="min-h-[2.85em] w-full flex items-center pr-8 text-sm font-medium tracking-[-0.01em] text-slate-900 outline-none cursor-pointer">
      <span :class="{'text-slate-400 font-normal': displayLabel === placeholder && (modelValue === '' || modelValue == null)}" class="truncate block w-full text-left">
        {{ displayLabel }}
      </span>
    </div>

    <button
      v-if="showClear"
      type="button"
      class="clear-btn absolute right-8 mr-1 rounded-full bg-slate-100/85 px-2 py-1 text-slate-400 transition hover:bg-white hover:text-slate-700"
      @click.stop="onClear"
    >
      ×
    </button>
    <span v-else class="pointer-events-none absolute right-4 text-slate-400 transition-transform" :class="{'rotate-180': isOpen}">▾</span>
    
    <!-- Dropdown menu -->
    <Teleport to="body">
      <div 
        v-show="isOpen" 
        class="fixed z-[9999] max-h-60 overflow-y-auto rounded-lg border border-slate-200 bg-white py-1 shadow-lg cursor-default" 
        :style="dropdownStyle"
        @click.stop
      >
        <slot />
        <div v-if="!hasRegisteredOptions" class="px-3 py-2 text-sm text-slate-400 text-center">暂无选项</div>
      </div>
    </Teleport>
  </div>
</template>
