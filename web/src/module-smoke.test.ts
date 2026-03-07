import { expect, it } from "vitest";

const modules = import.meta.glob([
  "./App.tsx",
  "./types.ts",
  "./components/**/*.{ts,tsx}",
  "./pages/**/*.{ts,tsx}",
  "./lib/**/*.{ts,tsx}",
  "!./**/*.test.*",
  "!./main.tsx",
]);

it("imports frontend modules without crashing", async () => {
  const loaded = await Promise.all(Object.values(modules).map((loader) => loader()));
  expect(loaded.length).toBeGreaterThan(0);
});
