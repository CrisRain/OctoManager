import { computed, ref } from "vue";
import { useMessage } from "./useMessage";
import { Notification } from "@/lib/feedback";

/**
 * 常见错误类型的用户友好映射
 */
const ERROR_MESSAGES: Record<string, string> = {
  // 网络错误
  "Network Error": "网络连接失败，请检查网络设置",
  "timeout": "请求超时，请稍后重试",
  "ERR_NETWORK": "网络连接失败",

  // HTTP 状态码
  "400": "请求参数有误",
  "401": "未授权，请检查管理员密钥",
  "403": "没有权限执行此操作",
  "404": "请求的资源不存在",
  "409": "数据冲突，请刷新后重试",
  "422": "输入数据格式不正确",
  "429": "请求过于频繁，请稍后再试",
  "500": "服务器内部错误",
  "502": "网关错误，服务暂时不可用",
  "503": "服务维护中，请稍后再试",

  // 业务错误
  "not_found": "找不到相关数据",
  "invalid_input": "输入数据不正确",
  "unauthorized": "权限不足",
  "duplicate": "数据已存在",
  "foreign_key": "有关联数据无法删除",
};

/**
 * 错误恢复建议：针对常见错误给出操作提示
 */
const ERROR_HINTS: Record<string, string> = {
  "Network Error": "请检查网络连接后刷新页面",
  "ERR_NETWORK": "请检查网络连接后刷新页面",
  "timeout": "可尝试刷新页面或稍后重试",
  "401": "请在系统设置中确认 API 密钥是否正确",
  "403": "如需帮助，请联系管理员",
  "404": "该资源可能已被删除，请返回列表页",
  "409": "请刷新页面后再试",
  "500": "如问题持续出现，请检查服务端日志",
  "502": "服务可能正在重启，请稍后刷新",
  "503": "请稍后刷新页面",
  "foreign_key": "请先删除引用该资源的其他数据",
  "duplicate": "请使用不同的名称或标识符",
};

/**
 * 获取用户友好的错误消息
 */
function getFriendlyMessage(error: Error | string): string {
  const message = typeof error === "string" ? error : error.message;
  const lowerMessage = message.toLowerCase();

  // 检查常见错误模式
  if (lowerMessage.includes("network") || lowerMessage.includes("fetch")) {
    return ERROR_MESSAGES["Network Error"];
  }
  if (lowerMessage.includes("timeout")) {
    return ERROR_MESSAGES["timeout"];
  }

  // 检查HTTP状态码
  for (const [code, msg] of Object.entries(ERROR_MESSAGES)) {
    if (message.includes(code)) {
      return msg;
    }
  }

  // 返回原始消息（如果不太长）或通用消息
  if (message.length < 50) {
    return message;
  }
  return "操作失败，请稍后重试";
}

/**
 * 获取错误恢复建议
 */
function getErrorHint(error: Error | string): string | undefined {
  const message = typeof error === "string" ? error : error.message;
  const lowerMessage = message.toLowerCase();

  if (lowerMessage.includes("network") || lowerMessage.includes("fetch")) {
    return ERROR_HINTS["Network Error"];
  }
  if (lowerMessage.includes("timeout")) {
    return ERROR_HINTS["timeout"];
  }

  for (const [code, hint] of Object.entries(ERROR_HINTS)) {
    if (message.includes(code)) {
      return hint;
    }
  }
  return undefined;
}

/**
 * 统一的错误处理composable
 */
export const useErrorHandler = () => {
  const message = useMessage();

  /**
   * 处理错误并显示用户友好的提示
   */
  const handleError = (error: Error | string, action?: string) => {
    const friendlyMsg = getFriendlyMessage(error);
    const hint = getErrorHint(error);

    if (hint) {
      // 有恢复建议时，使用通知（带标题+提示）
      Notification.error({
        title: action ? `${action}失败` : "操作失败",
        content: `${friendlyMsg}。${hint}`,
        duration: 6000,
      });
    } else if (action) {
      message.error(`${action}失败：${friendlyMsg}`);
    } else {
      message.error(friendlyMsg);
    }

    return friendlyMsg;
  };

  /**
   * 包装异步操作，自动处理错误
   */
  const withErrorHandler = async <T>(
    fn: () => Promise<T>,
    options?: {
      action?: string; // 操作名称，用于错误提示
      onSuccess?: (result: T) => void;
      onError?: (error: Error) => void;
      showSuccess?: boolean; // 是否显示成功提示
    }
  ): Promise<T | null> => {
    try {
      const result = await fn();
      if (options?.showSuccess && options.action) {
        message.successAction(options.action);
      }
      options?.onSuccess?.(result);
      return result;
    } catch (e) {
      const error = e instanceof Error ? e : new Error(String(e));
      handleError(error, options?.action);
      options?.onError?.(error);
      return null;
    }
  };

  return {
    handleError,
    withErrorHandler,
    getFriendlyMessage,
    getErrorHint,
  };
};

/**
 * 创建表单错误处理器
 */
export const useFormErrors = <T extends Record<string, any>>() => {
  const errors = ref<Record<keyof T, string>>({} as Record<keyof T, string>);

  /**
   * 设置字段错误
   */
  const setFieldError = (field: keyof T, message: string) => {
    errors.value[field] = message;
  };

  /**
   * 清除字段错误
   */
  const clearFieldError = (field: keyof T) => {
    delete errors.value[field];
  };

  /**
   * 清除所有错误
   */
  const clearAllErrors = () => {
    errors.value = {} as Record<keyof T, string>;
  };

  /**
   * 检查是否有错误
   */
  const hasErrors = computed(() => Object.keys(errors.value).length > 0);

  /**
   * 获取字段错误
   */
  const getError = (field: keyof T) => errors.value[field];

  /**
   * 处理API验证错误
   */
  const handleValidationErrors = (error: Error) => {
    clearAllErrors();
    // 假设API返回的错误格式为 { field: "error message" }
    try {
      const data = JSON.parse(error.message);
      for (const [field, msg] of Object.entries(data)) {
        setFieldError(field as keyof T, String(msg));
      }
    } catch {
      // 如果不是验证错误，显示通用错误
      return false;
    }
    return true;
  };

  return {
    errors,
    setFieldError,
    clearFieldError,
    clearAllErrors,
    hasErrors,
    getError,
    handleValidationErrors,
  };
};
