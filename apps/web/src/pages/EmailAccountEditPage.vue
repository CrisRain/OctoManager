<script setup lang="ts">
import { ref, reactive, watch, onMounted } from "vue";
import { useRoute, useRouter } from "vue-router";
import { useMessage } from "@/composables";
import {
  usePatchEmailAccount,
  useBuildAuthorizeURL,
  useExchangeCode,
} from "@/composables/useEmailAccounts";
import { getEmailAccount } from "@/api/email-accounts";
import { FormActionBar, FormPageLayout, PageHeader } from "@/components/index";
import { to } from "@/router/registry";
import type { EmailAccount } from "@/types";

const route = useRoute();
const router = useRouter();
const accountId = Number(route.params.id);
const message = useMessage();

// ── Load by ID ────────────────────────────────────────────────────────────────
const account = ref<EmailAccount | null>(null);
const loading = ref(false);
const loadError = ref<string | null>(null);

onMounted(async () => {
  loading.value = true;
  try {
    account.value = await getEmailAccount(accountId);
  } catch (e) {
    loadError.value = e instanceof Error ? e.message : "加载失败";
  } finally {
    loading.value = false;
  }
});

// ── Form state ────────────────────────────────────────────────────────────────
const form = reactive({ provider: "", status: "active" });

interface ConfigEntry { key: string; val: string; hidden: boolean }
const configEntries = ref<ConfigEntry[]>([]);

const SECRET_RE = /secret|token|password/i;
function isSecret(k: string) { return SECRET_RE.test(k); }

watch(account, (acc) => {
  if (!acc) return;
  form.provider = acc.provider;
  form.status = acc.status;
  configEntries.value = Object.entries(acc.config ?? {}).map(([key, value]) => ({
    key,
    val: typeof value === "string" ? value : JSON.stringify(value),
    hidden: isSecret(key),
  }));
}, { immediate: true });

function addConfigField() {
  configEntries.value.push({ key: "", val: "", hidden: false });
}

function removeConfigField(index: number) {
  configEntries.value.splice(index, 1);
}

function toggleHide(entry: ConfigEntry) {
  entry.hidden = !entry.hidden;
}

function buildConfig(): Record<string, unknown> {
  const out: Record<string, unknown> = {};
  for (const { key, val } of configEntries.value) {
    const k = key.trim();
    if (!k) continue;
    try { out[k] = JSON.parse(val); } catch { out[k] = val; }
  }
  return out;
}

// ── Save ──────────────────────────────────────────────────────────────────────
const patch = usePatchEmailAccount();

async function handleSave() {
  try {
    await patch.execute(accountId, {
      provider: form.provider || undefined,
      status: form.status || undefined,
      config: buildConfig(),
    });
    message.success("已保存");
    router.push(to.emailAccounts.preview(accountId));
  } catch (e) {
    message.error(e instanceof Error ? e.message : "保存失败");
  }
}

// ── OAuth ─────────────────────────────────────────────────────────────────────
const buildAuthorize = useBuildAuthorizeURL();
const exchangeCode = useExchangeCode();
const authorizeURL = ref("");
const authCode = ref("");

async function handleBuildAuthorize() {
  try {
    const result = await buildAuthorize.execute(accountId);
    if (result && "authorize_url" in result) {
      authorizeURL.value = result.authorize_url as string;
    }
  } catch (e) {
    message.error(e instanceof Error ? e.message : "生成授权链接失败");
  }
}

async function handleExchange() {
  if (!authCode.value.trim()) { message.error("请填写授权码"); return; }
  try {
    await exchangeCode.execute(accountId, { code: authCode.value.trim() });
    authCode.value = "";
    message.success("授权码交换成功");
  } catch (e) {
    message.error(e instanceof Error ? e.message : "交换失败");
  }
}
</script>

<template>
  <div class="page-shell">
    <PageHeader
      title="编辑邮箱账号"
      :subtitle="account ? `正在编辑 ${account.address}` : '邮箱账号详情加载中…'"
      icon-bg="linear-gradient(135deg, rgba(234,88,12,0.12), rgba(249,115,22,0.12))"
      icon-color="var(--icon-orange)"
      :back-to="to.emailAccounts.list()"
      back-label="返回邮箱账号列表"
    >
      <template #icon><icon-email /></template>
    </PageHeader>

    <FormPageLayout
      :loading="loading"
      :ready="!!account && !loadError"
      :empty-description="loadError ?? '未找到该邮箱账号'"
    >
      <template #empty-action>
        <ui-button type="primary" @click="router.push(to.emailAccounts.list())">返回邮箱账号列表</ui-button>
      </template>

      <template #main>
        <ui-card>
          <template #title>
            <div class="flex items-center gap-2">
              <icon-edit class="h-4 w-4 text-[var(--accent)]" />
              <span>基本信息</span>
            </div>
          </template>
          <ui-form layout="vertical">
            <ui-form-item label="服务商">
              <ui-input v-model="form.provider" placeholder="例如 gmail / outlook" allow-clear />
            </ui-form-item>
            <ui-form-item label="状态">
              <ui-select v-model="form.status" class="w-full" popup-container="body">
                <ui-option value="active">已激活</ui-option>
                <ui-option value="pending">待验证</ui-option>
                <ui-option value="inactive">已停用</ui-option>
              </ui-select>
            </ui-form-item>
          </ui-form>
        </ui-card>

        <ui-card>
          <template #title>
            <div class="flex items-center justify-between gap-2">
              <div class="flex items-center gap-2">
                <icon-setting class="h-4 w-4 text-[var(--accent)]" />
                <span>配置项</span>
              </div>
              <ui-button size="small" @click="addConfigField">
                <template #icon><icon-plus /></template>
                添加字段
              </ui-button>
            </div>
          </template>

          <div v-if="configEntries.length === 0" class="py-6 text-center text-sm text-slate-400">
            暂无配置项，点击"添加字段"新增
          </div>

          <div v-else class="flex flex-col gap-3">
            <div
              v-for="(entry, idx) in configEntries"
              :key="idx"
              class="flex items-center gap-2"
            >
              <div class="w-36 flex-shrink-0">
                <ui-input
                  v-model="entry.key"
                  placeholder="字段名"
                  class="font-mono text-sm"
                  @change="entry.hidden = isSecret(entry.key)"
                />
              </div>
              <div class="min-w-0 flex-1">
                <ui-input
                  v-model="entry.val"
                  :type="entry.hidden ? 'password' : 'text'"
                  placeholder="值"
                  class="font-mono text-sm"
                >
                  <template v-if="isSecret(entry.key)" #suffix>
                    <button
                      type="button"
                      class="inline-flex h-5 w-5 items-center justify-center rounded text-slate-400 transition-colors hover:text-slate-600"
                      :aria-label="entry.hidden ? '显示' : '隐藏'"
                      @click="toggleHide(entry)"
                    >
                      <icon-eye v-if="entry.hidden" />
                      <icon-eye-invisible v-else />
                    </button>
                  </template>
                </ui-input>
              </div>
              <button
                type="button"
                class="inline-flex h-7 w-7 flex-shrink-0 items-center justify-center rounded-lg text-slate-400 transition-colors hover:bg-red-50 hover:text-red-500"
                aria-label="删除此字段"
                @click="removeConfigField(idx)"
              >
                <icon-delete class="h-4 w-4" />
              </button>
            </div>
          </div>
        </ui-card>

        <ui-card>
          <template #title>
            <div class="flex items-center gap-2">
              <icon-link class="h-4 w-4 text-[var(--accent)]" />
              <span>OAuth 授权</span>
            </div>
          </template>

          <div class="flex flex-wrap gap-3">
            <ui-button :loading="buildAuthorize.loading.value" @click="handleBuildAuthorize">
              <template #icon><icon-link /></template>
              生成授权链接
            </ui-button>
          </div>

          <div v-if="authorizeURL" class="mt-4 rounded-xl border border-sky-200 bg-sky-50/70 p-4">
            <p class="mb-2 text-xs font-semibold uppercase tracking-wider text-slate-500">授权链接（点击在新标签页打开）</p>
            <a
              :href="authorizeURL"
              target="_blank"
              rel="noreferrer"
              class="break-all text-sm font-medium text-[var(--accent)] underline decoration-dotted hover:decoration-solid"
            >{{ authorizeURL }}</a>
          </div>

          <ui-divider class="my-4" />

          <ui-form layout="vertical">
            <ui-form-item label="授权码">
              <div class="flex gap-2">
                <ui-input
                  v-model="authCode"
                  placeholder="粘贴授权码"
                  allow-clear
                  class="flex-1"
                  @keyup.enter="handleExchange"
                />
                <ui-button
                  type="primary"
                  :disabled="!authCode.trim()"
                  :loading="exchangeCode.loading.value"
                  @click="handleExchange"
                >交换授权码</ui-button>
              </div>
            </ui-form-item>
          </ui-form>
        </ui-card>
      </template>

      <template #aside>
        <ui-card class="min-w-0 lg:sticky lg:top-[var(--space-6)]">
          <template #title>
            <div class="flex items-center gap-2">
              <icon-info-circle class="h-4 w-4 text-[var(--accent)]" />
              <span>基本信息</span>
            </div>
          </template>
          <div class="flex flex-col gap-3">
            <div class="flex flex-col gap-1.5 rounded-xl border border-slate-200 bg-slate-50 p-4 shadow-sm">
              <span class="text-xs font-semibold uppercase tracking-wider text-slate-500">邮箱地址</span>
              <span class="break-all text-sm font-medium text-slate-700">{{ account?.address }}</span>
            </div>
            <div class="flex flex-col gap-1.5 rounded-xl border border-slate-200 bg-slate-50 p-4 shadow-sm">
              <span class="text-xs font-semibold uppercase tracking-wider text-slate-500">ID</span>
              <code class="text-sm text-slate-600">#{{ account?.id }}</code>
            </div>
          </div>
        </ui-card>
      </template>

      <template #actions>
        <FormActionBar
          cancel-text="取消"
          submit-text="保存修改"
          submit-loading-text="保存中…"
          :submit-loading="patch.loading.value"
          @cancel="router.push(to.emailAccounts.list())"
          @submit="handleSave"
        />
      </template>
    </FormPageLayout>
  </div>
</template>
