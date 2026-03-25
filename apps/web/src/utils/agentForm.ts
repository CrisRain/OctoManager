import type { Account } from "@/types";

type JsonRecord = Record<string, unknown>;

function asRecord(value: unknown): JsonRecord {
  if (!value || typeof value !== "object" || Array.isArray(value)) {
    return {};
  }
  return value as JsonRecord;
}

export function parseAgentParamsJSON(raw: string): JsonRecord {
  const trimmed = raw.trim();
  if (!trimmed) {
    return {};
  }

  const parsed = JSON.parse(trimmed) as unknown;
  if (!parsed || typeof parsed !== "object" || Array.isArray(parsed)) {
    throw new Error("额外参数必须是 JSON 对象");
  }
  return parsed as JsonRecord;
}

export function buildAgentInput(account: Account | null, params: JsonRecord): JsonRecord {
  const input: JsonRecord = {};

  if (account) {
    input.account = {
      id: account.id,
      identifier: account.identifier,
      spec: account.spec ?? {},
    };
  }

  if (Object.keys(params).length > 0) {
    input.params = params;
  }

  return input;
}

export function splitAgentInput(input: Record<string, unknown> | null | undefined): {
  accountId: string;
  params: JsonRecord;
} {
  const rawInput = asRecord(input);
  const rawAccount = asRecord(rawInput.account);
  const rawParams = asRecord(rawInput.params);

  const extraParams: JsonRecord = {};
  for (const [key, value] of Object.entries(rawInput)) {
    if (key === "account" || key === "params") {
      continue;
    }
    extraParams[key] = value;
  }

  const accountId = typeof rawAccount.id === "number" ? String(rawAccount.id) : "";
  const params = Object.keys(rawParams).length > 0 ? rawParams : extraParams;

  return { accountId, params };
}

export function stringifyAgentParams(params: JsonRecord): string {
  return JSON.stringify(params, null, 2);
}

export function formatAccountOptionLabel(account: Account): string {
  const typeKey = String(account.account_type_key ?? "").trim();
  const status = String(account.status ?? "").trim();
  const suffix = [typeKey, status].filter(Boolean).join(" / ");
  return suffix ? `${account.identifier} · ${suffix}` : account.identifier;
}
