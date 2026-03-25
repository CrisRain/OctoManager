<script setup lang="ts">
import { computed, useAttrs, useSlots } from "vue";

defineOptions({ inheritAttrs: false });
import { cx } from "../utils";
import UiButton from "./UiButton.vue";

interface Props {
  visible?: boolean;
  title?: string;
  footer?: boolean | string;
  okText?: string;
  cancelText?: string;
  okLoading?: boolean;
  closable?: boolean;
  maskClosable?: boolean;
}

const props = withDefaults(defineProps<Props>(), {
  visible: false,
  title: "",
  footer: true,
  okText: "确定",
  cancelText: "取消",
  okLoading: false,
  closable: true,
  maskClosable: true,
});

const emit = defineEmits<{
  "update:visible": [value: boolean];
  ok: [];
  cancel: [];
  close: [];
}>();

const attrs = useAttrs();
const slots = useSlots();
const hasTitle = computed(() => Boolean(props.title || slots.title));

function close() {
  emit("update:visible", false);
  emit("cancel");
  emit("close");
}

function onBackdropClick(event: MouseEvent) {
  if (props.maskClosable && event.target === event.currentTarget) close();
}
</script>

<template>
  <Teleport to="body">
    <Transition name="modal-backdrop">
      <div
        v-if="visible"
        v-bind="{ ...attrs, class: undefined }"
        :class="cx('ui-modal fixed inset-0 z-modal flex items-center justify-center p-4', attrs.class as string)"
        style="background: var(--modal-overlay); backdrop-filter: blur(10px) saturate(150%); -webkit-backdrop-filter: blur(10px) saturate(150%)"
        @click="onBackdropClick"
      >
        <Transition name="modal-panel" appear>
          <div
              v-if="visible"
              class="ui-modal-simple w-full max-h-[90vh] overflow-hidden rounded-xl border border-slate-200 bg-white shadow-xl"
              style="max-inline-size: var(--modal-inline-size);"
            >
            <header
              v-if="hasTitle"
              class="ui-modal-header flex items-center justify-between border-b px-6 py-4"
              style="border-color: var(--surface-panel); background: var(--modal-header-bg);"
            >
              <h3 class="font-display text-[1.05em] font-semibold tracking-[-0.03em] text-slate-900">
                <slot name="title">{{ title }}</slot>
              </h3>
              <button
                v-if="closable"
                type="button"
                class="flex items-center justify-center rounded-full border text-slate-400 transition-all hover:text-slate-700"
                style="inline-size: 2.1em; block-size: 2.1em; border-color: var(--modal-close-btn-border); background: var(--modal-close-btn-bg);"
                @click="close"
              >
                <svg class="h-[1em] w-[1em]" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M18 6 6 18"/><path d="m6 6 12 12"/></svg>
              </button>
            </header>

            <div class="ui-modal-body overflow-auto px-6 py-5" style="max-block-size: var(--modal-body-block-size)">
              <slot />
            </div>

            <footer
              v-if="footer !== false"
              class="ui-modal-footer flex items-center justify-end gap-3 border-t px-6 py-4"
              style="border-color: var(--surface-panel); background: var(--modal-footer-bg);"
            >
              <slot name="footer">
                <UiButton @click="close">{{ cancelText }}</UiButton>
                <UiButton type="primary" :loading="okLoading" @click="emit('ok')">{{ okText }}</UiButton>
              </slot>
            </footer>
          </div>
        </Transition>
      </div>
    </Transition>
  </Teleport>
</template>
