import { Message, Notification } from "@/lib/feedback";

/**
 * 统一的消息提示系统
 * 提供简单易用的消息提示方法，支持多种类型
 */
export const useMessage = () => {
  const success = (content: string, duration = 2200) => {
    return Message.success({
      content,
      duration,
      closable: false,
    });
  };

  const error = (content: string, duration = 4200) => {
    return Message.error({
      content,
      duration,
      closable: true,
    });
  };

  const warning = (content: string, duration = 3200) => {
    return Message.warning({
      content,
      duration,
      closable: false,
    });
  };

  const info = (content: string, duration = 2400) => {
    return Message.info({
      content,
      duration,
      closable: false,
    });
  };

  /**
   * 操作成功提示（绿色，较简洁）
   */
  const successAction = (action: string) => {
    return success(`${action}成功`);
  };

  /**
   * 操作失败提示（红色，较详细）
   */
  const errorAction = (action: string, detail?: string) => {
    const msg = detail ? `${action}失败：${detail}` : `${action}失败`;
    return error(msg);
  };

  /**
   * 通知（右上角，适合重要消息）
   */
  const notify = {
    success: (title: string, content = "") => {
      Notification.success({
        title,
        content,
        duration: 3600,
        closable: true,
      });
    },
    error: (title: string, content = "") => {
      Notification.error({
        title,
        content,
        duration: 5200,
        closable: true,
      });
    },
    warning: (title: string, content = "") => {
      Notification.warning({
        title,
        content,
        duration: 4200,
        closable: true,
      });
    },
    info: (title: string, content = "") => {
      Notification.info({
        title,
        content,
        duration: 3600,
        closable: true,
      });
    },
  };

  return {
    success,
    error,
    warning,
    info,
    successAction,
    errorAction,
    notify,
  };
};
