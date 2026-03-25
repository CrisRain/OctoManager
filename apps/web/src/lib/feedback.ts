type ToastType = "success" | "error" | "warning" | "info";
type ToastChannel = "message" | "notification";

type MessageOptions = {
  content?: string;
  duration?: number;
  closable?: boolean;
};

type NotificationOptions = MessageOptions & {
  title?: string;
};

type ToastHandle = {
  close: () => void;
};

type ToastPayload = {
  id: string;
  channel: ToastChannel;
  type: ToastType;
  title?: string;
  content: string;
  duration: number;
  closable: boolean;
  count: number;
  createdAt: number;
  signature: string;
  isSummary?: boolean;
};

type ToastInstance = {
  payload: ToastPayload;
  root: HTMLDivElement;
  glowEl: HTMLDivElement;
  iconWrapEl: HTMLDivElement;
  iconTextEl: HTMLSpanElement;
  titleEl: HTMLDivElement;
  contentEl: HTMLDivElement;
  countEl: HTMLSpanElement;
  progressEl: HTMLDivElement;
  close: () => void;
  timerId?: number;
  cleanupId?: number;
};

const TOAST_PRIORITY: Record<ToastType, number> = {
  success: 0,
  info: 1,
  warning: 2,
  error: 3,
};

const CHANNEL_CONFIG: Record<
  ToastChannel,
  {
    containerId: string;
    containerClass: string;
    maxVisible: number;
    maxQueue: number;
    dedupeWindow: number;
    summaryTitle: string;
    summaryDuration: number;
    summaryLabel: string;
  }
> = {
  message: {
    containerId: "octo-message-container",
    containerClass: "pointer-events-none fixed left-1/2 top-4 z-[720] flex w-[min(calc(100vw-1.5rem),28rem)] -translate-x-1/2 flex-col gap-2 md:top-5",
    maxVisible: 2,
    maxQueue: 2,
    dedupeWindow: 1600,
    summaryTitle: "消息已折叠",
    summaryDuration: 3600,
    summaryLabel: "条消息已合并显示",
  },
  notification: {
    containerId: "octo-notification-container",
    containerClass: "pointer-events-none fixed right-4 top-4 z-[720] flex w-[min(calc(100vw-1.5rem),24rem)] flex-col gap-2 md:right-5 md:top-5",
    maxVisible: 2,
    maxQueue: 2,
    dedupeWindow: 2400,
    summaryTitle: "通知已折叠",
    summaryDuration: 5200,
    summaryLabel: "条通知已合并显示",
  },
};

const TOAST_META: Record<
  ToastType,
  {
    label: string;
    symbol: string;
    borderClass: string;
    iconWrapClass: string;
    iconTextClass: string;
    titleClass: string;
    progressColor: string;
    glow: string;
  }
> = {
  success: {
    label: "成功",
    symbol: "✓",
    borderClass: "border-emerald-200/90 bg-white/95 text-slate-700",
    iconWrapClass: "border border-emerald-200/90 bg-emerald-50",
    iconTextClass: "text-emerald-600",
    titleClass: "text-emerald-900",
    progressColor: "linear-gradient(90deg, rgba(16,185,129,0.9), rgba(16,185,129,0.45))",
    glow: "radial-gradient(circle at top left, rgba(16,185,129,0.15), transparent 62%)",
  },
  error: {
    label: "错误",
    symbol: "!",
    borderClass: "border-rose-200/90 bg-white/95 text-slate-700",
    iconWrapClass: "border border-rose-200/90 bg-rose-50",
    iconTextClass: "text-rose-600",
    titleClass: "text-rose-900",
    progressColor: "linear-gradient(90deg, rgba(244,63,94,0.9), rgba(244,63,94,0.45))",
    glow: "radial-gradient(circle at top left, rgba(244,63,94,0.15), transparent 62%)",
  },
  warning: {
    label: "警告",
    symbol: "!",
    borderClass: "border-amber-200/90 bg-white/95 text-slate-700",
    iconWrapClass: "border border-amber-200/90 bg-amber-50",
    iconTextClass: "text-amber-600",
    titleClass: "text-amber-900",
    progressColor: "linear-gradient(90deg, rgba(245,158,11,0.92), rgba(245,158,11,0.45))",
    glow: "radial-gradient(circle at top left, rgba(245,158,11,0.16), transparent 62%)",
  },
  info: {
    label: "提示",
    symbol: "i",
    borderClass: "border-sky-200/90 bg-white/95 text-slate-700",
    iconWrapClass: "border border-sky-200/90 bg-sky-50",
    iconTextClass: "text-sky-600",
    titleClass: "text-sky-900",
    progressColor: "linear-gradient(90deg, rgba(14,165,233,0.9), rgba(14,165,233,0.45))",
    glow: "radial-gradient(circle at top left, rgba(14,165,233,0.14), transparent 62%)",
  },
};

const channelState: Record<
  ToastChannel,
  {
    active: ToastInstance[];
    queue: ToastPayload[];
  }
> = {
  message: {
    active: [],
    queue: [],
  },
  notification: {
    active: [],
    queue: [],
  },
};

let toastSequence = 0;

function nextToastId() {
  toastSequence += 1;
  return `octo-toast-${toastSequence}`;
}

function noopHandle(): ToastHandle {
  return { close: () => {} };
}

function createHandle(close: () => void): ToastHandle {
  return { close };
}

function normalizeMessageOptions(options: string | MessageOptions | undefined) {
  if (typeof options === "string") {
    return {
      content: options,
      duration: 2600,
      closable: false,
    };
  }

  return {
    content: options?.content ?? "",
    duration: options?.duration ?? 2600,
    closable: options?.closable ?? false,
  };
}

function normalizeNotificationOptions(options: string | NotificationOptions | undefined) {
  if (typeof options === "string") {
    return {
      title: "通知",
      content: options,
      duration: 4200,
      closable: true,
    };
  }

  return {
    title: options?.title ?? "通知",
    content: options?.content ?? "",
    duration: options?.duration ?? 4200,
    closable: options?.closable ?? true,
  };
}

function buildSignature(type: ToastType, title: string | undefined, content: string) {
  return `${type}::${title ?? ""}::${content}`;
}

function promoteType(current: ToastType, incoming: ToastType) {
  return TOAST_PRIORITY[incoming] > TOAST_PRIORITY[current] ? incoming : current;
}

function getContainer(id: string, className: string) {
  if (typeof document === "undefined") return null;
  let container = document.getElementById(id);
  if (!container) {
    container = document.createElement("div");
    container.id = id;
    document.body.appendChild(container);
  }
  container.className = className;
  return container;
}

function maybeCleanupContainer(channel: ToastChannel) {
  if (typeof document === "undefined") return;
  const config = CHANNEL_CONFIG[channel];
  const container = document.getElementById(config.containerId);
  const state = channelState[channel];
  if (!container) return;
  if (!state.active.length && !state.queue.length) {
    container.remove();
  }
}

function getSummaryContent(channel: ToastChannel, count: number) {
  const config = CHANNEL_CONFIG[channel];
  return `还有 ${count} ${config.summaryLabel}`;
}

function createSummaryPayload(channel: ToastChannel, type: ToastType, count: number): ToastPayload {
  const config = CHANNEL_CONFIG[channel];
  return {
    id: `octo-summary-${channel}`,
    channel,
    type,
    title: config.summaryTitle,
    content: getSummaryContent(channel, count),
    duration: config.summaryDuration,
    closable: true,
    count,
    createdAt: Date.now(),
    signature: "__summary__",
    isSummary: true,
  };
}

function applyTone(instance: ToastInstance) {
  const meta = TOAST_META[instance.payload.type];
  instance.root.className = [
    "pointer-events-auto relative w-full overflow-hidden rounded-[1.1rem] border shadow-[0_20px_50px_-28px_rgba(15,23,42,0.42)] backdrop-blur-md transition-[opacity,transform] duration-200 ease-out",
    meta.borderClass,
  ].join(" ");
  instance.iconWrapEl.className = [
    "mt-0.5 flex h-9 w-9 flex-shrink-0 items-center justify-center rounded-xl",
    meta.iconWrapClass,
  ].join(" ");
  instance.iconTextEl.className = [
    "text-sm font-bold leading-none",
    meta.iconTextClass,
  ].join(" ");
  instance.iconTextEl.textContent = meta.symbol;
  instance.titleEl.className = [
    "truncate text-[0.8rem] font-semibold leading-5",
    meta.titleClass,
  ].join(" ");
  instance.progressEl.style.background = meta.progressColor;
  instance.glowEl.style.background = meta.glow;
}

function updateCountBadge(instance: ToastInstance) {
  if (instance.payload.isSummary || instance.payload.count <= 1) {
    instance.countEl.style.display = "none";
    instance.countEl.textContent = "";
    return;
  }
  instance.countEl.style.display = "inline-flex";
  instance.countEl.textContent = `x${instance.payload.count}`;
}

function updateText(instance: ToastInstance) {
  const hasTitle = Boolean(instance.payload.title);
  instance.titleEl.style.display = hasTitle ? "block" : "none";
  instance.titleEl.textContent = instance.payload.title ?? "";
  instance.contentEl.className = hasTitle
    ? "mt-0.5 text-[0.82rem] leading-5 text-slate-600"
    : "text-[0.88rem] font-medium leading-5 text-slate-700";
  instance.contentEl.textContent = instance.payload.content;
}

function clearInstanceTimers(instance: ToastInstance) {
  if (instance.timerId) {
    window.clearTimeout(instance.timerId);
    instance.timerId = undefined;
  }
  if (instance.cleanupId) {
    window.clearTimeout(instance.cleanupId);
    instance.cleanupId = undefined;
  }
}

function restartAutoClose(instance: ToastInstance) {
  clearInstanceTimers(instance);
  instance.progressEl.style.transitionDuration = "0ms";
  instance.progressEl.style.transform = "scaleX(1)";
  requestAnimationFrame(() => {
    instance.progressEl.style.transitionDuration = `${Math.max(instance.payload.duration, 800)}ms`;
    instance.progressEl.style.transform = "scaleX(0)";
  });
  instance.timerId = window.setTimeout(instance.close, Math.max(instance.payload.duration, 800));
}

function flushQueue(channel: ToastChannel) {
  const state = channelState[channel];
  const config = CHANNEL_CONFIG[channel];
  while (state.active.length < config.maxVisible && state.queue.length > 0) {
    const next = state.queue.shift();
    if (!next) break;
    showToast(next);
  }
}

function removeQueuedToast(channel: ToastChannel, id: string) {
  const state = channelState[channel];
  state.queue = state.queue.filter((item) => item.id !== id);
  maybeCleanupContainer(channel);
}

function dismissToast(channel: ToastChannel, id: string) {
  const state = channelState[channel];
  const instance = state.active.find((item) => item.payload.id === id);
  if (!instance) {
    removeQueuedToast(channel, id);
    return;
  }
  if (instance.root.dataset.state === "closing") return;

  instance.root.dataset.state = "closing";
  clearInstanceTimers(instance);
  instance.root.style.opacity = "0";
  instance.root.style.transform = channel === "message"
    ? "translateY(-8px) scale(0.98)"
    : "translateX(8px) scale(0.98)";

  instance.cleanupId = window.setTimeout(() => {
    instance.root.remove();
    state.active = state.active.filter((item) => item.payload.id !== id);
    flushQueue(channel);
    maybeCleanupContainer(channel);
  }, 180);
}

function createToastInstance(payload: ToastPayload): ToastInstance | null {
  const config = CHANNEL_CONFIG[payload.channel];
  const container = getContainer(config.containerId, config.containerClass);
  if (!container) return null;

  const root = document.createElement("div");
  root.dataset.toastId = payload.id;
  root.style.opacity = "0";
  root.style.transform = payload.channel === "message"
    ? "translateY(-8px) scale(0.98)"
    : "translateX(8px) scale(0.98)";

  const glowEl = document.createElement("div");
  glowEl.className = "pointer-events-none absolute inset-0 opacity-90";
  root.appendChild(glowEl);

  const rowEl = document.createElement("div");
  rowEl.className = "relative flex items-start gap-3 px-3.5 py-3";
  root.appendChild(rowEl);

  const iconWrapEl = document.createElement("div");
  const iconTextEl = document.createElement("span");
  iconWrapEl.appendChild(iconTextEl);
  rowEl.appendChild(iconWrapEl);

  const textWrapEl = document.createElement("div");
  textWrapEl.className = "min-w-0 flex-1";

  const titleEl = document.createElement("div");
  textWrapEl.appendChild(titleEl);

  const contentEl = document.createElement("div");
  textWrapEl.appendChild(contentEl);

  rowEl.appendChild(textWrapEl);

  const actionsEl = document.createElement("div");
  actionsEl.className = "ml-auto flex items-start gap-2";

  const countEl = document.createElement("span");
  countEl.className = "hidden min-w-[2rem] items-center justify-center rounded-full bg-slate-900/[0.06] px-2 py-1 text-[11px] font-semibold leading-none text-slate-500";
  actionsEl.appendChild(countEl);

  if (payload.closable) {
    const closeBtn = document.createElement("button");
    closeBtn.type = "button";
    closeBtn.className = "inline-flex h-7 w-7 items-center justify-center rounded-full text-slate-400 transition-colors hover:bg-slate-900/[0.06] hover:text-slate-700";
    closeBtn.setAttribute("aria-label", "关闭消息");
    closeBtn.textContent = "×";
    actionsEl.appendChild(closeBtn);
  }

  rowEl.appendChild(actionsEl);

  const progressTrackEl = document.createElement("div");
  progressTrackEl.className = "absolute inset-x-0 bottom-0 h-[2px] bg-slate-900/[0.05]";
  const progressEl = document.createElement("div");
  progressEl.className = "h-full origin-left";
  progressTrackEl.appendChild(progressEl);
  root.appendChild(progressTrackEl);

  const instance: ToastInstance = {
    payload,
    root,
    glowEl,
    iconWrapEl,
    iconTextEl,
    titleEl,
    contentEl,
    countEl,
    progressEl,
    close: () => dismissToast(payload.channel, payload.id),
  };

  const closeButton = actionsEl.querySelector("button");
  if (closeButton) {
    closeButton.addEventListener("click", instance.close);
  }

  applyTone(instance);
  updateText(instance);
  updateCountBadge(instance);
  container.appendChild(root);

  requestAnimationFrame(() => {
    root.style.opacity = "1";
    root.style.transform = "translate(0, 0) scale(1)";
  });

  restartAutoClose(instance);
  return instance;
}

function showToast(payload: ToastPayload) {
  const instance = createToastInstance(payload);
  if (!instance) return noopHandle();
  channelState[payload.channel].active.push(instance);
  return createHandle(instance.close);
}

function mergeWithRecentToast(payload: ToastPayload): ToastHandle | null {
  const state = channelState[payload.channel];
  const config = CHANNEL_CONFIG[payload.channel];
  const now = Date.now();

  const activeMatch = state.active.find(
    (item) =>
      !item.payload.isSummary &&
      item.payload.signature === payload.signature &&
      now - item.payload.createdAt <= config.dedupeWindow
  );
  if (activeMatch) {
    activeMatch.payload.count += payload.count;
    activeMatch.payload.createdAt = now;
    activeMatch.payload.duration = Math.max(activeMatch.payload.duration, payload.duration);
    updateCountBadge(activeMatch);
    restartAutoClose(activeMatch);
    return createHandle(activeMatch.close);
  }

  const queuedMatch = state.queue.find(
    (item) =>
      !item.isSummary &&
      item.signature === payload.signature &&
      now - item.createdAt <= config.dedupeWindow
  );
  if (queuedMatch) {
    queuedMatch.count += payload.count;
    queuedMatch.createdAt = now;
    queuedMatch.duration = Math.max(queuedMatch.duration, payload.duration);
    return createHandle(() => removeQueuedToast(payload.channel, queuedMatch.id));
  }

  return null;
}

function upsertSummaryToast(channel: ToastChannel, type: ToastType, addedCount: number) {
  const state = channelState[channel];

  const activeSummary = state.active.find((item) => item.payload.isSummary);
  if (activeSummary) {
    activeSummary.payload.type = promoteType(activeSummary.payload.type, type);
    activeSummary.payload.count += addedCount;
    activeSummary.payload.content = getSummaryContent(channel, activeSummary.payload.count);
    activeSummary.payload.createdAt = Date.now();
    activeSummary.payload.duration = CHANNEL_CONFIG[channel].summaryDuration;
    applyTone(activeSummary);
    updateText(activeSummary);
    restartAutoClose(activeSummary);
    return createHandle(activeSummary.close);
  }

  const queuedSummary = state.queue.find((item) => item.isSummary);
  if (queuedSummary) {
    queuedSummary.type = promoteType(queuedSummary.type, type);
    queuedSummary.count += addedCount;
    queuedSummary.content = getSummaryContent(channel, queuedSummary.count);
    queuedSummary.createdAt = Date.now();
    queuedSummary.duration = CHANNEL_CONFIG[channel].summaryDuration;
    return createHandle(() => removeQueuedToast(channel, queuedSummary.id));
  }

  const config = CHANNEL_CONFIG[channel];
  let collapsedCount = addedCount;
  let summaryType = type;

  if (state.queue.length >= config.maxQueue) {
    const replaced = state.queue.pop();
    if (replaced) {
      collapsedCount += replaced.count;
      summaryType = promoteType(summaryType, replaced.type);
    }
  }

  const summary = createSummaryPayload(channel, summaryType, collapsedCount);
  if (state.active.length < config.maxVisible) {
    return showToast(summary);
  }

  state.queue.push(summary);
  return createHandle(() => removeQueuedToast(channel, summary.id));
}

function enqueueToast(
  channel: ToastChannel,
  type: ToastType,
  normalized: { title?: string; content: string; duration: number; closable: boolean }
) {
  const trimmedTitle = normalized.title?.trim() || undefined;
  const trimmedContent = normalized.content.trim();
  if (!trimmedTitle && !trimmedContent) {
    return noopHandle();
  }

  const payload: ToastPayload = {
    id: nextToastId(),
    channel,
    type,
    title: trimmedTitle,
    content: trimmedContent,
    duration: normalized.duration,
    closable: normalized.closable,
    count: 1,
    createdAt: Date.now(),
    signature: buildSignature(type, trimmedTitle, trimmedContent),
  };

  const merged = mergeWithRecentToast(payload);
  if (merged) return merged;

  const state = channelState[channel];
  const config = CHANNEL_CONFIG[channel];
  if (state.active.length < config.maxVisible) {
    return showToast(payload);
  }
  if (state.queue.length < config.maxQueue) {
    state.queue.push(payload);
    return createHandle(() => removeQueuedToast(channel, payload.id));
  }
  return upsertSummaryToast(channel, type, payload.count);
}

function message(type: ToastType, options: string | MessageOptions) {
  const normalized = normalizeMessageOptions(options);
  return enqueueToast("message", type, normalized);
}

function notification(type: ToastType, options: string | NotificationOptions) {
  const normalized = normalizeNotificationOptions(options);
  return enqueueToast("notification", type, normalized);
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
