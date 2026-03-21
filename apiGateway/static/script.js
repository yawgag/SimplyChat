let socket = null;
let activeChatId = null;
let login = "";
let selectedChatForAddUser = null;
let chats = [];
let selectedFiles = [];

const MAX_FILES_PER_MESSAGE = 10;

// DOM элементы
const logs = document.getElementById("logs");
const chatList = document.getElementById("chatList");
const messageHistory = document.getElementById("messageHistory");
const logoutBtn = document.getElementById("logoutBtn");
const sendMessageBtn = document.getElementById("sendMessageBtn");
const messageInput = document.getElementById("messageInput");
const newChatBtn = document.getElementById("newChatBtn");
const chatTypeModal = document.getElementById("chatTypeModal");
const cancelNewChatBtn = document.getElementById("cancelNewChatBtn");
const publicChatBtn = document.getElementById("publicChatBtn");
const privateChatBtn = document.getElementById("privateChatBtn");
const addUserModal = document.getElementById("addUserModal");
const addUserInput = document.getElementById("addUserInput");
const cancelAddUserBtn = document.getElementById("cancelAddUserBtn");
const confirmAddUserBtn = document.getElementById("confirmAddUserBtn");
const fileInput = document.getElementById("fileInput");
const attachFileBtn = document.getElementById("attachFileBtn");
const selectedFilesPanel = document.getElementById("selectedFilesPanel");
const selectedFilesList = document.getElementById("selectedFilesList");
const clearFilesBtn = document.getElementById("clearFilesBtn");

function log(msg) {
  logs.innerHTML += `<div>${msg}</div>`;
  logs.scrollTop = logs.scrollHeight;
}

function formatFileSize(bytes) {
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

function buildDownloadURL(fileId) {
  return `/files/${encodeURIComponent(fileId)}/download`;
}

function buildContentURL(fileId) {
  return `/files/${encodeURIComponent(fileId)}/content`;
}

function normalizeMessage(message) {
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

function appendMessageToHistory(message) {
  const normalized = normalizeMessage(message);
  if (messageHistory.firstElementChild?.dataset?.emptyState === "true") {
    messageHistory.innerHTML = "";
  }

  messageHistory.appendChild(renderMessageItem(normalized));
  messageHistory.scrollTop = messageHistory.scrollHeight;
}

function renderMessageHistory(messages) {
  messageHistory.innerHTML = "";

  if (!messages || !Array.isArray(messages) || messages.length === 0) {
    const emptyState = document.createElement("div");
    emptyState.dataset.emptyState = "true";
    emptyState.className = "mb-2 text-gray-500";
    emptyState.textContent = "Нет сообщений в этом чате";
    messageHistory.appendChild(emptyState);
    return;
  }

  messages.forEach((message) => {
    messageHistory.appendChild(renderMessageItem(message));
  });
  messageHistory.scrollTop = messageHistory.scrollHeight;
}

function renderSelectedFiles() {
  selectedFilesList.innerHTML = "";

  if (selectedFiles.length === 0) {
    selectedFilesPanel.classList.add("hidden");
    return;
  }

  selectedFilesPanel.classList.remove("hidden");

  selectedFiles.forEach((file, index) => {
    const row = document.createElement("div");
    row.className = "flex items-center justify-between gap-3 rounded border border-gray-200 bg-white px-3 py-2";

    const meta = document.createElement("div");
    meta.className = "min-w-0";

    const name = document.createElement("div");
    name.className = "truncate font-medium text-gray-800";
    name.textContent = file.name;

    const size = document.createElement("div");
    size.className = "text-xs text-gray-500";
    size.textContent = formatFileSize(file.size);

    meta.appendChild(name);
    meta.appendChild(size);

    const removeBtn = document.createElement("button");
    removeBtn.className = "shrink-0 text-sm text-red-500";
    removeBtn.textContent = "Убрать";
    removeBtn.onclick = () => removeSelectedFile(index);

    row.appendChild(meta);
    row.appendChild(removeBtn);
    selectedFilesList.appendChild(row);
  });
}

function clearSelectedFiles() {
  selectedFiles = [];
  fileInput.value = "";
  renderSelectedFiles();
}

function removeSelectedFile(index) {
  selectedFiles.splice(index, 1);
  renderSelectedFiles();
}

function addSelectedFiles(files) {
  const nextFiles = [...selectedFiles];
  for (const file of files) {
    if (nextFiles.length >= MAX_FILES_PER_MESSAGE) {
      log(`⚠️ Можно выбрать максимум ${MAX_FILES_PER_MESSAGE} файлов`);
      break;
    }
    nextFiles.push(file);
  }

  selectedFiles = nextFiles;
  renderSelectedFiles();
}

// Проверка текущей аутентификации
async function checkAuth() {
  try {
    const response = await fetch("/api/auth/check", {
      method: "GET",
      credentials: "include",
    });
    return response.ok;
  } catch (error) {
    console.error("Auth check failed:", error);
    return false;
  }
}

// Обновление токенов
async function refreshTokens() {
  try {
    const response = await fetch("/api/auth/refresh", {
      method: "POST",
      credentials: "include",
    });
    return response.ok;
  } catch (error) {
    console.error("Token refresh failed:", error);
    return false;
  }
}

function highlightChatWithNewMessage(chatId) {
  const chatElement = document.getElementById(`chat-item-${chatId}`);
  if (!chatElement) {
    return;
  }

  if (!chatElement.querySelector(".new-message-indicator")) {
    const indicator = document.createElement("span");
    indicator.className = "new-message-indicator bg-red-500 text-white text-xs rounded-full px-2 py-1 ml-2";
    indicator.textContent = "Новое";
    chatElement.appendChild(indicator);
  }

  chatElement.classList.add("bg-yellow-100");
}

function markActiveChat(chatElement) {
  document.querySelectorAll("#chatList li").forEach((item) => {
    item.classList.remove("bg-blue-100", "bg-yellow-100");
    const indicator = item.querySelector(".new-message-indicator");
    if (indicator) {
      indicator.remove();
    }
  });

  if (chatElement) {
    chatElement.classList.add("bg-blue-100");
  }
}

// Подключение к WebSocket
async function connectWebSocket() {
  const isAuth = await checkAuth();
  if (!isAuth) {
    const tokensRefreshed = await refreshTokens();
    if (!tokensRefreshed) {
      log("❌ Аутентификация не удалась, перенаправляем на страницу входа");
      window.location.href = "/auth.html";
      return;
    }
  }

  try {
    log(`🔄 Подключение к WebSocket с логином: ${login}`);
    socket = new WebSocket(`ws://localhost:8080/chat?login=${encodeURIComponent(login)}`);

    socket.onopen = () => {
      log("✅ WebSocket подключен");
      sendEvent("all_user_chats", {});
    };

    socket.onmessage = (event) => {
      try {
        const msg = JSON.parse(event.data);
        log(`📡 Получено: ${JSON.stringify(msg)}`);

        switch (msg.event_type) {
          case "all_user_chats":
            chats = Array.isArray(msg.data) ? msg.data : [];
            renderChatList(chats);
            break;
          case "message_history":
            renderMessageHistory(msg.data);
            break;
          case "add_user_to_chat": {
            const newChat = { chat_id: msg.data.chat_id };
            if (!chats.some((chat) => chat.chat_id === newChat.chat_id)) {
              chats.push(newChat);
              renderChatList(chats);
            }
            break;
          }
          case "send_message":
            if (msg.data.chat_id === activeChatId) {
              appendMessageToHistory(msg.data);
            } else {
              highlightChatWithNewMessage(msg.data.chat_id);
            }
            break;
          default:
            log(`⚠️ Неизвестный тип события: ${msg.event_type}`);
        }
      } catch (error) {
        log(`❌ Ошибка парсинга JSON: ${error.message}`);
        log(`🔹 Полученные данные: ${event.data}`);
      }
    };

    socket.onerror = (err) => {
      log(`❌ Ошибка WebSocket: ${err.message || "unknown error"}`);
      setTimeout(() => {
        log("🔄 Попытка переподключения...");
        connectWebSocket();
      }, 5000);
    };

    socket.onclose = (event) => {
      log(`🔌 Соединение закрыто: ${event.code} ${event.reason}`);
      if (event.code !== 1000 && event.code !== 1001) {
        setTimeout(() => {
          log("🔄 Попытка переподключения...");
          connectWebSocket();
        }, 5000);
      }
    };
  } catch (err) {
    log(`❌ Ошибка подключения: ${err.message}`);
    setTimeout(() => {
      connectWebSocket();
    }, 5000);
  }
}

function sendEvent(type, data) {
  if (!socket || socket.readyState !== WebSocket.OPEN) {
    log(`⚠️ Не могу отправить событие "${type}": соединение не установлено`);
    return;
  }

  const payload = { event_type: type, data };
  log(`📤 Отправлено: ${JSON.stringify(payload)}`);
  socket.send(JSON.stringify(payload));
}

function renderChatList(chatItems = []) {
  chatList.innerHTML = "";
  if (chatItems.length === 0) {
    chatList.innerHTML = "<p class='text-gray-500'>Нет активных чатов</p>";
    return;
  }

  chatItems.forEach((chat) => {
    const li = document.createElement("li");
    li.id = `chat-item-${chat.chat_id}`;
    li.className = "p-2 border-b border-gray-200 hover:bg-gray-100 flex justify-between items-center cursor-pointer";

    li.onclick = () => {
      activeChatId = chat.chat_id;
      sendEvent("set_active_chat", { login, chat_id: chat.chat_id });
      markActiveChat(li);
    };

    const span = document.createElement("span");
    span.textContent = `Чат #${chat.chat_id}`;
    span.className = "flex-1";

    const addButton = document.createElement("button");
    addButton.textContent = "⋮";
    addButton.className = "text-sm bg-blue-100 text-blue-600 px-2 py-1 rounded ml-2";
    addButton.onclick = (e) => {
      e.stopPropagation();
      selectedChatForAddUser = chat.chat_id;
      addUserModal.classList.remove("hidden");
      addUserModal.classList.add("flex");
    };

    li.appendChild(span);
    li.appendChild(addButton);
    chatList.appendChild(li);
  });
}

async function sendTextMessage() {
  if (!activeChatId) {
    log("⚠️ Чат не выбран");
    return;
  }

  const content = messageInput.value.trim();
  if (!content) {
    log("⚠️ Сообщение пустое");
    return;
  }

  const event = {
    chat_id: activeChatId,
    sender_login: login,
    content,
    created_at: new Date().toISOString(),
  };

  sendEvent("send_message", event);
  appendMessageToHistory({
    ...event,
    kind: "text",
    attachments: [],
  });
  messageInput.value = "";
}

async function sendFileMessage() {
  if (!activeChatId) {
    log("⚠️ Чат не выбран");
    return;
  }

  if (selectedFiles.length === 0) {
    log("⚠️ Нет выбранных файлов");
    return;
  }

  if (selectedFiles.length > MAX_FILES_PER_MESSAGE) {
    log(`⚠️ Можно отправить максимум ${MAX_FILES_PER_MESSAGE} файлов`);
    return;
  }

  if (messageInput.value.trim()) {
    log("ℹ️ Подпись к файлам пока не поддерживается, текст будет проигнорирован");
  }

  const formData = new FormData();
  selectedFiles.forEach((file) => {
    formData.append("files", file);
  });

  try {
    const response = await fetch(`/chats/${activeChatId}/messages/files?login=${encodeURIComponent(login)}`, {
      method: "POST",
      body: formData,
      credentials: "include",
    });

    if (!response.ok) {
      let message = `Ошибка загрузки файлов: ${response.status}`;
      const rawBody = await response.text();
      if (rawBody) {
        try {
          const errorPayload = JSON.parse(rawBody);
          if (errorPayload?.message) {
            message = errorPayload.message;
          } else if (errorPayload?.error?.message) {
            message = errorPayload.error.message;
          } else {
            message = rawBody;
          }
        } catch (_) {
          message = rawBody;
        }
      }
      log(`❌ ${message}`);
      return;
    }

    const payload = await response.json();
    appendMessageToHistory(payload);
    clearSelectedFiles();
    messageInput.value = "";
    log("✅ Файлы отправлены");
  } catch (error) {
    log(`❌ Ошибка сети при отправке файлов: ${error.message}`);
  }
}

document.addEventListener("DOMContentLoaded", async () => {
  log("🔄 Инициализация чата...");

  login = localStorage.getItem("userLogin");
  if (!login) {
    log("❌ Логин не найден в localStorage, перенаправляем на страницу входа");
    window.location.href = "/auth.html";
    return;
  }

  await connectWebSocket();

  logoutBtn.onclick = async () => {
    log("🚪 Пользователь нажал кнопку выхода");
    try {
      await fetch("/logout", {
        method: "POST",
        credentials: "include",
      });
      localStorage.removeItem("userLogin");
      if (socket) {
        socket.close(1000, "User logged out");
        socket = null;
      }
      log("✅ Выход выполнен успешно");
      window.location.href = "/auth.html";
    } catch (err) {
      log(`❌ Ошибка при выходе: ${err.message}`);
    }
  };

  attachFileBtn.onclick = () => {
    fileInput.click();
  };

  fileInput.onchange = (event) => {
    const files = Array.from(event.target.files || []);
    addSelectedFiles(files);
    fileInput.value = "";
  };

  clearFilesBtn.onclick = () => {
    clearSelectedFiles();
  };

  sendMessageBtn.onclick = async () => {
    if (selectedFiles.length > 0) {
      await sendFileMessage();
      return;
    }
    await sendTextMessage();
  };

  messageInput.addEventListener("keydown", async (e) => {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      sendMessageBtn.click();
    }
  });

  newChatBtn.onclick = () => {
    chatTypeModal.classList.remove("hidden");
    chatTypeModal.classList.add("flex");
  };

  publicChatBtn.onclick = () => {
    chatTypeModal.classList.add("hidden");
    sendEvent("new_chat", { chat_type: "public" });
  };

  privateChatBtn.onclick = () => {
    chatTypeModal.classList.add("hidden");
    sendEvent("new_chat", { chat_type: "private" });
  };

  cancelNewChatBtn.onclick = () => {
    chatTypeModal.classList.add("hidden");
  };

  cancelAddUserBtn.onclick = () => {
    addUserModal.classList.add("hidden");
    addUserInput.value = "";
  };

  confirmAddUserBtn.onclick = () => {
    const userLogin = addUserInput.value;
    if (!userLogin || !selectedChatForAddUser) {
      log("⚠️ Не указан логин пользователя или чат");
      return;
    }

    sendEvent("add_user_to_chat", {
      chat_id: selectedChatForAddUser,
      user_login: userLogin,
    });

    addUserModal.classList.add("hidden");
    addUserInput.value = "";
  };

  addUserModal.addEventListener("click", (e) => {
    if (e.target === addUserModal) {
      addUserModal.classList.add("hidden");
      addUserInput.value = "";
    }
  });

  chatTypeModal.addEventListener("click", (e) => {
    if (e.target === chatTypeModal) {
      chatTypeModal.classList.add("hidden");
    }
  });

  renderSelectedFiles();
  log("✅ Инициализация завершена");
});
