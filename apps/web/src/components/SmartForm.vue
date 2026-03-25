<script setup lang="ts">
import { ref, watch } from "vue";
import type { FieldConfig } from "./smart-form.types";
import { IconEye, IconEyeInvisible, IconCheck, IconInfoCircle } from "@/lib/icons";

interface Props {
  fields: FieldConfig[];
  modelValue: Record<string, any>;
  loading?: boolean;
}

const props = withDefaults(defineProps<Props>(), {
  loading: false,
});

const emit = defineEmits<{
  (e: "update:modelValue", value: Record<string, any>): void;
}>();

const formData = ref<Record<string, any>>({});

// 初始化表单数据，并在父组件直接修改字段时保持同步
watch(
  () => props.modelValue,
  (val) => {
    // Shallow field comparison to break circular loop: second watcher emits up,
    // parent echoes the same object back; if content hasn't changed, skip the assign.
    const cur = formData.value;
    const next = val ?? {};
    const keys = Object.keys({ ...next, ...cur });
    const hasChange = keys.some((k) => (next as any)[k] !== (cur as any)[k]);
    if (hasChange) {
      formData.value = { ...next };
    }
  },
  { immediate: true, deep: true }
);

function emitFormData(next: Record<string, any>) {
  emit("update:modelValue", { ...next });
}

function setFieldValue(name: string, value: any) {
  const currentValue = formData.value[name];
  if (currentValue === value) {
    return;
  }

  const next = {
    ...formData.value,
    [name]: value,
  };
  formData.value = next;
  emitFormData(next);
}

// 字段可见性
const visiblePasswords = ref<Record<string, boolean>>({});
const tagInput = ref<Record<string, string>>({});

// 生成智能默认值
function generateAutoValue(field: FieldConfig) {
  if (!field.autoSuggest) return;

  switch (field.autoSuggest.type) {
    case "timestamp":
      setFieldValue(field.name, new Date().toISOString());
      break;
    case "uuid":
      setFieldValue(field.name, crypto.randomUUID());
      break;
    case "random":
      setFieldValue(field.name, Math.random().toString(36).substring(2));
      break;
  }
}

// 切换密码可见性
function togglePasswordVisibility(name: string) {
  visiblePasswords.value[name] = !visiblePasswords.value[name];
}

// 添加标签
function addTag(fieldName: string, tag: string) {
  if (!tag.trim()) return;

  const currentTags = Array.isArray(formData.value[fieldName]) ? [...formData.value[fieldName]] : [];

  if (!currentTags.includes(tag)) {
    currentTags.push(tag);
    setFieldValue(fieldName, currentTags);
  }
}

// 移除标签
function removeTag(fieldName: string, index: number) {
  const currentTags = Array.isArray(formData.value[fieldName]) ? [...formData.value[fieldName]] : [];
  currentTags.splice(index, 1);
  setFieldValue(fieldName, currentTags);
}

// 验证表单
const errors = ref<Record<string, string>>({});

// 单字段验证（失焦时调用）
function validateField(name: string) {
  const field = props.fields.find((f) => f.name === name);
  if (!field) return;

  if (field.required && !formData.value[name]) {
    errors.value[name] = `${field.label}为必填项`;
    return;
  }

  if (field.type === "number" && formData.value[name] !== undefined) {
    const value = Number(formData.value[name]);
    if (field.min !== undefined && value < field.min) {
      errors.value[name] = `不能小于${field.min}`;
      return;
    }
    if (field.max !== undefined && value > field.max) {
      errors.value[name] = `不能大于${field.max}`;
      return;
    }
  }

  // 通过则清除该字段错误
  delete errors.value[name];
}

function validate(): boolean {
  errors.value = {};
  let isValid = true;

  for (const field of props.fields) {
    if (field.required && !formData.value[field.name]) {
      errors.value[field.name] = `${field.label}为必填项`;
      isValid = false;
    }

    if (field.type === "number" && formData.value[field.name] !== undefined) {
      const value = Number(formData.value[field.name]);
      if (field.min !== undefined && value < field.min) {
        errors.value[field.name] = `不能小于${field.min}`;
        isValid = false;
      }
      if (field.max !== undefined && value > field.max) {
        errors.value[field.name] = `不能大于${field.max}`;
        isValid = false;
      }
    }
  }

  return isValid;
}

// 清除验证错误（获得焦点时）
function clearError(name: string) {
  delete errors.value[name];
}

// 重置表单
function resetForm() {
  const next: Record<string, any> = {};
  for (const field of props.fields) {
    if (field.type === "tags") {
      next[field.name] = Array.isArray(field.defaultValue)
        ? field.defaultValue
        : [];
      continue;
    }
    next[field.name] = field.defaultValue ?? "";
  }
  formData.value = next;
  emitFormData(next);
  errors.value = {};
}

// 获取输入类型
function getInputType(field: FieldConfig): string {
  if (field.type === "password") {
    return visiblePasswords.value[field.name] ? "text" : "password";
  }
  return field.type;
}

// 暴露方法
defineExpose({
  validate,
  resetForm,
  getData: () => formData.value,
});
</script>

<template>
  <ui-form :model="formData" layout="vertical">
    <ui-form-item
      v-for="field in fields"
      :key="field.name"
      :label="field.help ? undefined : field.label"
      :required="field.required"
      :validate-status="errors[field.name] ? 'error' : undefined"
      :help="errors[field.name]"
    >
      <!-- 带帮助提示的标签 -->
      <template v-if="field.help" #label>
        {{ field.label }}
        <span v-if="field.required" class="text-red-500">*</span>
        <span class="group relative ml-1 inline-flex cursor-help items-center">
          <icon-info-circle class="h-3.5 w-3.5 text-slate-400 transition-colors group-hover:text-slate-600" />
          <span class="pointer-events-none absolute bottom-full left-0 z-10 mb-1.5 w-max max-w-[220px] rounded-lg border border-slate-200 bg-gray-900/90 px-3 py-2 text-xs font-normal leading-relaxed text-white/90 shadow-lg opacity-0 transition-opacity duration-150 group-hover:opacity-100">{{ field.help }}</span>
        </span>
      </template>
      <div class="flex flex-col gap-2">
        <!-- 文本输入 -->
        <template v-if="field.type === 'text'">
          <ui-input
            :model-value="formData[field.name]"
            :placeholder="field.placeholder"
            @update:model-value="setFieldValue(field.name, $event)"
            @focus="clearError(field.name)"
            @blur="validateField(field.name)"
          >
            <template v-if="field.autoSuggest" #suffix>
              <ui-button
                type="text"
                size="small"
                :aria-label="`自动填充 ${field.label}`"
                @click="generateAutoValue(field)"
              >
                <icon-check aria-hidden="true" />
              </ui-button>
            </template>
          </ui-input>
        </template>

        <!-- 密码输入 -->
        <template v-else-if="field.type === 'password'">
          <ui-input
            :model-value="formData[field.name]"
            :type="getInputType(field)"
            :placeholder="field.placeholder"
            @update:model-value="setFieldValue(field.name, $event)"
            @focus="clearError(field.name)"
            @blur="validateField(field.name)"
          >
            <template #suffix>
              <button
                type="button"
                :aria-label="visiblePasswords[field.name] ? `隐藏${field.label}` : `显示${field.label}`"
                :aria-pressed="!!visiblePasswords[field.name]"
                class="relative h-7 w-7 rounded text-slate-500 inline-flex items-center justify-center border-0 bg-transparent cursor-pointer transition-all focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--accent)]/50 hover:text-slate-700 before:absolute before:-inset-[8px] before:content-['']"
                @click="togglePasswordVisibility(field.name)"
              >
                <icon-eye v-if="!visiblePasswords[field.name]" aria-hidden="true" />
                <icon-eye-invisible v-else aria-hidden="true" />
              </button>
            </template>
          </ui-input>
        </template>

        <!-- 文本域 -->
        <template v-else-if="field.type === 'textarea'">
          <ui-textarea
            :model-value="formData[field.name]"
            :placeholder="field.placeholder"
            :rows="field.rows || 4"
            @update:model-value="setFieldValue(field.name, $event)"
            @focus="clearError(field.name)"
            @blur="validateField(field.name)"
          />
        </template>

        <!-- 数字输入 -->
        <template v-else-if="field.type === 'number'">
          <ui-input-number
            :model-value="formData[field.name]"
            :placeholder="field.placeholder"
            :min="field.min"
            :max="field.max"
            class="w-full"
            @update:model-value="setFieldValue(field.name, $event)"
            @focus="clearError(field.name)"
            @blur="validateField(field.name)"
          />
        </template>

        <!-- 选择器 -->
        <template v-else-if="field.type === 'select'">
          <ui-select
            :model-value="formData[field.name]"
            :placeholder="field.placeholder"
            class="w-full"
            :popup-container="'body'"
            @update:model-value="setFieldValue(field.name, $event)"
            @focus="clearError(field.name)"
          >
            <ui-option
              v-for="option in field.options"
              :key="option.value"
              :value="option.value"
            >
              {{ option.label }}
            </ui-option>
          </ui-select>
        </template>

        <!-- 开关 -->
        <template v-else-if="field.type === 'switch'">
          <ui-switch
            :model-value="formData[field.name]"
            class="self-start"
            @update:model-value="setFieldValue(field.name, $event)"
            @change="clearError(field.name)"
          />
        </template>

        <!-- 标签输入 -->
        <template v-else-if="field.type === 'tags'">
          <div class="flex flex-col gap-2">
            <div class="flex min-h-[38px] flex-wrap items-center gap-2 rounded-lg border border-slate-200 bg-white p-2">
              <ui-tag
                v-for="(tag, index) in formData[field.name] ?? []"
                :key="index"
                closable
                @close="removeTag(field.name, index as number)"
              >
                {{ tag }}
              </ui-tag>
              <ui-input
                v-model="tagInput[field.name]"
                :placeholder="field.placeholder || '输入后按回车添加'"
                size="small"
                class="min-w-[100px] flex-1"
                @keyup.enter="(e: KeyboardEvent) => { addTag(field.name, (e.target as HTMLInputElement).value); tagInput[field.name] = ''; }"
              />
            </div>
          </div>
        </template>

        <!-- 描述 -->
        <div v-if="field.description" class="text-xs text-slate-500">{{ field.description }}</div>
      </div>
    </ui-form-item>

    <!-- 插槽：自定义字段 -->
    <slot name="extra-fields" :formData="formData" :errors="errors" />
  </ui-form>
</template>
