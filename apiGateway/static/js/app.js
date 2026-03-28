import { elements } from "./dom.js";
import { state } from "./state.js";
import { log } from "./logger.js";
import { connectWebSocket, sendEvent } from "./ws.js";
import { appendMessageToHistory, renderMessageHistory } from "./messages.js";
import { renderSelectedFiles, initFileControls } from "./files.js";
import { renderChatList, highlightChatWithNewMessage, addChatIfMissing, initChatControls } from "./chats.js";

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
  appendMessageToHistory({
    ...event,
    kind: "text",
    attachments: [],
  });
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
      renderChatList(state.chats, sendEvent);
    },
    onMessageHistory(data) {
      renderMessageHistory(data);
    },
    onAddUserToChat(data) {
      addChatIfMissing(data.chat_id, sendEvent);
    },
    onSendMessage(data) {
      if (data.chat_id === state.activeChatId) {
        appendMessageToHistory(data);
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
  log("✅ Инициализация завершена");
});
