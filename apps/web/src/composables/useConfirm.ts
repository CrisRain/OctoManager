import { render, createVNode, ref, h } from "vue";
import { UiModal, UiButton, UiInput } from "@/lib/ui";

function openModal(options: any): Promise<boolean> {
  return new Promise((resolve) => {
    const mountNode = document.createElement("div");
    document.body.appendChild(mountNode);

    const visible = ref(true);
    const inputValue = ref("");
    const inputError = ref("");

    const destroy = () => {
      visible.value = false;
      setTimeout(() => {
        render(null, mountNode);
        mountNode.remove();
      }, 300); // Wait for transition
    };

    const handleOk = () => {
      if (options.withInput) {
        if (options.requiredText && inputValue.value !== options.requiredText) {
          inputError.value = `输入内容必须为：${options.requiredText}`;
          return;
        }
        if (!options.requiredText && !inputValue.value.trim()) {
          inputError.value = "输入内容不能为空";
          return;
        }
      }
      resolve(true);
      destroy();
    };

    const handleCancel = () => {
      resolve(false);
      destroy();
    };

    const vnode = createVNode({
      setup() {
        return () => {
          const bodyContent: any[] = [];
          bodyContent.push(h("div", { class: "whitespace-pre-wrap text-sm text-slate-600 mb-4" }, options.content));

          if (options.withInput) {
            const reqHint = options.requiredText ? `请输入 ${options.requiredText} 确认` : options.placeholder;
            bodyContent.push(
              h("div", { class: "flex flex-col gap-1" }, [
                h(UiInput, {
                  modelValue: inputValue.value,
                  "onUpdate:modelValue": (val: string) => {
                    inputValue.value = val;
                    inputError.value = "";
                  },
                  placeholder: reqHint,
                  status: inputError.value ? "error" : undefined,
                  autofocus: true,
                  onKeyup: (e: KeyboardEvent) => {
                    if (e.key === "Enter") handleOk();
                  }
                }),
                inputError.value ? h("span", { class: "text-xs text-red-500 mt-1" }, inputError.value) : null
              ])
            );
          }

          return h(UiModal, {
            visible: visible.value,
            "onUpdate:visible": (val: boolean) => {
              if (!val) handleCancel();
            },
            title: options.title,
            maskClosable: false,
          }, {
            default: () => bodyContent,
            footer: () => h("div", { class: "flex items-center justify-end gap-3 w-full" }, [
              h(UiButton, { onClick: handleCancel }, () => options.cancelText || "取消"),
              h(UiButton, { 
                type: "primary", 
                status: options.okButtonProps?.status || "normal",
                onClick: handleOk,
              }, () => options.okText || "确定")
            ])
          });
        };
      }
    });

    render(vnode, mountNode);
  });
}

/**
 * 统一的确认对话框系统
 * 使用模态框代替原生 confirm/prompt。
 */
export const useConfirm = () => {
  const confirm = (
    content: string,
    title = "确认操作",
    options?: {
      okText?: string;
      cancelText?: string;
      okButtonProps?: { status?: "danger" | "success" | "warning" | "normal" };
    },
  ): Promise<boolean> => {
    return openModal({
      content,
      title,
      ...options,
    });
  };

  const confirmDanger = (
    content: string,
    title = "危险操作",
    okText = "确认删除",
  ): Promise<boolean> => {
    return confirm(content, title, {
      okText,
      okButtonProps: { status: "danger" },
    });
  };

  const confirmDelete = (itemName?: string): Promise<boolean> => {
    const content = itemName
      ? `确定要删除"${itemName}"吗？此操作无法撤销。`
      : "确定要删除吗？此操作无法撤销。";
    return confirmDanger(content, "确认删除");
  };

  const confirmStop = (itemName?: string): Promise<boolean> => {
    const content = itemName
      ? `确定要停止"${itemName}"吗？`
      : "确定要停止吗？";
    return confirm(content, "确认停止", {
      okText: "停止",
      okButtonProps: { status: "warning" },
    });
  };

  const confirmStart = (itemName?: string): Promise<boolean> => {
    const content = itemName
      ? `确定要启动"${itemName}"吗？`
      : "确定要启动吗？";
    return confirm(content, "确认启动", {
      okText: "启动",
      okButtonProps: { status: "success" },
    });
  };

  const confirmBatch = (
    action: string,
    count: number,
  ): Promise<boolean> => {
    return confirm(
      `确定要对选中的 ${count} 项执行"${action}"操作吗？`,
      `批量${action}`,
      {
        okText: "确定",
        okButtonProps: { status: "normal" },
      },
    );
  };

  const confirmWithInput = (
    content: string,
    title = "确认操作",
    placeholder = "请输入确认",
    options?: {
      requiredText?: string;
      okText?: string;
      cancelText?: string;
      okButtonProps?: { status?: "danger" | "success" | "warning" | "normal" };
    },
  ): Promise<boolean> => {
    return openModal({
      content,
      title,
      placeholder,
      withInput: true,
      ...options,
    });
  };

  const confirmDangerWithInput = (
    content: string,
    confirmText: string,
    title = "危险操作",
  ): Promise<boolean> => {
    return confirmWithInput(content, title, `请输入 ${confirmText} 确认`, {
      requiredText: confirmText,
      okText: "确认删除",
      okButtonProps: { status: "danger" },
    });
  };

  return {
    confirm,
    confirmDanger,
    confirmDelete,
    confirmStop,
    confirmStart,
    confirmBatch,
    confirmWithInput,
    confirmDangerWithInput,
  };
};
