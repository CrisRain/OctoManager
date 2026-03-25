import { defineStore } from "pinia";
import { ref } from "vue";
import { createJobDefinition, deleteJobDefinition, enqueueJobExecution, listJobDefinitions, listJobExecutions, patchJobDefinition } from "@/api";
import type { JobDefinition, JobDefinitionCreateInput, JobDefinitionPatchInput, JobExecution } from "@/types";
import { normalizeListResponse } from "@/utils/normalizeListResponse";

export const useJobsStore = defineStore("jobs", () => {
  const jobDefinitions = ref<JobDefinition[]>([]);
  const jobExecutions = ref<JobExecution[]>([]);
  const loadingDefinitions = ref(false);
  const loadingExecutions = ref(false);
  const errorDefinitions = ref<string | null>(null);
  const errorExecutions = ref<string | null>(null);

  async function fetchJobDefinitions() {
    loadingDefinitions.value = true;
    errorDefinitions.value = null;
    try {
      jobDefinitions.value = normalizeListResponse<JobDefinition>(await listJobDefinitions());
    } catch (e) {
      errorDefinitions.value = e instanceof Error ? e.message : "请求失败";
    } finally {
      loadingDefinitions.value = false;
    }
  }

  async function fetchJobExecutions() {
    loadingExecutions.value = true;
    errorExecutions.value = null;
    try {
      jobExecutions.value = normalizeListResponse<JobExecution>(await listJobExecutions());
    } catch (e) {
      errorExecutions.value = e instanceof Error ? e.message : "请求失败";
    } finally {
      loadingExecutions.value = false;
    }
  }

  async function createDefinition(payload: JobDefinitionCreateInput) {
    const result = await createJobDefinition(payload);
    jobDefinitions.value = [result, ...jobDefinitions.value];
    return result;
  }

  async function enqueueExecution(id: number) {
    const result = await enqueueJobExecution(id);
    jobExecutions.value = [result, ...jobExecutions.value];
    return result;
  }

  async function patchDefinition(id: number, payload: JobDefinitionPatchInput) {
    const result = await patchJobDefinition(id, payload);
    jobDefinitions.value = jobDefinitions.value.map((d) => (d.id === id ? result : d));
    return result;
  }

  async function deleteDefinition(id: number) {
    await deleteJobDefinition(id);
    jobDefinitions.value = jobDefinitions.value.filter((d) => d.id !== id);
  }

  return {
    jobDefinitions,
    jobExecutions,
    loadingDefinitions,
    loadingExecutions,
    errorDefinitions,
    errorExecutions,
    fetchJobDefinitions,
    fetchJobExecutions,
    createDefinition,
    patchDefinition,
    deleteDefinition,
    enqueueExecution,
  };
});
