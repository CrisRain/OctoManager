import { ref, computed, onMounted, unref, type Ref } from "vue";
import { storeToRefs } from "pinia";
import { getAgentEventsUrl } from "@/api";
import { useAgentsStore } from "@/store";
import type { AgentStatus } from "@/types";
import { useAsyncAction } from "./useAsyncAction";
import { useEventStream } from "./useEventStream";

export function useAgents() {
  const store = useAgentsStore();
  const { agents, loading, error } = storeToRefs(store);

  async function refresh() {
    await store.fetchAgents();
  }

  onMounted(() => { void refresh(); });

  return { data: agents, loading, error, refresh };
}

export function useAgent(id: number) {
  const store = useAgentsStore();
  const { agents, loading, error } = storeToRefs(store);
  const data = computed(() => agents.value.find((a) => a.id === id) ?? null);

  async function refresh() {
    await store.fetchAgents();
  }

  onMounted(() => { void refresh(); });

  return { data, loading, error, refresh };
}

export function useCreateAgent() {
  const store = useAgentsStore();
  return useAsyncAction((payload: Parameters<typeof store.create>[0]) => store.create(payload));
}

export function usePatchAgent() {
  const store = useAgentsStore();
  return useAsyncAction((id: number, payload: Parameters<typeof store.update>[1]) =>
    store.update(id, payload),
  );
}

export function useStartAgent() {
  const store = useAgentsStore();
  return useAsyncAction((id: number) => store.start(id));
}

export function useDeleteAgent() {
  const store = useAgentsStore();
  return useAsyncAction((id: number) => store.remove(id));
}

export function useStopAgent() {
  const store = useAgentsStore();
  return useAsyncAction((id: number) => store.stop(id));
}

// ── useAgentStatus ────────────────────────────────────────────────────────────
export function useAgentStatus(agentId: number) {
  const data = ref<AgentStatus | null>(null);
  const error = ref<string | null>(null);
  const store = useAgentsStore();

  async function refresh() {
    try {
      data.value = await store.fetchStatus(agentId);
      error.value = null;
    } catch (e) {
      error.value = e instanceof Error ? e.message : "请求失败";
    }
  }

  onMounted(() => { void refresh(); });

  return { data, error, refresh };
}

export interface AgentStreamState {
  runtimeState: string | null;
  desiredState: string | null;
  lastError: string;
  lastHeartbeatAt: string | null;
  updatedAt: string | null;
}

type MaybeRef<T> = T | Ref<T>;

const agentEventNames = ["heartbeat", "log", "progress", "error"];

function readAgentStreamState(payload: unknown): AgentStreamState | null {
  if (!payload || typeof payload !== "object") {
    return null;
  }

  const raw = payload as Record<string, unknown>;
  const runtimeState = typeof raw.runtime_state === "string" ? raw.runtime_state : null;
  const desiredState = typeof raw.desired_state === "string" ? raw.desired_state : null;
  const lastHeartbeatAt = typeof raw.last_heartbeat_at === "string" ? raw.last_heartbeat_at : null;
  const updatedAt = typeof raw.updated_at === "string" ? raw.updated_at : null;
  const lastError = typeof raw.last_error === "string" ? raw.last_error : "";

  if (!runtimeState && !desiredState && !lastHeartbeatAt && !updatedAt && !lastError) {
    return null;
  }

  return {
    runtimeState,
    desiredState,
    lastError,
    lastHeartbeatAt,
    updatedAt,
  };
}

export function useAgentEventStream(agentId: MaybeRef<number | null>) {
  const streamUrl = computed(() => {
    const id = unref(agentId);
    return id ? getAgentEventsUrl(id) : null;
  });

  const stream = useEventStream(streamUrl, {
    eventNames: agentEventNames,
  });

  const statusSnapshot = computed(() => readAgentStreamState(stream.heartbeatPayload.value));
  const runtimeState = computed(() => {
    return statusSnapshot.value?.runtimeState ?? null;
  });
  const desiredState = computed(() => {
    return statusSnapshot.value?.desiredState ?? null;
  });
  const lastError = computed(() => {
    return statusSnapshot.value?.lastError ?? "";
  });
  const statusLastHeartbeatAt = computed(() => {
    return statusSnapshot.value?.lastHeartbeatAt ?? null;
  });
  const updatedAt = computed(() => {
    return statusSnapshot.value?.updatedAt ?? null;
  });
  const connected = computed(() => {
    return stream.status.value === "open" || stream.status.value === "connecting";
  });

  const isRunning = computed(() => {
    if (runtimeState.value) {
      return runtimeState.value === "running" || runtimeState.value === "idle";
    }
    return connected.value;
  });

  return {
    ...stream,
    connected,
    statusSnapshot,
    runtimeState,
    desiredState,
    lastError,
    statusLastHeartbeatAt,
    updatedAt,
    isRunning,
  };
}

export function useAgentStream(agentId: number | null) {
  const stream = useAgentEventStream(agentId);

  return {
    lines: stream.lines,
    connected: stream.connected,
    receivedHeartbeatAt: stream.lastHeartbeatAt,
    runtimeState: stream.runtimeState,
    desiredState: stream.desiredState,
    lastError: stream.lastError,
    statusSnapshot: stream.statusSnapshot,
    statusLastHeartbeatAt: stream.statusLastHeartbeatAt,
    updatedAt: stream.updatedAt,
  };
}
