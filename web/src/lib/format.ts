import type { JsonObject } from "@/types";

export function formatDateTime(raw: string): string {
  if (!raw) {
    return "-";
  }
  const date = new Date(raw);
  if (Number.isNaN(date.getTime())) {
    return raw;
  }
  return new Intl.DateTimeFormat("zh-CN", {
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
    hour12: false
  }).format(date);
}

export function parseJSONObjectText(input: string, fieldName: string): JsonObject {
  const raw = input.trim() === "" ? "{}" : input.trim();
  let parsed: unknown;
  try {
    parsed = JSON.parse(raw);
  } catch {
    throw new Error(`${fieldName} 不是合法 JSON`);
  }
  if (!parsed || typeof parsed !== "object" || Array.isArray(parsed)) {
    throw new Error(`${fieldName} 必须是 JSON object`);
  }
  return parsed as JsonObject;
}

export function parseJSONOrNull(input: string, fieldName: string): unknown {
  const raw = input.trim();
  if (raw === "") {
    return null;
  }
  try {
    return JSON.parse(raw);
  } catch {
    throw new Error(`${fieldName} 不是合法 JSON`);
  }
}

export function compactId(id: string | number): string {
  const s = String(id);
  if (!s || s === "0") {
    return "-";
  }
  if (s.length <= 14) {
    return s;
  }
  return `${s.slice(0, 6)}...${s.slice(-6)}`;
}

export function prettyJSON(value: unknown): string {
  return JSON.stringify(value ?? {}, null, 2);
}
