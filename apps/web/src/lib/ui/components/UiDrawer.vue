<script setup lang="ts">
defineOptions({
  inheritAttrs: false,
});

import { computed, useAttrs } from "vue";
import { cx } from "../utils";

interface Props {
  visible?: boolean;
  placement?: "left" | "right" | "top" | "bottom";
  closable?: boolean;
  header?: boolean;
  footer?: boolean;
  title?: string;
  maskClosable?: boolean;
  popupContainer?: string;
}

const props = withDefaults(defineProps<Props>(), {
  visible: false,
  placement: "right",
  closable: true,
  header: true,
  footer: false,
  title: "",
  maskClosable: true,
  popupContainer: "body",
});

const emit = defineEmits<{
  "update:visible": [value: boolean];
  cancel: [];
  close: [];
}>();

const attrs = useAttrs();

const panelClass = computed(() => {
  if (props.placement === "left") return "h-full max-w-full rounded-r-2xl";
  if (props.placement === "top") return "w-full rounded-b-2xl";
  if (props.placement === "bottom") return "mt-auto w-full rounded-t-2xl";
  return "ml-auto h-full max-w-full rounded-l-2xl";
});

const panelStyle = computed(() => {
  if (props.placement === "top" || props.placement === "bottom") {
    return { blockSize: "var(--drawer-block-size)" };
  }
  return { inlineSize: "var(--drawer-inline-size)" };
});

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
  <Teleport :to="popupContainer">
    <Transition name="drawer-backdrop">
      <div
        v-if="visible"
        v-bind="{ ...attrs, class: undefined }"
        :class="cx('ui-drawer-container fixed inset-0 z-[450] flex', attrs.class as string)"
        style="background: var(--drawer-overlay)"
        @click="onBackdropClick"
      >
        <Transition :name="`drawer-slide-${placement}`" appear>
          <section
            v-if="visible"
            :class="cx('ui-drawer flex w-full max-w-full flex-col bg-white shadow-none', panelClass)"
            :style="panelStyle"
            @click.stop
          >
            <header
              v-if="header"
              class="ui-drawer-header flex items-center justify-between border-b border-slate-100 px-5 py-4"
            >
              <h3 class="text-base font-semibold text-slate-900">
                <slot name="title">{{ title }}</slot>
              </h3>
              <button
                v-if="closable"
                type="button"
                class="flex items-center justify-center w-7 h-7 rounded-lg text-slate-400 transition-colors hover:bg-slate-100 hover:text-slate-600"
                @click="close"
              >
                <svg class="h-[1em] w-[1em]" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M18 6 6 18"/><path d="m6 6 12 12"/></svg>
              </button>
            </header>

            <div class="ui-drawer-body min-h-0 flex-1 overflow-auto p-5">
              <slot />
            </div>

            <footer v-if="footer" class="border-t border-slate-100 px-5 py-4">
              <slot name="footer" />
            </footer>
          </section>
        </Transition>
      </div>
    </Transition>
  </Teleport>
</template>
