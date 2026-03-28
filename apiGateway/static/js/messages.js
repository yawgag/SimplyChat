import { elements } from "./dom.js";
import { buildContentURL, buildDownloadURL } from "./api.js";

const HISTORY_STATUS_ID = "history-status";

export function formatFileSize(bytes) {
  if (!Number.isFinite(bytes) || bytes < 0) {
    return "0 B";
  }

  if (bytes < 1024) {
    return `${bytes} B`;
  }
  if (bytes < 1024 * 1024) {
    return `${(bytes / 1024).toFixed(1)} KB`;
  }
  if (bytes < 1024 * 1024 * 1024) {
    return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
  }
  return `${(bytes / (1024 * 1024 * 1024)).toFixed(1)} GB`;
}

export function normalizeMessage(message) {
  const normalized = {
    id: message?.id ?? 0,
    chat_id: message?.chat_id ?? 0,
    sender_login: message?.sender_login ?? "unknown",
    kind: message?.kind ?? "text",
    content: typeof message?.content === "string" ? message.content : "",
    attachments: Array.isArray(message?.attachments) ? message.attachments : [],
    created_at: message?.created_at ?? new Date().toISOString(),
  };

  normalized.attachments = normalized.attachments.map((attachment) => ({
    file_id: attachment?.file_id ?? "",
    original_filename: attachment?.original_filename ?? "file",
    mime_type: attachment?.mime_type ?? "",
    size: Number.isFinite(attachment?.size) ? attachment.size : 0,
    kind: attachment?.kind ?? "file",
    content_url: attachment?.content_url || buildContentURL(attachment?.file_id ?? ""),
    download_url: attachment?.download_url || buildDownloadURL(attachment?.file_id ?? ""),
  }));

  if (normalized.kind === "file") {
    normalized.content = "";
  }

  return normalized;
}

function isImageAttachment(attachment) {
  return typeof attachment?.mime_type === "string" && attachment.mime_type.toLowerCase().startsWith("image/");
}

function isVideoAttachment(attachment) {
  return typeof attachment?.mime_type === "string" && attachment.mime_type.toLowerCase().startsWith("video/");
}

function renderAttachments(attachments) {
  const wrapper = document.createElement("div");
  wrapper.className = "mt-2 space-y-2";

  attachments.forEach((attachment) => {
    const item = document.createElement("div");
    item.className = "rounded border border-gray-200 bg-gray-50 p-3";

    if (isImageAttachment(attachment)) {
      const image = document.createElement("img");
      image.src = attachment.content_url;
      image.alt = attachment.original_filename;
      image.className = "mb-3 max-h-80 w-full rounded object-contain bg-black/5";
      image.loading = "lazy";
      item.appendChild(image);
    } else if (isVideoAttachment(attachment)) {
      const video = document.createElement("video");
      video.src = attachment.content_url;
      video.controls = true;
      video.preload = "metadata";
      video.className = "mb-3 max-h-80 w-full rounded bg-black";
      item.appendChild(video);
    }

    const contentRow = document.createElement("div");
    contentRow.className = "flex items-center justify-between gap-3";

    const meta = document.createElement("div");
    meta.className = "min-w-0";

    const name = document.createElement("div");
    name.className = "truncate font-medium text-gray-800";
    name.textContent = attachment.original_filename;

    const info = document.createElement("div");
    info.className = "text-xs text-gray-500";
    info.textContent = `${attachment.kind} • ${attachment.mime_type || "unknown"} • ${formatFileSize(attachment.size)}`;

    meta.appendChild(name);
    meta.appendChild(info);

    const link = document.createElement("a");
    link.href = attachment.download_url;
    link.download = attachment.original_filename;
    link.className = "shrink-0 rounded bg-blue-500 px-3 py-1 text-sm text-white hover:bg-blue-600";
    link.textContent = "Скачать";

    contentRow.appendChild(meta);
    contentRow.appendChild(link);
    item.appendChild(contentRow);
    wrapper.appendChild(item);
  });

  return wrapper;
}

function renderMessageItem(message) {
  const msg = normalizeMessage(message);
  const container = document.createElement("div");
  container.dataset.messageId = String(msg.id);
  container.className = "mb-3 rounded border border-gray-200 bg-white p-3";

  const header = document.createElement("div");
  header.className = "mb-1 text-sm font-semibold text-gray-800";
  header.textContent = msg.sender_login;
  container.appendChild(header);

  if (msg.kind === "text") {
    const content = document.createElement("div");
    content.className = "text-sm text-gray-900";
    content.textContent = msg.content;
    container.appendChild(content);
  } else if (msg.attachments.length > 0) {
    container.appendChild(renderAttachments(msg.attachments));
  }

  const footer = document.createElement("div");
  footer.className = "mt-2 text-xs text-gray-400";
  footer.textContent = new Date(msg.created_at).toLocaleString();
  container.appendChild(footer);

  return container;
}

function getHistoryStatusElement() {
  let status = document.getElementById(HISTORY_STATUS_ID);
  if (status) {
    return status;
  }

  status = document.createElement("div");
  status.id = HISTORY_STATUS_ID;
  status.className = "sticky top-0 z-10 hidden mb-3 rounded border border-gray-200 bg-gray-50 px-3 py-2 text-center text-xs text-gray-500";
  elements.messageHistory.prepend(status);
  return status;
}

function setHistoryStatus(text, hidden = false) {
  const status = getHistoryStatusElement();
  status.textContent = text;
  status.classList.toggle("hidden", hidden);
}

function clearEmptyState() {
  const emptyState = elements.messageHistory.querySelector("[data-empty-state='true']");
  if (emptyState) {
    emptyState.remove();
  }
}

function hasMessageNode(messageId) {
  return elements.messageHistory.querySelector(`[data-message-id='${CSS.escape(String(messageId))}']`) !== null;
}

function renderEmptyState() {
  clearEmptyState();
  const emptyState = document.createElement("div");
  emptyState.dataset.emptyState = "true";
  emptyState.className = "mb-2 text-gray-500";
  emptyState.textContent = "Нет сообщений в этом чате";
  elements.messageHistory.appendChild(emptyState);
}

export function setHistoryLoading(isLoadingOlder) {
  if (isLoadingOlder) {
    setHistoryStatus("Загружаем более ранние сообщения...", false);
    return;
  }

  setHistoryStatus("", true);
}

export function setInitialHistoryLoading(isLoading) {
  if (isLoading) {
    elements.messageHistory.innerHTML = "";
    const loader = document.createElement("div");
    loader.dataset.loadingState = "true";
    loader.className = "mb-2 text-gray-500";
    loader.textContent = "Загружаем историю сообщений...";
    elements.messageHistory.appendChild(loader);
    return;
  }

  const loader = elements.messageHistory.querySelector("[data-loading-state='true']");
  if (loader) {
    loader.remove();
  }
}

export function setHistoryComplete(isComplete) {
  if (isComplete) {
    setHistoryStatus("Вся история загружена", false);
    return;
  }

  setHistoryStatus("", true);
}

export function replaceMessageHistory(messages) {
  elements.messageHistory.innerHTML = "";
  setHistoryStatus("", true);

  if (!messages || !Array.isArray(messages) || messages.length === 0) {
    renderEmptyState();
    return;
  }

  clearEmptyState();
  messages.forEach((message) => {
    elements.messageHistory.appendChild(renderMessageItem(message));
  });
  elements.messageHistory.scrollTop = elements.messageHistory.scrollHeight;
}

export function prependMessagesToHistory(messages) {
  if (!messages || messages.length === 0) {
    return;
  }

  clearEmptyState();
  const previousScrollHeight = elements.messageHistory.scrollHeight;
  const previousScrollTop = elements.messageHistory.scrollTop;

  const fragment = document.createDocumentFragment();
  messages.forEach((message) => {
    const normalized = normalizeMessage(message);
    if (hasMessageNode(normalized.id)) {
      return;
    }
    fragment.appendChild(renderMessageItem(normalized));
  });

  const firstMessage = elements.messageHistory.querySelector("[data-message-id]");
  if (firstMessage) {
    elements.messageHistory.insertBefore(fragment, firstMessage);
  } else {
    elements.messageHistory.appendChild(fragment);
  }

  const newScrollHeight = elements.messageHistory.scrollHeight;
  elements.messageHistory.scrollTop = previousScrollTop + (newScrollHeight - previousScrollHeight);
}

export function appendMessageToHistory(message, options = {}) {
  const normalized = normalizeMessage(message);
  if (hasMessageNode(normalized.id)) {
    return;
  }

  clearEmptyState();
  elements.messageHistory.appendChild(renderMessageItem(normalized));

  if (options.stickToBottom !== false) {
    elements.messageHistory.scrollTop = elements.messageHistory.scrollHeight;
  }
}
