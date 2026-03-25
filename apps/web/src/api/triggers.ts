import { client } from "@/shared/api/generated/client";
import type {
  ListTriggersResponse,
  Trigger,
  TriggerCreateInput,
  TriggerFireInput,
  TriggerFireResult,
  TriggerPatchInput,
} from "@/types";

export const listTriggers = (): Promise<ListTriggersResponse> => client.listTriggers();

export const getTrigger = (id: number): Promise<Trigger> =>
  client.getTrigger({ path: { id } });

export const createTrigger = (payload: TriggerCreateInput) =>
  client.createTrigger({ body: payload });

export const patchTrigger = (id: number, payload: TriggerPatchInput): Promise<Trigger> =>
  client.patchTrigger({ path: { id }, body: payload });

export const deleteTrigger = (id: number) =>
  client.deleteTrigger({ path: { id } });

export const fireTrigger = (
  id: number,
  payload?: TriggerFireInput,
): Promise<TriggerFireResult> =>
  client.fireTrigger({ path: { id }, body: payload });
