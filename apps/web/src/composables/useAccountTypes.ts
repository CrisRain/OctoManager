import { onMounted } from "vue";
import { storeToRefs } from "pinia";
import { useAccountTypesStore } from "@/store";
import type { AccountTypeCreateInput } from "@/types";
import { useAsyncAction } from "./useAsyncAction";

export function useAccountTypes() {
  const store = useAccountTypesStore();
  const { accountTypes, loading, error } = storeToRefs(store);

  async function refresh() {
    await store.fetchAccountTypes();
  }

  onMounted(() => { void refresh(); });

  return { data: accountTypes, loading, error, refresh };
}

export function useCreateAccountType() {
  const store = useAccountTypesStore();
  return useAsyncAction((payload: AccountTypeCreateInput) => store.create(payload));
}

export function usePatchAccountType() {
  const store = useAccountTypesStore();
  return useAsyncAction((key: string, payload: Parameters<typeof store.patch>[1]) =>
    store.patch(key, payload),
  );
}

export function useDeleteAccountType() {
  const store = useAccountTypesStore();
  return useAsyncAction((key: string) => store.remove(key));
}
