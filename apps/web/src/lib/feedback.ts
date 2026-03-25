type ToastType = "success" | "error" | "warning" | "info";

type MessageOptions = {
  content?: string;
  duration?: number;
  closable?: boolean;
};

type NotificationOptions = MessageOptions & {
  title?: string;
};

function normalizeMessageOptions(options: string | MessageOptions | undefined) {
  if (typeof options === "string") {
    return {
      content: options,
      duration: 3000,
      closable: true,
    };
  }

  return {
    content: options?.content ?? "",
    duration: options?.duration ?? 3000,
    closable: options?.closable ?? true,
  };
}

function normalizeNotificationOptions(options: string | NotificationOptions | undefined) {
  if (typeof options === "string") {
    return {
      title: "通知",
      content: options,
      duration: 3000,
      closable: true,
    };
  }

  return {
    title: options?.title ?? "通知",
    content: options?.content ?? "",
    duration: options?.duration ?? 3000,
    closable: options?.closable ?? true,
  };
}

const TOAST_TONE_CLASS: Record<ToastType, string> = {
  success: "border-emerald-200 bg-emerald-50 text-emerald-700",
  error: "border-[#F4D26B] bg-[#FFF8E6] text-[#BD651F]", // Updated error tone matching design
  warning: "border-amber-200 bg-amber-50 text-amber-700",
  info: "border-sky-200 bg-sky-50 text-sky-700",
};

function getContainer(id: string, className: string) {
  if (typeof document === "undefined") return null;
  let container = document.getElementById(id);
  if (!container) {
    container = document.createElement("div");
    container.id = id;
    container.className = className;
    document.body.appendChild(container);
  }
  return container;
}

function mountToast(args: {
  type: ToastType;
  content: string;
  title?: string;
  duration: number;
  closable: boolean;
  containerId: string;
  containerClass: string;
}) {
  const container = getContainer(args.containerId, args.containerClass);
  if (!container) {
    return { close: () => {} };
  }

  const toast = document.createElement("div");
  toast.className = [
    "pointer-events-auto flex min-w-[240px] max-w-[420px] items-start gap-3 rounded-xl border px-4 py-3 shadow-md backdrop-blur",
    TOAST_TONE_CLASS[args.type],
    "animate-[fade-in_.2s_ease]",
  ].join(" ");

  const textWrap = document.createElement("div");
  textWrap.className = "min-w-0 flex-1";

  if (args.title) {
    const titleEl = document.createElement("div");
    titleEl.className = "text-sm font-semibold";
    titleEl.textContent = args.title;
    textWrap.appendChild(titleEl);
  }

  const contentEl = document.createElement("div");
  contentEl.className = "text-sm";
  contentEl.textContent = args.content;
  textWrap.appendChild(contentEl);

  toast.appendChild(textWrap);

  const close = () => {
    if (!toast.isConnected) return;
    toast.remove();
  };

  if (args.closable) {
    const closeBtn = document.createElement("button");
    closeBtn.type = "button";
    closeBtn.className = "shrink-0 rounded-md px-1 text-current/70 transition hover:bg-black/5 hover:text-current";
    closeBtn.textContent = "x";
    closeBtn.addEventListener("click", close);
    toast.appendChild(closeBtn);
  }

  container.appendChild(toast);

  const timer = window.setTimeout(close, Math.max(args.duration, 600));

  return {
    close: () => {
      window.clearTimeout(timer);
      close();
    },
  };
}

function message(type: ToastType, options: string | MessageOptions) {
  const normalized = normalizeMessageOptions(options);
  return mountToast({
    type,
    content: normalized.content,
    duration: normalized.duration,
    closable: normalized.closable,
    containerId: "octo-message-container",
    containerClass: "pointer-events-none fixed left-1/2 top-5 z-[720] flex -translate-x-1/2 flex-col gap-2",
  });
}

function notification(type: ToastType, options: string | NotificationOptions) {
  const normalized = normalizeNotificationOptions(options);
  return mountToast({
    type,
    title: normalized.title,
    content: normalized.content,
    duration: normalized.duration,
    closable: normalized.closable,
    containerId: "octo-notification-container",
    containerClass: "pointer-events-none fixed right-5 top-5 z-[720] flex flex-col gap-2",
  });
}

export const Message = {
  success: (options: string | MessageOptions) => message("success", options),
  error: (options: string | MessageOptions) => message("error", options),
  warning: (options: string | MessageOptions) => message("warning", options),
  info: (options: string | MessageOptions) => message("info", options),
};

export const Notification = {
  success: (options: string | NotificationOptions) => notification("success", options),
  error: (options: string | NotificationOptions) => notification("error", options),
  warning: (options: string | NotificationOptions) => notification("warning", options),
  info: (options: string | NotificationOptions) => notification("info", options),
};
