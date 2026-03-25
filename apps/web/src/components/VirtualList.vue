<script setup lang="ts" generic="T extends Record<string, any>">
import { ref, computed, onMounted, onUnmounted, shallowRef, watch } from 'vue';

const props = withDefaults(defineProps<{
  data: T[];
  itemHeight?: number;
  keyField?: keyof T;
  buffer?: number;
}>(), {
  itemHeight: 24,
  keyField: 'id' as keyof T,
  buffer: 10
});

const wrapperRef = ref<HTMLElement | null>(null);
const scrollerRef = ref<HTMLElement | null>(null);
const scrollTop = ref(0);
const containerHeight = ref(0);

const totalHeight = computed(() => props.data.length * props.itemHeight);

const startIndex = computed(() => {
  const start = Math.floor(scrollTop.value / props.itemHeight) - props.buffer;
  return Math.max(0, start);
});

const endIndex = computed(() => {
  const visibleCount = Math.ceil(containerHeight.value / props.itemHeight);
  const end = startIndex.value + visibleCount + props.buffer * 2;
  return Math.min(props.data.length, end);
});

const visibleData = computed(() => {
  return props.data.slice(startIndex.value, endIndex.value);
});

const offsetY = computed(() => startIndex.value * props.itemHeight);

let animationFrameId: number;

function handleScroll() {
  if (!scrollerRef.value) return;
  const currentScrollTop = scrollerRef.value.scrollTop;
  
  if (animationFrameId) {
    cancelAnimationFrame(animationFrameId);
  }
  
  animationFrameId = requestAnimationFrame(() => {
    scrollTop.value = currentScrollTop;
  });
}

let resizeObserver: ResizeObserver | null = null;

onMounted(() => {
  // Find the closest scrollable parent
  let parent = wrapperRef.value?.parentElement;
  while (parent) {
    const style = window.getComputedStyle(parent);
    if (style.overflowY === 'auto' || style.overflowY === 'scroll') {
      scrollerRef.value = parent;
      break;
    }
    parent = parent.parentElement;
  }

  if (scrollerRef.value) {
    scrollerRef.value.addEventListener('scroll', handleScroll, { passive: true });
    
    resizeObserver = new ResizeObserver((entries) => {
      if (entries[0]) {
        containerHeight.value = entries[0].contentRect.height;
      }
    });
    resizeObserver.observe(scrollerRef.value);
    
    containerHeight.value = scrollerRef.value.clientHeight;
    scrollTop.value = scrollerRef.value.scrollTop;
  }
});

onUnmounted(() => {
  if (scrollerRef.value) {
    scrollerRef.value.removeEventListener('scroll', handleScroll);
  }
  if (resizeObserver) {
    resizeObserver.disconnect();
  }
  if (animationFrameId) {
    cancelAnimationFrame(animationFrameId);
  }
});
</script>

<template>
  <div ref="wrapperRef" class="w-full relative" :style="{ height: totalHeight + 'px' }">
    <div
      class="absolute left-0 right-0 top-0 will-change-transform flex flex-col"
      :style="{ transform: `translateY(${offsetY}px)` }"
    >
      <div v-for="item in visibleData" :key="String(item[keyField as keyof typeof item])" class="flex-shrink-0">
        <slot :item="item" :index="data.indexOf(item)"></slot>
      </div>
    </div>
  </div>
</template>