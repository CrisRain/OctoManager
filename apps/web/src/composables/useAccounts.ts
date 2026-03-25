import { ref, onMounted } from "vue";
import { storeToRefs } from "pinia";
import { getAccount } from "@/api";
import { useAccountsStore } from "@/store";
import type { Account, AccountExecuteInput, AccountPatchInput } from "@/types";
import { useAsyncAction } from "./useAsyncAction";

export function useAccount(id: number) {
  const data = ref<Account | null>(null);
  const loading = ref(false);
  const error = ref<string | null>(null);

  async function refresh() {
    loading.value = true;
    error.value = null;
    try {
      data.value = await getAccount(id);
    } catch (e) {
      error.value = e instanceof Error ? e.message : "请求失败";
    } finally {
      loading.value = false;
    }
  }

  onMounted(() => { void refresh(); });

  return { data, loading, error, refresh };
}

export function useAccounts() {
  const store = useAccountsStore();
  const { accounts, loading, error } = storeToRefs(store);

  async function refresh() {
    await store.fetchAccounts();
  }

  onMounted(() => { void refresh(); });

  return { data: accounts, loading, error, refresh };
}

export function useCreateAccount() {
  const store = useAccountsStore();
  return useAsyncAction((payload: Parameters<typeof store.create>[0]) => store.create(payload));
}

export function usePatchAccount() {
  const store = useAccountsStore();
  return useAsyncAction((id: number, payload: AccountPatchInput) => store.update(id, payload));
}

export function useExecuteAccount() {
  const store = useAccountsStore();
  return useAsyncAction((id: number, action: string, params: Record<string, unknown> = {}) =>
    store.execute(id, { action, params } as AccountExecuteInput),
  );
}

export function useDeleteAccount() {
  const store = useAccountsStore();
  return useAsyncAction((id: number) => store.remove(id));
}
