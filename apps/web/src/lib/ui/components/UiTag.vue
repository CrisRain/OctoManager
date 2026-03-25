<script setup lang="ts">
import { computed, useAttrs } from "vue";
import { cx, TAG_TONE_CLASS } from "../utils";

interface Props {
  color?: string;
  size?: "small" | "medium";
  closable?: boolean;
}

const props = withDefaults(defineProps<Props>(), {
  color: "gray",
  size: "medium",
  closable: false,
});

const emit = defineEmits<{ close: [event: MouseEvent] }>();
const attrs = useAttrs();

const classes = computed(() =>
  cx(
    "ui-tag glass inline-flex items-center gap-2 rounded-full border px-2.5 py-1 font-semibold tracking-[-0.01em]",
    props.size === "small" ? "text-[0.72rem]" : "text-[0.78rem]",
    TAG_TONE_CLASS[props.color] ?? TAG_TONE_CLASS.gray,
    attrs.class as string,
  ),
);

function handleClose(event: MouseEvent) {
  event.preventDefault();
  event.stopPropagation();
  emit("close", event);
}
</script>

<template>
  <span v-bind="{ ...attrs, class: undefined }" :class="classes">
    <slot name="icon" />
    <slot />
    <button
      v-if="closable"
      type="button"
      class="ml-1 rounded-full bg-black/5 px-1.5 py-0.5 text-current/70 transition hover:bg-black/10 hover:text-current"
      @click="handleClose"
    >
      ×
    </button>
  </span>
</template>
