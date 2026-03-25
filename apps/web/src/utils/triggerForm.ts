import type { JobDefinition } from "@/types";

type JsonRecord = Record<string, unknown>;

export function parseTriggerDefaultInput(raw: string): JsonRecord {
  const trimmed = raw.trim();
  if (!trimmed) {
    return {};
  }

  const parsed = JSON.parse(trimmed) as unknown;
  if (!parsed || typeof parsed !== "object" || Array.isArray(parsed)) {
    throw new Error("默认输入必须是 JSON 对象");
  }
  return parsed as JsonRecord;
}

export function stringifyTriggerDefaultInput(input: Record<string, unknown> | null | undefined): string {
  return JSON.stringify(input ?? {}, null, 2);
}

export function formatJobDefinitionOptionLabel(job: JobDefinition): string {
  const meta = [job.key, `${job.plugin_key}:${job.action}`].filter(Boolean).join(" · ");
  return meta ? `${job.name} (#${job.id}) · ${meta}` : `${job.name} (#${job.id})`;
}
