<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from "vue";
import { useRoute, useRouter } from "vue-router";
import { useAccountTypes } from "@/composables";
import { navRoutes, routeNames, searchRoutes, shortcutRoutes, to, type IconKey } from "@/router/registry";
import { useCommandPaletteStore } from "@/store/command-palette";

import {
  IconDashboard, IconLayers, IconUser, IconEmail, IconSchedule,
  IconRobot, IconFile, IconApps, IconThunderbolt,
  IconSettings, IconMenu, IconSearch, IconPlus, IconClose
} from "@/lib/icons";

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

const route = useRoute();
const router = useRouter();
const { data: accountTypes } = useAccountTypes();
const commandPalette = useCommandPaletteStore();

const mobileOpen = ref(false);
const keySequence = ref<string[]>([]);
const keySequenceTimer = ref<number>();

const iconMap: Record<IconKey, any> = {
  dashboard: IconDashboard,
  layers: IconLayers,
  user: IconUser,
  email: IconEmail,
  schedule: IconSchedule,
  robot: IconRobot,
  apps: IconApps,
  file: IconFile,
  settings: IconSettings,
  thunderbolt: IconThunderbolt,
};

function buildNavItems(): NavItem[] {
  const parents = navRoutes.filter((route) => !route.navParent);
  const childrenByParent = new Map<string, NavChild[]>();

  for (const route of navRoutes) {
    if (!route.navParent) continue;
    const children = childrenByParent.get(route.navParent) ?? [];
    children.push({ to: route.path, label: route.label });
    childrenByParent.set(route.navParent, children);
  }

  return parents.map((route) => ({
    name: route.name,
    to: route.path,
    label: route.label,
    icon: iconMap[route.iconKey ?? "settings"] ?? IconSettings,
    children: childrenByParent.get(route.name),
  }));
}

const navItems = computed<NavItem[]>(() => {
  const baseItems = buildNavItems();
  const accountsItem = baseItems.find((item) => item.name === routeNames.accountsList);
  if (accountsItem) {
    const genericTypes = accountTypes.value.filter((item) => item.category === "generic");
    accountsItem.children = [
      { to: to.accounts.list(), label: "全部账号" },
      ...genericTypes.map((item) => ({
        to: to.accounts.byType(item.key),
        label: item.name,
      })),
      ...(accountsItem.children ?? []).filter((child) => child.to !== to.accounts.list()),
    ];
  }
  return baseItems;
});

const activeNavItem = computed(() => {
  const path = route.path;
  return navItems.value.find((item) => path === item.to || path.startsWith(`${item.to}/`)) ?? null;
});

const activeNavChild = computed(() => {
  const path = route.path;
  return activeNavItem.value?.children?.find(
    (child) => path === child.to || path.startsWith(`${child.to}/`)
  ) ?? null;
});

const currentTitle = computed(() => {
  return activeNavChild.value?.label ?? activeNavItem.value?.label ?? "控制台";
});

function closeMobile() {
  mobileOpen.value = false;
}

const quickCreate = computed(() => {
  if (route.path.startsWith("/jobs")) {
    return {
      label: "新建任务",
      description: "把一个新的自动化任务接入当前流程",
      action: () => router.push(to.jobs.create()),
    };
  }
  if (route.path.startsWith("/agents")) {
    return {
      label: "新建 Agent",
      description: "配置一个持续运行的 Agent",
      action: () => router.push(to.agents.create()),
    };
  }
  if (route.path.startsWith("/triggers")) {
    return {
      label: "新建触发器",
      description: "创建一个新的 Webhook 入口",
      action: () => router.push(to.triggers.create()),
    };
  }
  if (route.path.startsWith("/email-accounts")) {
    return {
      label: "新建邮箱账号",
      description: "接入新的邮箱账号并开始同步",
      action: () => router.push(to.emailAccounts.create()),
    };
  }
  if (route.path.startsWith("/account-types")) {
    return {
      label: "新建账号类型",
      description: "定义新的账号类型和字段结构",
      action: () => router.push(to.accountTypes.create()),
    };
  }
  if (route.path.startsWith("/accounts")) {
    return {
      label: "新建账号",
      description: "添加一个新的账号供任务和 Agent 使用",
      action: () => router.push(to.accounts.create()),
    };
  }

  return {
    label: "新建账号",
    description: "从这里开始新增常用资源",
    action: () => router.push(to.accounts.create()),
  };
});

function openCommandSearch() {
  closeMobile();
  commandPalette.open();
}

function handleQuickCreate() {
  closeMobile();
  quickCreate.value.action();
}

function goToSectionRoot() {
  closeMobile();
  router.push(activeNavItem.value?.to ?? to.dashboard());
}

const baseCommands = computed(() => [
  {
    id: "open-search",
    label: "打开搜索",
    description: "快速搜索任何资源",
    shortcut: "⌘K",
    action: () => {
      commandPalette.open();
    },
  },
  {
    id: "new-resource",
    label: "新建资源",
    description: "创建新的账号、任务等",
    shortcut: "⌘N",
    action: () => {
      router.push(to.accounts.create());
    },
  },
  {
    id: "refresh",
    label: "刷新当前页面",
    description: "重新加载数据",
    shortcut: "⌘R",
    action: () => router.go(0),
  },
]);

const routeCommands = computed(() =>
  searchRoutes.map((item) => ({
    id: `search-${item.name}`,
    label: item.label,
    description: item.type === "page" ? `前往 ${item.label}` : `打开 ${item.label}`,
    keywords: item.keywords,
    action: () => router.push(item.path),
  }))
);

const commands = computed(() => [...baseCommands.value, ...routeCommands.value]);

function handleCommandExecute() {
  commandPalette.close();
}

function handleKeyDown(e: KeyboardEvent) {
  const target = e.target as HTMLElement;
  if (
    target.tagName === "INPUT" ||
    target.tagName === "TEXTAREA" ||
    target.contentEditable === "true"
  ) {
    if (e.key !== "Escape") return;
  }

  const isMac = navigator.platform.toUpperCase().indexOf("MAC") >= 0;
  const modKey = isMac ? e.metaKey : e.ctrlKey;

  if (modKey) {
    switch (e.key.toLowerCase()) {
      case "k":
        e.preventDefault();
        commandPalette.open();
        return;
      case "n":
        e.preventDefault();
        router.push(to.accounts.create());
        return;
      case "/":
        e.preventDefault();
        commandPalette.open();
        return;
      case "r":
        e.preventDefault();
        router.go(0);
        return;
    }
  }

  if (e.key === "Escape") {
    commandPalette.close();
    return;
  }

  clearTimeout(keySequenceTimer.value);
  keySequence.value.push(e.key.toUpperCase());

  keySequenceTimer.value = window.setTimeout(() => {
    const sequence = keySequence.value.join(" then ");
    const shortcut = shortcutRoutes.find((item) => item.key === sequence);
    if (shortcut) {
      router.push(shortcut.path);
    }
    keySequence.value = [];
  }, 500);
}

onMounted(() => {
  window.addEventListener("keydown", handleKeyDown);
});

onUnmounted(() => {
  window.removeEventListener("keydown", handleKeyDown);
  if (keySequenceTimer.value) {
    clearTimeout(keySequenceTimer.value);
  }
});
</script>

<template>
  <div class="flex h-full flex-col bg-white">
    <header class="sticky top-0 z-40 flex items-center gap-2 border-b border-slate-200 bg-white px-4 py-3 lg:hidden">
      <button
        type="button"
        class="inline-flex h-9 w-9 flex-shrink-0 items-center justify-center rounded-lg text-slate-600 transition-colors hover:bg-slate-100 focus-visible:outline-none"
        @click="mobileOpen = true"
      >
        <icon-menu class="h-5 w-5" />
      </button>

      <div class="flex min-w-0 flex-1 items-center gap-2">
        <div class="flex h-7 w-7 flex-shrink-0 items-center justify-center rounded-lg bg-[var(--highlight)] text-[var(--text-primary)]">
          <component :is="activeNavItem?.icon ?? IconLayers" class="h-4 w-4" />
        </div>
        <span class="truncate text-sm font-semibold text-slate-900">{{ currentTitle }}</span>
      </div>

      <button
        type="button"
        class="inline-flex h-9 w-9 flex-shrink-0 items-center justify-center rounded-lg text-slate-500 transition-colors hover:bg-slate-100 focus-visible:outline-none"
        @click="openCommandSearch"
      >
        <icon-search class="h-4.5 w-4.5" />
      </button>

      <button
        type="button"
        class="inline-flex h-9 flex-shrink-0 items-center justify-center gap-1.5 rounded-lg bg-[var(--accent)] px-3 text-xs font-semibold text-white transition-colors hover:opacity-90 focus-visible:outline-none"
        @click="handleQuickCreate"
      >
        <icon-plus class="h-3.5 w-3.5" />
        新建
      </button>
    </header>

    <ui-drawer
      :visible="mobileOpen"
      placement="left"
      :footer="false"
      :header="false"
      popup-container="body"
      class="[--drawer-inline-size:min(88vw,24rem)] [&.ui-drawer]:bg-[var(--sidebar-bg)] [&.ui-drawer]:text-white [&.ui-drawer-body]:p-0"
      @cancel="closeMobile"
    >
      <div class="flex h-full w-full flex-col overflow-y-auto overflow-x-hidden bg-[var(--sidebar-bg)] text-white">
        <div class="flex items-center gap-3 border-b border-white/12 px-4 py-4">
          <div class="flex h-10 w-10 flex-shrink-0 items-center justify-center rounded-lg bg-[var(--highlight)] text-[var(--text-primary)]">
            <icon-layers class="h-5 w-5" />
          </div>
          <div class="min-w-0 flex-1">
            <div class="text-[11px] font-semibold uppercase tracking-[0.24em] text-white/60">control</div>
            <div class="truncate text-base font-extrabold tracking-[-0.02em] text-white">OctoManager</div>
          </div>
          <button
            type="button"
            class="inline-flex h-9 w-9 items-center justify-center rounded-lg text-white/70 transition-colors hover:bg-white/10 focus-visible:outline-none"
            @click="closeMobile"
          >
            <icon-close class="h-4.5 w-4.5" />
          </button>
        </div>

        <div class="flex-1 overflow-y-auto px-3 py-4">
          <AppNavList :items="navItems" @navigate="closeMobile" />
        </div>
      </div>
    </ui-drawer>

    <div class="flex flex-1 overflow-hidden min-h-0">
      <aside class="relative hidden w-60 flex-shrink-0 overflow-hidden bg-[var(--sidebar-bg)] p-4 lg:flex">
        <div class="absolute -left-8 top-10 h-28 w-28 rounded-full bg-white/10" />
        <div class="absolute right-4 top-28 h-20 w-20 rotate-12 rounded-lg bg-white/10" />
        <div class="absolute bottom-8 right-[-1.25rem] h-24 w-24 rounded-full bg-white/12" />
        <div class="relative flex h-full w-full flex-col overflow-y-auto overflow-x-hidden">
          <div class="mb-8 flex items-center gap-3 px-2 py-2">
            <div class="flex h-12 w-12 items-center justify-center rounded-lg bg-[var(--highlight)] text-[var(--text-primary)]">
              <icon-layers class="h-5 w-5" />
            </div>
            <div>
              <div class="text-[11px] font-semibold uppercase tracking-[0.24em] text-white/72">control</div>
              <span class="text-[1.05rem] font-extrabold tracking-[-0.02em] text-white">OctoManager</span>
            </div>
          </div>
          <AppNavList :items="navItems" />
          <div class="mt-6 rounded-lg bg-white/14 p-4 text-white">
            <div class="text-[11px] font-semibold uppercase tracking-[0.24em] text-white/72">quick access</div>
            <div class="mt-2 text-base font-bold tracking-[-0.02em]">Open command search with ⌘K</div>
          </div>
        </div>
      </aside>
      <main class="relative flex-1 overflow-y-auto bg-[var(--page-bg)]">
        <router-view v-slot="{ Component }">
          <transition name="fade-slide" mode="out-in">
            <div class="relative min-h-full">
              <div class="pointer-events-none absolute right-10 top-8 h-24 w-24 rounded-full bg-[var(--accent)]/8" />
              <div class="pointer-events-none absolute left-12 top-28 h-16 w-16 rotate-12 rounded-lg bg-[var(--highlight)]/10" />
              <component :is="Component" />
            </div>
          </transition>
        </router-view>
      </main>
    </div>
    <KeyboardShortcuts
      v-model:open="commandPalette.isOpen"
      v-model:query="commandPalette.query"
      :commands="commands"
      @execute="handleCommandExecute"
    />
  </div>
</template>
