import { client } from "@/shared/api/generated/client";
import type { Agent, AgentCreateInput, AgentPatchInput, AgentStatus, ListAgentsResponse } from "@/types";

export const listAgents = (): Promise<ListAgentsResponse> => client.listAgents();

export const getAgent = (id: number): Promise<Agent> =>
  client.getAgent({ path: { id } });

export const createAgent = (payload: AgentCreateInput) =>
  client.createAgent({ body: payload });

export const startAgent = (id: number) => client.startAgent({ path: { id } });

export const stopAgent = (id: number) => client.stopAgent({ path: { id } });

export const getAgentStatus = (id: number): Promise<AgentStatus> =>
  client.getAgentStatus({ path: { id } });

export const patchAgent = (id: number, payload: AgentPatchInput): Promise<Agent> =>
  client.patchAgent({ path: { id }, body: payload });

export const deleteAgent = (id: number): Promise<void> =>
  client.deleteAgent({ path: { id } });
