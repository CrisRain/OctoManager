import { client } from "@/shared/api/generated/client";
import type {
  JobDefinition,
  JobDefinitionCreateInput,
  JobDefinitionPatchInput,
  JobExecution,
  ListJobDefinitionsResponse,
  ListJobExecutionsResponse,
} from "@/types";

export const listJobDefinitions = (): Promise<ListJobDefinitionsResponse> =>
  client.listJobDefinitions();

export const getJobDefinition = (id: number): Promise<JobDefinition> =>
  client.getJobDefinition({ path: { id } });

export const listJobExecutions = (): Promise<ListJobExecutionsResponse> =>
  client.listJobExecutions();

export const createJobDefinition = (
  payload: JobDefinitionCreateInput,
): Promise<JobDefinition> => client.createJobDefinition({ body: payload });

export const patchJobDefinition = (
  id: number,
  payload: JobDefinitionPatchInput,
): Promise<JobDefinition> => client.patchJobDefinition({ path: { id }, body: payload });

export const deleteJobDefinition = (id: number): Promise<void> =>
  client.deleteJobDefinition({ path: { id } });

export const enqueueJobExecution = (id: number): Promise<JobExecution> =>
  client.enqueueJobExecution({ path: { id } });

export const getJobExecution = (id: number): Promise<JobExecution> =>
  client.getJobExecution({ path: { id } });
