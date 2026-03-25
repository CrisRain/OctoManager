import { client } from "@/shared/api/generated/client";
import type { ListPluginsResponse, Plugin, PluginSyncResult } from "@/types";

export const listPlugins = (): Promise<ListPluginsResponse> => client.listPlugins();

export const getPlugin = (key: string): Promise<Plugin> =>
  client.getPlugin({ path: { key } });

export const getPluginSettings = (key: string): Promise<Record<string, unknown>> =>
  client.getPluginSettings({ path: { key } });

export const updatePluginSettings = (
  key: string,
  values: Record<string, unknown>,
): Promise<{ saved: boolean }> =>
  client.putPluginSettings({ path: { key }, body: values });

export const syncPlugins = (): Promise<PluginSyncResult> => client.syncPlugins();
