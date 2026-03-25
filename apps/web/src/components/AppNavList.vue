<script setup lang="ts">
import { computed } from "vue";
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

const props = withDefaults(defineProps<{
  items: NavItem[];
  variant?: "desktop" | "mobile";
}>(), {
  variant: "desktop",
});

const emit = defineEmits<{
  (e: "navigate"): void;
}>();

const route = useRoute();
const isMobileVariant = computed(() => props.variant === "mobile");

const isActive = (to: string): boolean => {
  return route.path === to || (to !== "/" && route.path.startsWith(`${to}/`));
};

const isChildActive = (to: string): boolean => {
  return route.path === to || route.path.startsWith(`${to}/`);
};

function itemClass(active: boolean) {
  if (isMobileVariant.value) {
    return active
      ? "border-white/45 bg-white text-[var(--accent)] shadow-[0_16px_30px_rgba(15,23,42,0.18)]"
      : "border-white/12 bg-white/10 text-[var(--sidebar-text-strong)] hover:border-white/20 hover:bg-white/16";
  }

  return active
    ? "bg-white text-[var(--accent)]"
    : "text-[var(--sidebar-text-strong)]";
}

function itemIconClass(active: boolean) {
  if (isMobileVariant.value) {
    return active ? "bg-[var(--accent)]/10 text-[var(--accent)]" : "bg-white/12 text-[var(--sidebar-icon)]";
  }

  return active ? "text-[var(--accent)]" : "text-[var(--sidebar-icon)]";
}

function childContainerClass() {
  return isMobileVariant.value
    ? "ml-3 min-w-0 flex flex-col gap-1.5 border-l border-white/15 pl-3"
    : "ml-4 min-w-0 flex flex-col gap-1 border-l-2 border-white/20 pl-3";
}

function childClass(active: boolean) {
  if (isMobileVariant.value) {
    return active
      ? "border-white/20 bg-white/16 text-white shadow-sm"
      : "border-transparent text-[var(--sidebar-text)] hover:border-white/12 hover:bg-white/10 hover:text-white";
  }

  return active
    ? "bg-white/16 text-white"
    : "text-[var(--sidebar-text)]";
}
</script>

<template>
  <nav
    class="dark-scroll flex min-w-0 w-full flex-1 overflow-x-hidden"
    :class="isMobileVariant ? 'flex-col gap-3' : 'flex-col gap-2'"
  >
    <template v-for="item in props.items" :key="item.to">
      <router-link
        :to="item.to"
        class="group relative flex min-w-0 w-full items-center gap-3 no-underline focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-white focus-visible:ring-offset-2 focus-visible:ring-offset-[var(--sidebar-bg)]"
        :class="[
          isMobileVariant
            ? 'rounded-2xl border px-4 py-3.5 transition-[background-color,color,border-color,box-shadow] duration-200'
            : 'rounded-lg px-4 py-3 text-[14px] transition-[background-color,color,box-shadow] duration-200 hover:bg-white/14 hover:shadow-sm',
          itemClass(isActive(item.to)),
        ]"
        @click="emit('navigate')"
      >
        <span
          v-if="isActive(item.to)"
          :class="isMobileVariant ? 'absolute inset-y-4 left-0 w-1 rounded-r-full bg-[var(--highlight)]' : 'absolute inset-y-3 left-0 w-1 rounded-r-md bg-[var(--highlight)]'"
        />
        <div
          :class="[
            'flex flex-shrink-0 items-center justify-center rounded-xl transition-all duration-200',
            isMobileVariant ? 'h-10 w-10' : 'h-8 w-8',
            itemIconClass(isActive(item.to)),
          ]"
        >
          <component
            :is="item.icon"
            class="h-[18px] w-[18px] flex-shrink-0 transition-transform duration-200 group-hover:translate-x-0.5"
          />
        </div>
        <span class="min-w-0 flex-1 truncate">{{ item.label }}</span>
        <span
          v-if="isMobileVariant && item.children?.length"
          class="rounded-full border border-white/14 bg-white/10 px-2 py-0.5 text-[10px] font-semibold uppercase tracking-[0.14em] text-white/70"
        >
          {{ item.children.length }}
        </span>
      </router-link>
      <div v-if="item.children?.length" :class="childContainerClass()">
        <router-link
          v-for="child in item.children"
          :key="child.to"
          :to="child.to"
          class="block min-w-0 no-underline focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-white focus-visible:ring-offset-2 focus-visible:ring-offset-[var(--sidebar-bg)]"
          :class="[
            isMobileVariant
              ? 'rounded-xl border px-3 py-2.5 text-[12px] font-semibold transition-[background-color,color,border-color,box-shadow] duration-200'
              : 'rounded-md px-3 py-2 text-[13px] font-medium transition-[background-color,color,box-shadow] duration-200 hover:bg-white/12 hover:text-white hover:shadow-sm',
            childClass(isChildActive(child.to)),
          ]"
          @click="emit('navigate')"
        >
          <span class="block truncate">{{ child.label }}</span>
        </router-link>
      </div>
    </template>
  </nav>
</template>
