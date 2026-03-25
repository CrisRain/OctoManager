import { onMounted } from "vue";
import { storeToRefs } from "pinia";
import { useSystemStore } from "@/store";
import type { DashboardSummary } from "@/types";

export type { DashboardSummary };

export function useSystemStatus() {
  const store = useSystemStore();
  const { systemStatus, loadingStatus, error } = storeToRefs(store);

  async function refresh() {
    await store.fetchSystemStatus();
  }

  onMounted(() => { void refresh(); });

  return { data: systemStatus, loading: loadingStatus, error, refresh };
}

export function useDashboardSnapshot() {
  const store = useSystemStore();
  const { dashboardSummary, loadingDashboard, error } = storeToRefs(store);

  async function refresh() {
    await store.fetchDashboardSummary();
  }

  onMounted(() => { void refresh(); });

  return { data: dashboardSummary, loading: loadingDashboard, error, refresh };
}
