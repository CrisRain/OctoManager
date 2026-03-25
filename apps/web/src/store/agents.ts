import { defineStore } from "pinia";
import { ref } from "vue";
import { createAgent, deleteAgent, getAgentStatus, listAgents, patchAgent as apiPatchAgent, startAgent, stopAgent } from "@/api";
import type { Agent, AgentCreateInput, AgentPatchInput, AgentStatus } from "@/types";
import { normalizeListResponse } from "@/utils/normalizeListResponse";

export const useAgentsStore = defineStore("agents", () => {
  const agents = ref<Agent[]>([]);
  const statuses = ref<Record<number, AgentStatus>>({});
  const loading = ref(false);
  const error = ref<string | null>(null);

  function patchAgent(id: number, patch: Partial<Agent>) {
    agents.value = agents.value.map((agent) =>
      agent.id === id ? { ...agent, ...patch } : agent
    );
  }

  function syncAgentStatus(status: AgentStatus) {
    patchAgent(status.id, {
      runtime_state: status.runtime_state,
      desired_state: status.desired_state,
      last_error: status.last_error ?? "",
    });
  }

  async function fetchAgents() {
    loading.value = true;
    error.value = null;
    try {
      agents.value = normalizeListResponse<Agent>(await listAgents());
    } catch (e) {
      error.value = e instanceof Error ? e.message : "请求失败";
    } finally {
      loading.value = false;
    }
  }

  async function create(payload: AgentCreateInput) {
    const result = await createAgent(payload);
    agents.value = [result, ...agents.value];
    return result;
  }

  async function update(id: number, payload: AgentPatchInput) {
    const result = await apiPatchAgent(id, payload);
    agents.value = agents.value.map((a) => (a.id === id ? result : a));
    return result;
  }

  async function start(id: number) {
    await startAgent(id);
    patchAgent(id, { desired_state: "running" });
  }

  async function stop(id: number) {
    await stopAgent(id);
    patchAgent(id, { desired_state: "stopped" });
  }

  async function fetchStatus(id: number) {
    const status = await getAgentStatus(id);
    statuses.value = { ...statuses.value, [id]: status };
    syncAgentStatus(status);
    return status;
  }

  async function remove(id: number) {
    await deleteAgent(id);
    agents.value = agents.value.filter((a) => a.id !== id);
  }

  return {
    agents,
    statuses,
    loading,
    error,
    patchAgent,
    fetchAgents,
    create,
    update,
    start,
    stop,
    fetchStatus,
    remove,
  };
});
