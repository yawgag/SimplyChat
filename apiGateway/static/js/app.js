import { elements } from "./dom.js";
import { state, ensureHistoryState, MESSAGE_HISTORY_PAGE_SIZE } from "./state.js";
import { log } from "./logger.js";
import { connectWebSocket, sendEvent, requestMessageHistory } from "./ws.js";
import {
  appendMessageToHistory,
  prependMessagesToHistory,
  replaceMessageHistory,
  setInitialHistoryLoading,
  setHistoryLoading,
  setHistoryComplete,
} from "./messages.js";
import { renderSelectedFiles, initFileControls } from "./files.js";
import { renderChatList, highlightChatWithNewMessage, addChatIfMissing, initChatControls } from "./chats.js";

const HISTORY_SCROLL_THRESHOLD = 80;
const HISTORY_REQUEST_TIMEOUT_MS = 10000;

function compareMessages(left, right) {
  const leftTime = new Date(left.created_at).getTime();
  const rightTime = new Date(right.created_at).getTime();
  if (leftTime !== rightTime) {
    return leftTime - rightTime;
  }
  return (left.id ?? 0) - (right.id ?? 0);
}

function mergeMessageLists(existingItems, incomingItems) {
  const byId = new Map();

  [...existingItems, ...incomingItems].forEach((message) => {
    byId.set(message.id, message);
  });

  return Array.from(byId.values()).sort(compareMessages);
}

function syncItemIds(chatState) {
  chatState.itemIds = new Set(chatState.items.map((message) => message.id));
}

function clearHistoryRequestTimeout(chatState) {
  if (chatState.requestTimeoutId) {
    clearTimeout(chatState.requestTimeoutId);
    chatState.requestTimeoutId = null;
  }
}

function finishHistoryLoading(chatState) {
  clearHistoryRequestTimeout(chatState);
  chatState.isLoadingInitial = false;
  chatState.isLoadingOlder = false;
}

function failHistoryLoading(chatId, message) {
  const chatState = ensureHistoryState(chatId);
  finishHistoryLoading(chatState);
  if (chatId === state.activeChatId) {
    setInitialHistoryLoading(false);
    setHistoryLoading(false);
    setHistoryComplete(chatState.initialized && !chatState.hasMore && chatState.items.length > 0);
  }
  if (message) {
    log(`❌ ${message}`);
  }
}

function mergeHistoryPage(chatId, page, { prepend = false } = {}) {
  const chatState = ensureHistoryState(chatId);
  const incomingItems = Array.isArray(page?.items) ? page.items : [];
  const insertedItems = incomingItems.filter((message) => !chatState.itemIds.has(message.id));

  if (prepend) {
    chatState.items = mergeMessageLists(insertedItems, chatState.items);
  } else {
    chatState.items = mergeMessageLists(chatState.items, incomingItems);
  }

  syncItemIds(chatState);
  chatState.nextCursor = page?.next_cursor ?? null;
  chatState.hasMore = Boolean(page?.has_more);
  chatState.initialized = true;
  return { chatState, insertedItems };
}

function pushRealtimeMessage(message) {
  const chatState = ensureHistoryState(message.chat_id);
  if (chatState.itemIds.has(message.id)) {
    return false;
  }

  chatState.items = mergeMessageLists(chatState.items, [message]);
  syncItemIds(chatState);
  return true;
}

function renderChatHistory(chatId) {
  const chatState = ensureHistoryState(chatId);
  replaceMessageHistory(chatState.items);
  setHistoryLoading(false);
  setHistoryComplete(chatState.initialized && !chatState.hasMore && chatState.items.length > 0);
}

function loadMessageHistory(chatId, options = {}) {
  const chatState = ensureHistoryState(chatId);
  const isOlder = Boolean(options.before);

  if (isOlder) {
    if (chatState.isLoadingOlder || !chatState.hasMore) {
      return;
    }
    chatState.isLoadingOlder = true;
    setHistoryLoading(true);
  } else {
    if (chatState.isLoadingInitial || chatState.initialized) {
      return;
    }
    chatState.isLoadingInitial = true;
    if (chatState.items.length > 0) {
      replaceMessageHistory(chatState.items);
    } else {
      setInitialHistoryLoading(true);
    }
  }

  clearHistoryRequestTimeout(chatState);
  chatState.requestTimeoutId = setTimeout(() => {
    failHistoryLoading(chatId, "Не удалось загрузить историю сообщений");
  }, HISTORY_REQUEST_TIMEOUT_MS);

  requestMessageHistory({
    login: state.login,
    chatId,
    limit: MESSAGE_HISTORY_PAGE_SIZE,
    before: options.before ?? null,
  });
}

function handleChatSelected(chatId) {
  const chatState = ensureHistoryState(chatId);
  if (chatState.items.length > 0) {
    renderChatHistory(chatId);
  }
  if (!chatState.initialized) {
    loadMessageHistory(chatId);
  }
}

function maybeLoadOlderHistory() {
  const activeChatId = state.activeChatId;
  if (!activeChatId) {
    return;
  }

  const chatState = ensureHistoryState(activeChatId);
  if (
    elements.messageHistory.scrollTop > HISTORY_SCROLL_THRESHOLD ||
    !chatState.initialized ||
    chatState.isLoadingInitial ||
    chatState.isLoadingOlder ||
    !chatState.hasMore ||
    !chatState.nextCursor
  ) {
    return;
  }

  loadMessageHistory(activeChatId, { before: chatState.nextCursor });
}

async function sendTextMessage() {
  if (!state.activeChatId) {
    log("⚠️ Чат не выбран");
    return;
  }

  const content = elements.messageInput.value.trim();
  if (!content) {
    log("⚠️ Сообщение пустое");
    return;
  }

  const event = {
    chat_id: state.activeChatId,
    sender_login: state.login,
    content,
    created_at: new Date().toISOString(),
  };

  sendEvent("send_message", event);
  const chatState = ensureHistoryState(state.activeChatId);
  const optimisticMessage = {
    ...event,
    id: Date.now(),
    kind: "text",
    attachments: [],
  };
  chatState.items = mergeMessageLists(chatState.items, [optimisticMessage]);
  syncItemIds(chatState);
  appendMessageToHistory(optimisticMessage);
  elements.messageInput.value = "";
}

document.addEventListener("DOMContentLoaded", async () => {
  log("🔄 Инициализация чата...");

  state.login = localStorage.getItem("userLogin");
  if (!state.login) {
    log("❌ Логин не найден в localStorage, перенаправляем на страницу входа");
    window.location.href = "/auth.html";
    return;
  }

  await connectWebSocket({
    onAllUserChats(data) {
      state.chats = Array.isArray(data) ? data : [];
      renderChatList(state.chats, handleChatSelected);
    },
    onMessageHistory(data) {
      const chatId = data?.chat_id;
      if (!chatId) {
        return;
      }

      const chatState = ensureHistoryState(chatId);
      const isOlderRequest = chatState.isLoadingOlder;
      finishHistoryLoading(chatState);
      const { insertedItems } = mergeHistoryPage(chatId, data, { prepend: isOlderRequest });

      if (chatId !== state.activeChatId) {
        return;
      }

      setInitialHistoryLoading(false);
      if (isOlderRequest) {
        prependMessagesToHistory(insertedItems);
      } else {
        renderChatHistory(chatId);
      }

      setHistoryLoading(false);
      setHistoryComplete(!chatState.hasMore && chatState.items.length > 0);
    },
    onAddUserToChat(data) {
      addChatIfMissing(data.chat_id, handleChatSelected);
    },
    onMessageHistoryError(data) {
      const chatId = data?.chat_id || state.activeChatId;
      failHistoryLoading(chatId, data?.message || "Не удалось загрузить историю сообщений");
    },
    onSendMessage(data) {
      pushRealtimeMessage(data);
      if (data.chat_id === state.activeChatId) {
        appendMessageToHistory(data);
        const chatState = ensureHistoryState(data.chat_id);
        setHistoryComplete(!chatState.hasMore && chatState.items.length > 0);
      } else {
        highlightChatWithNewMessage(data.chat_id);
      }
    },
  });

  elements.logoutBtn.onclick = async () => {
    log("🚪 Пользователь нажал кнопку выхода");
    try {
      await fetch("/logout", {
        method: "POST",
        credentials: "include",
      });
      localStorage.removeItem("userLogin");
      if (state.socket) {
        state.socket.close(1000, "User logged out");
        state.socket = null;
      }
      log("✅ Выход выполнен успешно");
      window.location.href = "/auth.html";
    } catch (error) {
      log(`❌ Ошибка при выходе: ${error.message}`);
    }
  };

  initFileControls(sendTextMessage);
  initChatControls(sendEvent);
  renderSelectedFiles();
  elements.messageHistory.addEventListener("scroll", maybeLoadOlderHistory);
  log("✅ Инициализация завершена");
});
