import { beforeEach, describe, expect, it, vi } from "vitest";

import { ApiRequestError, api, extractErrorMessage, fetchHealth } from "@/lib/api";

describe("api", () => {
  beforeEach(() => {
    vi.stubGlobal("fetch", vi.fn());
  });

  it("requests health successfully", async () => {
    vi.mocked(fetch).mockResolvedValue({
      ok: true,
      json: async () => ({
        code: 0,
        message: "success",
        data: { status: "ok", time: "2026-03-07T00:00:00Z" },
      }),
    } as Response);

    await expect(fetchHealth()).resolves.toEqual({
      status: "ok",
      time: "2026-03-07T00:00:00Z",
    });

    expect(fetch).toHaveBeenCalledWith(
      "/healthz",
      expect.objectContaining({
        headers: { "Content-Type": "application/json" },
      }),
    );
  });

  it("builds filtered query strings", async () => {
    vi.mocked(fetch).mockResolvedValue({
      ok: true,
      json: async () => ({
        code: 0,
        message: "success",
        data: { items: [], total: 0, limit: 10, offset: 0 },
      }),
    } as Response);

    await api.listAccounts({ limit: 5, offset: 0, type_key: "demo" });

    expect(fetch).toHaveBeenCalledWith(
      "/api/v1/accounts/?limit=5&offset=0&type_key=demo",
      expect.any(Object),
    );
  });

  it("adds authorization header for trigger firing", async () => {
    vi.mocked(fetch).mockResolvedValue({
      ok: true,
      json: async () => ({
        code: 0,
        message: "success",
        data: { queued: true },
      }),
    } as Response);

    await api.fireTrigger("demo trigger", "secret-token", { ok: true });

    expect(fetch).toHaveBeenCalledWith(
      "/webhooks/demo%20trigger",
      expect.objectContaining({
        method: "POST",
        headers: expect.objectContaining({
          Authorization: "Bearer secret-token",
        }),
      }),
    );
  });

  it("throws ApiRequestError for failed envelopes", async () => {
    vi.mocked(fetch).mockResolvedValue({
      ok: false,
      json: async () => ({
        code: 40000,
        message: "bad request",
      }),
    } as Response);

    await expect(api.listAccountTypes()).rejects.toBeInstanceOf(ApiRequestError);
  });

  it("extracts error messages consistently", () => {
    const apiError = new ApiRequestError({
      code: "BAD_REQUEST",
      message: "bad request",
    });

    expect(extractErrorMessage(apiError)).toBe("BAD_REQUEST: bad request");
    expect(extractErrorMessage(new Error("plain error"))).toBe("plain error");
    expect(extractErrorMessage("oops")).toBe("unknown error");
  });
});
