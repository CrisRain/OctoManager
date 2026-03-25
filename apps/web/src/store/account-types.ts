import { defineStore } from "pinia";
import { ref } from "vue";
import { createAccountType, deleteAccountType, listAccountTypes, patchAccountType } from "@/api";
import type { AccountType, AccountTypeCreateInput, AccountTypePatchInput } from "@/types";
import { normalizeListResponse } from "@/utils/normalizeListResponse";

export const useAccountTypesStore = defineStore("accountTypes", () => {
  const accountTypes = ref<AccountType[]>([]);
  const loading = ref(false);
  const error = ref<string | null>(null);

  async function fetchAccountTypes() {
    loading.value = true;
    error.value = null;
    try {
      accountTypes.value = normalizeListResponse<AccountType>(await listAccountTypes());
    } catch (e) {
      error.value = e instanceof Error ? e.message : "请求失败";
    } finally {
      loading.value = false;
    }
  }

  async function create(payload: AccountTypeCreateInput) {
    const result = await createAccountType(payload);
    accountTypes.value = [result, ...accountTypes.value];
    return result;
  }

  async function patch(key: string, payload: AccountTypePatchInput) {
    const result = await patchAccountType(key, payload);
    accountTypes.value = accountTypes.value.map((t) => (t.key === key ? result : t));
    return result;
  }

  async function remove(key: string) {
    await deleteAccountType(key);
    accountTypes.value = accountTypes.value.filter((item) => item.key !== key);
  }

  return {
    accountTypes,
    loading,
    error,
    fetchAccountTypes,
    create,
    patch,
    remove,
  };
});
