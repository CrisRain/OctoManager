import { ref } from "vue";
import { useConfigStore } from "@/store";
import type { SetConfigRequestBody, SystemConfigValue } from "@/types";

const KNOWN_CONFIGS = [
  { key: "app.name", label: "应用名称", description: "平台显示名称" },
  { key: "job.default_timeout_minutes", label: "任务超时（分钟）", description: "默认任务执行超时" },
  { key: "job.max_concurrency", label: "最大并发数", description: "Worker 最大并发任务数" },
  { key: "plugins.grpc_services", label: "插件 gRPC 服务", description: "插件键到 gRPC 地址的 JSON 映射，启动时会自动初始化默认值" },
] as const;

export type KnownConfigItem = (typeof KNOWN_CONFIGS)[number];

export function useSystemConfigs() {
  const store = useConfigStore();
  const configs = ref<Record<string, string>>({});
  const configEdits = ref<Record<string, string>>({});
  const configSaving = ref<Record<string, boolean>>({});

  async function loadConfigs() {
    const values: Record<string, string> = {};
    for (const config of KNOWN_CONFIGS) {
      try {
        const res = await store.fetchConfig(config.key);
        values[config.key] = JSON.stringify(res?.value ?? "");
      } catch {
        values[config.key] = "";
      }
    }
    configs.value = { ...values };
    configEdits.value = { ...values };
  }

  async function saveConfig(key: string): Promise<SystemConfigValue> {
    const raw = configEdits.value[key] ?? "";
    let parsed: SetConfigRequestBody["value"];
    try {
      parsed = JSON.parse(raw) as SetConfigRequestBody["value"];
    } catch {
      throw new Error(`${key}: 值必须是有效 JSON`);
    }

    configSaving.value = { ...configSaving.value, [key]: true };
    try {
      const result = await store.updateConfig(key, parsed);
      if (!result) {
        throw new Error("保存失败");
      }
      configs.value = { ...configs.value, [key]: raw };
      return result;
    } finally {
      configSaving.value = { ...configSaving.value, [key]: false };
    }
  }

  return {
    knownConfigs: KNOWN_CONFIGS,
    configs,
    configEdits,
    configSaving,
    loadConfigs,
    saveConfig,
  };
}
