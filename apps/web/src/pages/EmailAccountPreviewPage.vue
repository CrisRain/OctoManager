<script setup lang="ts">
import { computed, ref, watch } from "vue";
import { useRoute } from "vue-router";
import { useMessage } from "@/composables";
import {
  useEmailAccounts,
  useEmailMailboxes,
  useEmailMessages,
  useEmailMessage,
  usePreviewMailboxes,
  usePreviewLatestMessage,
} from "@/composables/useEmailAccounts";
import { PageHeader } from "@/components/index";
import { to } from "@/router/registry";

const route = useRoute();
const accountId = Number(route.params.id);
const message = useMessage();

const { data: accounts, loading } = useEmailAccounts();
const account = computed(() => accounts.value.find((item) => item.id === accountId));

const mailboxPattern = ref("");
const selectedMailbox = ref("");
const selectedMessageId = ref("");
const previewOutput = ref("");
const previewLabel = ref("");
const previewGeneratedAt = ref("");
const readerTab = ref("body");

const {
  data: mailboxesData,
  loading: mailboxesLoading,
  error: mailboxesError,
  refresh: refreshMailboxes,
} = useEmailMailboxes(accountId, () => mailboxPattern.value);

const {
  data: messagesData,
  loading: messagesLoading,
  error: messagesError,
  refresh: refreshMessages,
} = useEmailMessages(accountId, () => selectedMailbox.value);

const {
  data: messageData,
  loading: messageLoading,
  error: messageError,
  refresh: refreshMessage,
} = useEmailMessage(accountId, () => selectedMessageId.value);

const previewMailboxes = usePreviewMailboxes();
const previewLatest = usePreviewLatestMessage();

const mailboxItems = computed(() => mailboxesData.value?.items ?? []);
const messageItems = computed(() => messagesData.value?.items ?? []);
const messageDetail = computed(() => messageData.value);
const selectedMailboxItem = computed(
  () => mailboxItems.value.find((item) => item.id === selectedMailbox.value) ?? null,
);
const selectedMessageSummary = computed(
  () => messageItems.value.find((item) => item.id === selectedMessageId.value) ?? null,
);
const selectedFlags = computed(() => {
  if (messageDetail.value?.flags?.length) return messageDetail.value.flags;
  return selectedMessageSummary.value?.flags ?? [];
});
const totalMessages = computed(() => messagesData.value?.total ?? messageItems.value.length);
const headerEntries = computed(() => Object.entries(messageDetail.value?.headers ?? {}));
const readerBody = computed(() => {
  const detail = messageDetail.value;
  if (!detail) return "";
  if (detail.text_body?.trim()) return detail.text_body.trim();
  if (detail.html_body?.trim()) return htmlToText(detail.html_body);
  return "No body content.";
});
const accountStatusText = computed(() => {
  switch (account.value?.status) {
    case "active":
      return "已连接";
    case "pending":
      return "待验证";
    case "inactive":
      return "已停用";
    default:
      return "未知状态";
  }
});
const accountStatusColor = computed(() => {
  switch (account.value?.status) {
    case "active":
      return "green";
    case "pending":
      return "orange";
    case "inactive":
      return "gray";
    default:
      return "gray";
  }
});

watch(selectedMailbox, () => {
  selectedMessageId.value = "";
  readerTab.value = "body";
  if (selectedMailbox.value) void refreshMessages();
});

watch(selectedMessageId, () => {
  readerTab.value = "body";
  if (selectedMessageId.value) void refreshMessage();
});

watch(mailboxItems, (items) => {
  if (!items.length) {
    selectedMailbox.value = "";
    return;
  }
  if (!items.some((item) => item.id === selectedMailbox.value)) {
    selectedMailbox.value = items[0].id;
  }
}, { immediate: true });

watch(messageItems, (items) => {
  if (!items.length) {
    selectedMessageId.value = "";
    return;
  }
  if (!items.some((item) => item.id === selectedMessageId.value)) {
    selectedMessageId.value = items[0].id;
  }
}, { immediate: true });

async function handleSearchMailboxes() {
  await refreshMailboxes();
}

async function handleRefreshWorkspace() {
  await refreshMailboxes();
  if (selectedMailbox.value) await refreshMessages();
  if (selectedMessageId.value) await refreshMessage();
}

function setPreviewPayload(label: string, payload: unknown) {
  previewLabel.value = label;
  previewGeneratedAt.value = new Date().toLocaleString("zh-CN");
  previewOutput.value = JSON.stringify(payload, null, 2);
  readerTab.value = "preview";
}

async function handlePreviewMailboxes() {
  if (!account.value) return;
  try {
    const result = await previewMailboxes.execute({
      config: account.value.config as Record<string, unknown>,
      pattern: mailboxPattern.value,
    });
    setPreviewPayload("文件夹快速预览", result);
  } catch (e) {
    message.error(e instanceof Error ? e.message : "预览失败");
  }
}

async function handlePreviewLatest() {
  if (!account.value) return;
  try {
    const result = await previewLatest.execute({
      config: account.value.config as Record<string, unknown>,
      mailbox: selectedMailbox.value || undefined,
    });
    setPreviewPayload(
      selectedMailboxItem.value
        ? `${selectedMailboxItem.value.name} 的最新邮件预览`
        : "最新邮件预览",
      result,
    );
  } catch (e) {
    message.error(e instanceof Error ? e.message : "预览失败");
  }
}

function formatDate(value?: string, withYear = false) {
  if (!value) return "未知时间";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  const options: Intl.DateTimeFormatOptions = withYear
    ? {
        year: "numeric",
        month: "2-digit",
        day: "2-digit",
        hour: "2-digit",
        minute: "2-digit",
      }
    : {
        month: "2-digit",
        day: "2-digit",
        hour: "2-digit",
        minute: "2-digit",
      };
  return date.toLocaleString("zh-CN", options);
}

function formatSize(size?: number) {
  if (typeof size !== "number" || Number.isNaN(size)) return "大小未知";
  if (size < 1024) return `${size} B`;
  if (size < 1024 * 1024) return `${(size / 1024).toFixed(1)} KB`;
  return `${(size / (1024 * 1024)).toFixed(1)} MB`;
}

function isUnread(flags: string[] = []) {
  return !flags.some((flag) => {
    const normalized = flag.toLowerCase();
    return normalized.includes("seen") || normalized.includes("read");
  });
}

function formatFlag(flag: string) {
  return flag.replace(/^[\\]+/, "");
}

function htmlToText(html: string) {
  return html
    .replace(/<style[\s\S]*?<\/style>/gi, " ")
    .replace(/<script[\s\S]*?<\/script>/gi, " ")
    .replace(/<\/(p|div|li|tr|h1|h2|h3|h4|h5|h6|br)>/gi, "\n")
    .replace(/<[^>]+>/g, " ")
    .replace(/&nbsp;/gi, " ")
    .replace(/&amp;/gi, "&")
    .replace(/&lt;/gi, "<")
    .replace(/&gt;/gi, ">")
    .replace(/\r/g, "")
    .replace(/\n{3,}/g, "\n\n")
    .replace(/[ \t]{2,}/g, " ")
    .trim();
}
</script>

<template>
  <div class="page-shell">
    <PageHeader
      title="邮箱预览"
      icon-bg="linear-gradient(135deg, rgba(234,88,12,0.12), rgba(249,115,22,0.12))"
      icon-color="var(--icon-orange)"
      :back-to="to.emailAccounts.list()"
      back-label="返回邮箱账号列表"
    >
      <template #icon><icon-email /></template>
      <template #subtitle>
        <span v-if="account">{{ account.address }}</span>
      </template>
      <template #actions>
        <div v-if="account" class="flex flex-wrap items-center justify-end gap-2">
          <ui-tag>{{ account.provider || "outlook" }}</ui-tag>
          <ui-tag :color="accountStatusColor">{{ accountStatusText }}</ui-tag>
          <ui-button :loading="mailboxesLoading" @click="handleRefreshWorkspace">
            <template #icon><icon-refresh /></template>
            刷新邮箱
          </ui-button>
          <ui-button :loading="previewLatest.loading.value" type="primary" @click="handlePreviewLatest">
            <template #icon><icon-eye /></template>
            预览最新
          </ui-button>
        </div>
      </template>
    </PageHeader>

    <div v-if="loading" class="flex flex-col items-center justify-center gap-3 rounded-xl border border-dashed px-6 py-16 text-center border-slate-200 bg-white/55 shadow-sm">
      <ui-spin size="2.25em" />
    </div>
    <div v-else-if="!account" class="flex flex-col items-center justify-center gap-3 rounded-xl border border-dashed px-6 py-16 text-center border-slate-200 bg-white/55 shadow-sm">
      <p class="text-sm leading-6 text-slate-500">未找到该邮箱账号。</p>
    </div>

    <!-- Three-pane layout -->
    <div
      v-else
      class="grid min-h-[500px] grid-cols-1 gap-4 lg:grid-cols-[minmax(14em,1fr)_minmax(17em,1.15fr)_minmax(0,2fr)] lg:h-[calc(100svh-14rem)]"
    >
      <!-- Pane 1: Folders -->
      <aside class="flex min-w-0 flex-col overflow-hidden rounded-xl border border-slate-200 bg-white shadow-sm">
        <div class="flex items-center gap-2.5 border-b border-slate-100 px-4 py-3">
          <span class="text-[10px] font-bold uppercase tracking-[0.15em] text-[var(--accent)]">Folders</span>
          <h3 class="flex-1 text-sm font-semibold text-slate-800">邮箱文件夹</h3>
          <span class="rounded-full border border-slate-200 bg-slate-50 px-2 py-0.5 text-xs font-semibold text-slate-500">{{ mailboxItems.length }}</span>
        </div>

        <div class="flex items-center gap-2 border-b border-slate-100 px-3 py-2">
          <ui-input
            v-model="mailboxPattern"
            allow-clear
            size="small"
            placeholder="筛选，例如 *alert*"
            class="flex-1"
            @press-enter="handleSearchMailboxes"
          >
            <template #prefix><icon-search /></template>
          </ui-input>
          <ui-button size="small" type="secondary" :loading="mailboxesLoading" @click="handleSearchMailboxes">筛选</ui-button>
        </div>

        <div class="min-h-0 flex-1 overflow-y-auto">
          <div v-if="mailboxesLoading" class="flex flex-col items-center justify-center gap-3 py-12 text-sm text-slate-500">
            <ui-spin /><span>正在读取文件夹…</span>
          </div>
          <div v-else-if="mailboxesError" class="flex flex-col items-center justify-center gap-3 px-4 py-12 text-center text-sm text-red-600">
            <p>{{ mailboxesError }}</p>
            <ui-button size="small" @click="handleSearchMailboxes">重试</ui-button>
          </div>
          <div v-else-if="!mailboxItems.length" class="flex flex-col items-center justify-center gap-2 py-12 text-sm text-slate-400">
            <icon-folder class="h-8 w-8 opacity-40" />
            <span>{{ mailboxPattern ? "没有匹配的文件夹" : "暂无文件夹" }}</span>
          </div>
          <button
            v-for="item in mailboxItems"
            :key="item.id"
            type="button"
            class="flex w-full items-center gap-3 px-4 py-3 text-left transition-colors hover:bg-slate-50 focus-visible:outline-none"
            :class="item.id === selectedMailbox ? 'bg-[var(--accent)]/8' : ''"
            @click="selectedMailbox = item.id"
          >
            <icon-folder
              class="h-4 w-4 flex-shrink-0"
              :class="item.id === selectedMailbox ? 'text-[var(--accent)]' : 'text-slate-400'"
            />
            <span
              class="flex-1 truncate text-sm"
              :class="item.id === selectedMailbox ? 'font-semibold text-[var(--accent)]' : 'font-medium text-slate-700'"
            >{{ item.name }}</span>
          </button>
        </div>

        <div class="border-t border-slate-100 p-3">
          <ui-button size="small" class="w-full justify-center" :loading="previewMailboxes.loading.value" @click="handlePreviewMailboxes">
            文件夹预览
          </ui-button>
        </div>
      </aside>

      <!-- Pane 2: Message list -->
      <section class="flex min-w-0 flex-col overflow-hidden rounded-xl border border-slate-200 bg-white shadow-sm">
        <div class="flex items-center gap-2.5 border-b border-slate-100 px-4 py-3">
          <span class="text-[10px] font-bold uppercase tracking-[0.15em] text-[var(--accent)]">Messages</span>
          <h3 class="flex-1 truncate text-sm font-semibold text-slate-800">{{ selectedMailboxItem?.name || "邮件列表" }}</h3>
          <span class="rounded-full border border-slate-200 bg-slate-50 px-2 py-0.5 text-xs font-semibold text-slate-500">{{ totalMessages }}</span>
        </div>

        <div class="flex items-center justify-between gap-2 border-b border-slate-100 px-4 py-2 text-xs text-slate-400">
          <span>{{ selectedMailboxItem ? `${messageItems.length} / ${totalMessages} 封已载入` : "请先选择文件夹" }}</span>
          <ui-button v-if="selectedMailbox" size="mini" type="text" :loading="messagesLoading" @click="refreshMessages">
            <template #icon><icon-refresh /></template>
          </ui-button>
        </div>

        <div class="min-h-0 flex-1 overflow-y-auto divide-y divide-slate-100">
          <div v-if="!selectedMailbox" class="flex flex-col items-center justify-center gap-2 py-16 text-sm text-slate-400">
            <icon-email class="h-8 w-8 opacity-40" />
            <span>选择左侧文件夹查看邮件</span>
          </div>
          <div v-else-if="messagesLoading" class="flex flex-col items-center justify-center gap-3 py-12 text-sm text-slate-500">
            <ui-spin /><span>正在拉取邮件列表…</span>
          </div>
          <div v-else-if="messagesError" class="flex flex-col items-center justify-center gap-3 px-4 py-12 text-center text-sm text-red-600">
            <p>{{ messagesError }}</p>
            <ui-button size="small" @click="refreshMessages">重试</ui-button>
          </div>
          <div v-else-if="!messageItems.length" class="flex flex-col items-center justify-center gap-2 py-16 text-sm text-slate-400">
            <icon-email class="h-8 w-8 opacity-40" />
            <span>这个文件夹没有邮件</span>
          </div>
          <button
            v-for="item in messageItems"
            :key="item.id"
            type="button"
            class="flex w-full items-start gap-3 px-4 py-3.5 text-left transition-colors hover:bg-slate-50 focus-visible:outline-none focus-visible:ring-inset focus-visible:ring-2 focus-visible:ring-[var(--accent)]/50"
            :class="item.id === selectedMessageId ? 'bg-slate-50' : ''"
            @click="selectedMessageId = item.id"
          >
            <span
              class="mt-2 h-2 w-2 flex-shrink-0 rounded-full"
              :class="isUnread(item.flags) ? 'bg-[var(--accent)]' : 'bg-transparent'"
            />
            <div class="min-w-0 flex-1">
              <div class="flex items-baseline justify-between gap-2">
                <span
                  class="truncate text-sm"
                  :class="isUnread(item.flags) ? 'font-semibold text-slate-900' : 'font-medium text-slate-600'"
                >{{ item.from || "未知发件人" }}</span>
                <span class="flex-shrink-0 text-xs text-slate-400">{{ formatDate(item.date) }}</span>
              </div>
              <div
                class="mt-0.5 truncate text-sm"
                :class="isUnread(item.flags) ? 'font-medium text-slate-700' : 'text-slate-500'"
              >{{ item.subject || "(no subject)" }}</div>
              <div class="mt-0.5 flex items-center justify-between gap-2 text-xs text-slate-400">
                <span class="truncate">{{ item.to || "—" }}</span>
                <span class="flex-shrink-0">{{ formatSize(item.size) }}</span>
              </div>
            </div>
          </button>
        </div>
      </section>

      <!-- Pane 3: Reader -->
      <section class="flex min-w-0 flex-col overflow-hidden rounded-xl border border-slate-200 bg-white shadow-sm">
        <!-- Empty: nothing selected -->
        <template v-if="!selectedMessageSummary && !messageDetail">
          <div class="flex flex-1 flex-col items-center justify-center gap-2 text-slate-400">
            <icon-eye class="h-10 w-10 opacity-30" />
            <span class="text-sm">从邮件列表选择一封邮件开始阅读</span>
          </div>
        </template>

        <template v-else>
          <!-- Message metadata header -->
          <div class="border-b border-slate-100 px-5 py-4">
            <h2 class="mb-3 text-lg font-bold leading-snug tracking-tight text-slate-900">
              {{ messageDetail?.subject || selectedMessageSummary?.subject || "(no subject)" }}
            </h2>
            <div class="grid grid-cols-[3.5rem_1fr] gap-x-3 gap-y-1.5 text-sm">
              <span class="self-center text-xs font-semibold uppercase tracking-[0.08em] text-slate-400">发件人</span>
              <span class="truncate text-slate-700">{{ messageDetail?.from || selectedMessageSummary?.from || "—" }}</span>
              <span class="self-center text-xs font-semibold uppercase tracking-[0.08em] text-slate-400">收件人</span>
              <span class="truncate text-slate-700">{{ messageDetail?.to || selectedMessageSummary?.to || "—" }}</span>
              <template v-if="messageDetail?.cc">
                <span class="self-center text-xs font-semibold uppercase tracking-[0.08em] text-slate-400">抄送</span>
                <span class="truncate text-slate-700">{{ messageDetail.cc }}</span>
              </template>
              <span class="self-center text-xs font-semibold uppercase tracking-[0.08em] text-slate-400">时间</span>
              <span class="text-xs text-slate-500">{{ formatDate(messageDetail?.date || selectedMessageSummary?.date, true) }}</span>
            </div>
            <div v-if="selectedFlags.length" class="mt-3 flex flex-wrap gap-2">
              <ui-tag v-for="flag in selectedFlags" :key="flag" size="small">{{ formatFlag(flag) }}</ui-tag>
            </div>
          </div>

          <!-- Tabs -->
          <ui-tabs
            v-model:active-key="readerTab"
            class="flex min-h-0 flex-1 flex-col overflow-hidden [&_.ui-tabs-content]:flex-1 [&_.ui-tabs-content]:min-h-0 [&_.ui-tabs-content]:overflow-hidden [&_.ui-tabs-content-list]:h-full [&_.ui-tabs-pane]:h-full [&_.ui-tabs-pane]:p-0"
          >
            <ui-tab-pane key="body" title="阅读视图">
              <div class="h-full overflow-y-auto p-5">
                <div v-if="messageLoading" class="flex flex-col items-center justify-center gap-3 py-12 text-sm text-slate-500">
                  <ui-spin /><span>正在加载邮件正文…</span>
                </div>
                <div v-else-if="messageError" class="flex flex-col items-center justify-center gap-3 py-12 text-center text-sm text-red-600">
                  <p>{{ messageError }}</p>
                  <ui-button size="small" @click="refreshMessage">重新拉取</ui-button>
                </div>
                <pre v-else-if="messageDetail" class="m-0 whitespace-pre-wrap break-words text-sm leading-7 text-slate-700">{{ readerBody }}</pre>
                <div v-else class="flex flex-col items-center justify-center gap-2 py-12 text-sm text-slate-400">
                  <icon-email class="h-8 w-8 opacity-40" />
                  <span>选中邮件后可在这里查看正文</span>
                </div>
              </div>
            </ui-tab-pane>

            <ui-tab-pane key="html" title="HTML">
              <div class="h-full overflow-y-auto p-5">
                <pre v-if="messageDetail?.html_body" class="m-0 h-full whitespace-pre-wrap break-words rounded-xl border border-slate-800 bg-slate-900 p-4 font-mono text-xs leading-7 text-slate-300">{{ messageDetail.html_body }}</pre>
                <div v-else class="flex flex-col items-center justify-center gap-2 py-12 text-sm text-slate-400">
                  <icon-file class="h-8 w-8 opacity-40" />
                  <span>这封邮件没有 HTML 内容</span>
                </div>
              </div>
            </ui-tab-pane>

            <ui-tab-pane key="headers" title="Headers">
              <div class="h-full overflow-y-auto p-5">
                <div v-if="headerEntries.length" class="overflow-hidden rounded-xl border border-slate-200 bg-white">
                  <div
                    v-for="[key, value] in headerEntries"
                    :key="key"
                    class="grid grid-cols-[minmax(10em,18ch)_minmax(0,1fr)] gap-4 border-b border-slate-100 px-4 py-3 last:border-b-0 max-md:grid-cols-1"
                  >
                    <span class="text-xs font-semibold tracking-wider text-slate-500">{{ key }}</span>
                    <span class="break-all font-mono text-xs text-slate-800">{{ value }}</span>
                  </div>
                </div>
                <div v-else class="flex flex-col items-center justify-center gap-2 py-12 text-sm text-slate-400">
                  <icon-info-circle class="h-8 w-8 opacity-40" />
                  <span>暂无可展示的邮件头</span>
                </div>
              </div>
            </ui-tab-pane>

            <ui-tab-pane key="preview" title="快速预览">
              <div class="h-full overflow-y-auto p-5">
                <div v-if="previewOutput" class="flex flex-col gap-4">
                  <div class="flex items-center justify-between gap-3">
                    <span class="text-[10px] font-bold uppercase tracking-[0.15em] text-[var(--accent)]">Developer Snapshot</span>
                    <span class="text-sm font-semibold text-slate-900">{{ previewLabel || "快速预览输出" }}</span>
                    <span class="text-xs text-slate-400">{{ previewGeneratedAt }}</span>
                  </div>
                  <pre class="m-0 whitespace-pre-wrap break-words rounded-xl border border-slate-800 bg-slate-900 p-4 font-mono text-xs leading-7 text-slate-300">{{ previewOutput }}</pre>
                </div>
                <div v-else class="flex flex-col items-center justify-center gap-2 py-12 text-sm text-slate-400">
                  <icon-code-block class="h-8 w-8 opacity-40" />
                  <span>点击顶部「预览最新」后，这里会显示原始输出</span>
                </div>
              </div>
            </ui-tab-pane>
          </ui-tabs>
        </template>
      </section>
    </div>
  </div>
</template>
