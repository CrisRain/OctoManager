<script setup lang="ts">
import { useRoute } from "vue-router";

interface NavChild {
  to: string;
  label: string;
}

interface NavItem {
  name: string;
  to: string;
  label: string;
  icon: any;
  children?: NavChild[];
}

const props = defineProps<{
  items: NavItem[];
}>();

const emit = defineEmits<{
  (e: "navigate"): void;
}>();

const route = useRoute();

const isActive = (to: string): boolean => {
  return route.path === to || (to !== "/" && route.path.startsWith(`${to}/`));
};

const isChildActive = (to: string): boolean => {
  return route.path === to || route.path.startsWith(`${to}/`);
};
</script>

<template>
  <nav class="dark-scroll flex w-full flex-1 flex-col gap-2">
    <template v-for="item in props.items" :key="item.to">
      <router-link
        :to="item.to"
        class="group relative flex items-center gap-3 rounded-lg px-4 py-3 text-[14px] font-semibold no-underline transition-all duration-200 hover:scale-[1.02] hover:bg-white/14 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-white focus-visible:ring-offset-2 focus-visible:ring-offset-[var(--sidebar-bg)]"
        :class="isActive(item.to)
          ? 'bg-white text-[var(--accent)]'
          : 'text-[var(--sidebar-text-strong)]'"
        @click="emit('navigate')"
      >
        <span
          v-if="isActive(item.to)"
          class="absolute inset-y-3 left-0 w-1 rounded-r-md bg-[var(--highlight)]"
        />
        <component
          :is="item.icon"
          class="h-[18px] w-[18px] flex-shrink-0 transition-transform duration-200 group-hover:scale-110"
          :class="isActive(item.to) ? 'text-[var(--accent)]' : 'text-[var(--sidebar-icon)]'"
        />
        <span>{{ item.label }}</span>
      </router-link>
      <div v-if="item.children?.length" class="ml-6 flex w-full flex-col gap-1 border-l-2 border-white/20 pl-3">
        <router-link
          v-for="child in item.children"
          :key="child.to"
          :to="child.to"
          class="rounded-md px-3 py-2 text-[13px] font-medium no-underline transition-all duration-200 hover:scale-[1.02] hover:bg-white/12 hover:text-white focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-white focus-visible:ring-offset-2 focus-visible:ring-offset-[var(--sidebar-bg)]"
          :class="isChildActive(child.to) ? 'bg-white/16 text-white' : 'text-[var(--sidebar-text)]'"
          @click="emit('navigate')"
        >
          {{ child.label }}
        </router-link>
      </div>
    </template>
  </nav>
</template>
