<script setup lang="ts">
import { computed, useSlots } from "vue";
import { useRouter } from "vue-router";

const props = defineProps<{
  title: string;
  subtitle?: string;
  /** 用于标题图标背景的 CSS gradient / color 字符串 */
  iconBg?: string;
  /** 用于标题图标前景色 */
  iconColor?: string;
  /** 返回按钮跳转路径，不传则不显示返回按钮 */
  backTo?: string;
  /** 返回按钮 tooltip 文字 */
  backLabel?: string;
}>();

const router = useRouter();
const slots = useSlots();
const hasSubtitle = computed(() => Boolean(props.subtitle || slots.subtitle?.().length));
const resolvedIconBg = computed(
  () =>
    props.iconBg ??
    "linear-gradient(135deg, rgba(10, 132, 255, 0.18) 0%, rgba(94, 92, 230, 0.16) 100%)"
);
const resolvedIconColor = computed(() => props.iconColor ?? "var(--accent)");
const backAriaLabel = computed(() =>
  props.backLabel ? `返回：${props.backLabel}` : "返回上一页"
);
</script>

<template>
  <div class="mb-8 flex items-start justify-between gap-4 max-md:flex-col max-md:items-start motion-safe:animate-[slide-up_0.3s_ease-out]">
    <div class="flex min-w-0 flex-1 items-start gap-4">
      <!-- Back button: h-9 visual but 44px tap zone via before: -->
      <button type="button"
        v-if="backTo"
        :aria-label="backAriaLabel"
        class="relative mt-1 inline-flex h-9 w-9 flex-shrink-0 items-center justify-center rounded-lg border border-slate-200 bg-white text-slate-500 shadow-sm transition-all hover:bg-slate-50 hover:text-slate-900 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--accent)]/50 focus-visible:ring-offset-1 before:absolute before:-inset-[4px] before:content-['']"
        @click="router.push(backTo)"
      >
        <icon-left aria-hidden="true" />
        <span v-if="backLabel" class="page-header__tooltip pointer-events-none absolute bottom-full left-1/2 mb-2 -translate-x-1/2 translate-y-1.5 whitespace-nowrap rounded-full border border-slate-200 bg-gray-900/90 px-2.5 py-1 text-xs text-white/90 opacity-0 transition-[opacity,transform] duration-[160ms]">{{ backLabel }}</span>
      </button>

      <div
        v-if="$slots.icon"
        class="mt-0.5 flex h-12 w-12 flex-shrink-0 items-center justify-center rounded-xl border border-slate-200 shadow-sm max-md:h-10 max-md:w-10 [&>svg]:h-6 [&>svg]:w-6 max-md:[&>svg]:h-5 max-md:[&>svg]:w-5"
        :style="{ background: resolvedIconBg, color: resolvedIconColor }"
        aria-hidden="true"
      >
        <slot name="icon" />
      </div>

      <div class="flex min-w-0 flex-col gap-1.5">
        <h1 class="text-2xl font-bold tracking-tight text-slate-900 max-md:text-xl">
          {{ title }}
        </h1>
        <p v-if="hasSubtitle" class="max-w-[72ch] text-[14px] leading-relaxed text-slate-600">
          <slot name="subtitle">{{ subtitle }}</slot>
        </p>
      </div>
    </div>

    <div v-if="$slots.actions" class="flex flex-shrink-0 flex-wrap items-center justify-end gap-2.5 max-md:w-full max-md:justify-start">
      <slot name="actions" />
    </div>
  </div>
</template>
