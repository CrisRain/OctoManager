import { onMounted } from "vue";
import { storeToRefs } from "pinia";
import { useTriggersStore } from "@/store";
import type { TriggerFireInput, TriggerFireResult } from "@/types";
import { useAsyncAction } from "./useAsyncAction";

export function useTriggers() {
  const store = useTriggersStore();
  const { triggers, loading, error } = storeToRefs(store);

  async function refresh() {
    await store.fetchTriggers();
  }

  onMounted(() => { void refresh(); });

  return { data: triggers, loading, error, refresh };
}

export function useCreateTrigger() {
  const store = useTriggersStore();
  return useAsyncAction((payload: Parameters<typeof store.create>[0]) => store.create(payload));
}

export function usePatchTrigger() {
  const store = useTriggersStore();
  return useAsyncAction((id: number, payload: Parameters<typeof store.update>[1]) =>
    store.update(id, payload),
  );
}

export function useDeleteTrigger() {
  const store = useTriggersStore();
  return useAsyncAction((id: number) => store.remove(id));
}

export function useFireTrigger() {
  const store = useTriggersStore();
  return useAsyncAction((id: number, payload?: TriggerFireInput): Promise<TriggerFireResult> =>
    store.fire(id, payload),
  );
}
