import { describe, expect, it } from "vitest";

import {
  compactId,
  formatDateTime,
  parseJSONOrNull,
  parseJSONObjectText,
  prettyJSON,
} from "@/lib/format";

describe("format helpers", () => {
  it("formats datetime and handles empty input", () => {
    expect(formatDateTime("")).toBe("-");
    expect(formatDateTime("not-a-date")).toBe("not-a-date");
    expect(formatDateTime("2026-03-07T12:34:56Z")).toContain("2026");
  });

  it("parses JSON objects", () => {
    expect(parseJSONObjectText("", "payload")).toEqual({});
    expect(parseJSONObjectText(`{"ok":true}`, "payload")).toEqual({ ok: true });
    expect(() => parseJSONObjectText(`[]`, "payload")).toThrow();
    expect(() => parseJSONObjectText(`{`, "payload")).toThrow();
  });

  it("parses nullable JSON", () => {
    expect(parseJSONOrNull("", "payload")).toBeNull();
    expect(parseJSONOrNull(`[1,2]`, "payload")).toEqual([1, 2]);
    expect(() => parseJSONOrNull("{", "payload")).toThrow();
  });

  it("compacts ids and pretty prints json", () => {
    expect(compactId(0)).toBe("-");
    expect(compactId("12345678901234567890")).toBe("123456...567890");
    expect(prettyJSON({ ok: true })).toContain(`"ok": true`);
  });
});
