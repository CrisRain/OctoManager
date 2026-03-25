import { defineStore } from "pinia";
import { ref } from "vue";
import { createTrigger, deleteTrigger, fireTrigger, listTriggers, patchTrigger } from "@/api";
import type { Trigger, TriggerCreateInput, TriggerFireInput, TriggerFireResult, TriggerPatchInput } from "@/types";
import { normalizeListResponse } from "@/utils/normalizeListResponse";

export const useTriggersStore = defineStore("triggers", () => {
  const triggers = ref<Trigger[]>([]);
  const loading = ref(false);
  const error = ref<string | null>(null);

  async function fetchTriggers() {
    loading.value = true;
    error.value = null;
    try {
      triggers.value = normalizeListResponse<Trigger>(await listTriggers());
    } catch (e) {
      error.value = e instanceof Error ? e.message : "请求失败";
    } finally {
      loading.value = false;
    }
  }

  async function create(payload: TriggerCreateInput) {
    const result = await createTrigger(payload);
    triggers.value = [result.trigger, ...triggers.value];
    return result;
  }

  async function update(id: number, payload: TriggerPatchInput) {
    const result = await patchTrigger(id, payload);
    triggers.value = triggers.value.map((t) => (t.id === id ? result : t));
    return result;
  }

  async function remove(id: number) {
    await deleteTrigger(id);
    triggers.value = triggers.value.filter((item) => item.id !== id);
  }

  async function fire(id: number, payload?: TriggerFireInput): Promise<TriggerFireResult> {
    return fireTrigger(id, payload);
  }

  return {
    triggers,
    loading,
    error,
    fetchTriggers,
    create,
    update,
    remove,
    fire,
  };
});
