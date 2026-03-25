<script setup lang="ts">
import { computed } from "vue";

interface Props {
  cancelText?: string;
  submitText?: string;
  submitLoadingText?: string;
  submitDisabled?: boolean;
  submitLoading?: boolean;
  submitVisible?: boolean;
}

const props = withDefaults(defineProps<Props>(), {
  cancelText: "取消",
  submitText: "保存",
  submitLoadingText: "",
  submitDisabled: false,
  submitLoading: false,
  submitVisible: true,
});

const emit = defineEmits<{
  cancel: [];
  submit: [];
}>();

const resolvedSubmitText = computed(() => {
  if (props.submitLoading && props.submitLoadingText) {
    return props.submitLoadingText;
  }
  return props.submitText;
});
</script>

<template>
  <div class="fixed bottom-0 left-0 right-0 z-40 flex items-center justify-end gap-3 border-t border-slate-200 bg-white/80 px-6 py-4 backdrop-blur-md lg:left-60">
    <ui-button size="large" @click="emit('cancel')">
      {{ cancelText }}
    </ui-button>
    <ui-button
      v-if="submitVisible"
      type="primary"
      size="large"
      :disabled="submitDisabled"
      :loading="submitLoading"
      @click="emit('submit')"
    >
      <template #icon><icon-check /></template>
      {{ resolvedSubmitText }}
    </ui-button>
  </div>
</template>
