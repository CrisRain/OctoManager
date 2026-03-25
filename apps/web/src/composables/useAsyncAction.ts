import { ref } from "vue";

export function useAsyncAction<TArgs extends unknown[], TReturn>(
  fn: (...args: TArgs) => Promise<TReturn>,
) {
  const loading = ref(false);
  const error = ref<string | null>(null);

  async function execute(...args: TArgs): Promise<TReturn> {
    loading.value = true;
    error.value = null;
    try {
      return await fn(...args);
    } catch (e) {
      error.value = e instanceof Error ? e.message : "请求失败";
      throw e;
    } finally {
      loading.value = false;
    }
  }

  return { loading, error, execute };
}
