import { computed, onMounted, unref, type Ref } from "vue";
import { storeToRefs } from "pinia";
import { getJobExecutionEventsUrl } from "@/api";
import { useJobsStore } from "@/store";
import type { JobDefinitionCreateInput, JobDefinitionPatchInput } from "@/types";
import { useAsyncAction } from "./useAsyncAction";
import { useEventStream } from "./useEventStream";

export function useJobDefinitions() {
  const store = useJobsStore();
  const { jobDefinitions, loadingDefinitions, errorDefinitions } = storeToRefs(store);

  async function refresh() {
    await store.fetchJobDefinitions();
  }

  onMounted(() => { void refresh(); });

  return { data: jobDefinitions, loading: loadingDefinitions, error: errorDefinitions, refresh };
}

export function useJobExecutions(jobId?: number) {
  const store = useJobsStore();
  const { jobExecutions, loadingExecutions, errorExecutions } = storeToRefs(store);

  const filteredExecutions = computed(() => {
    if (!jobId) return jobExecutions.value;
    return jobExecutions.value.filter(e => e.job_definition_id === jobId);
  });

  async function refresh() {
    await store.fetchJobExecutions();
  }

  onMounted(() => { void refresh(); });

  return { data: filteredExecutions, loading: loadingExecutions, error: errorExecutions, refresh };
}

export function useCreateJobDefinition() {
  const store = useJobsStore();
  return useAsyncAction((payload: JobDefinitionCreateInput) => store.createDefinition(payload));
}

export function usePatchJobDefinition() {
  const store = useJobsStore();
  return useAsyncAction((id: number, payload: JobDefinitionPatchInput) =>
    store.patchDefinition(id, payload),
  );
}

export function useDeleteJobDefinition() {
  const store = useJobsStore();
  return useAsyncAction((id: number) => store.deleteDefinition(id));
}

export function useEnqueueJobExecution() {
  const store = useJobsStore();
  return useAsyncAction((id: number) => store.enqueueExecution(id));
}

const executionEventNames = ["log", "progress", "result", "state", "error"];

type MaybeRef<T> = T | Ref<T>;

export function useJobExecutionStream(executionId: MaybeRef<number | null>) {
  const streamUrl = computed(() => {
    const id = unref(executionId);
    return id ? getJobExecutionEventsUrl(id) : null;
  });

  return useEventStream(streamUrl, {
    eventNames: executionEventNames,
    closeOn: ["state"],
  });
}
