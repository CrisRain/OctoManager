import { render, screen } from "@testing-library/react";
import React from "react";
import { describe, expect, it, vi } from "vitest";

vi.mock("@/App", () => ({
  App: () => <div>mounted-app</div>,
}));

vi.mock("sonner", () => ({
  Toaster: () => <div>toaster</div>,
}));

describe("main", () => {
  it("mounts the root app", async () => {
    document.body.innerHTML = `<div id="root"></div>`;

    await import("@/main");

    expect(await screen.findByText("mounted-app")).toBeInTheDocument();
    expect(screen.getByText("toaster")).toBeInTheDocument();
  });
});
