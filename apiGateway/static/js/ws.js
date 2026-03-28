import { state } from "./state.js";
import { log } from "./logger.js";
import { checkAuth, refreshTokens } from "./api.js";

export async function connectWebSocket(handlers) {
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
    log(`🔄 Подключение к WebSocket с логином: ${state.login}`);
    state.socket = new WebSocket(`ws://localhost:8080/chat?login=${encodeURIComponent(state.login)}`);

    state.socket.onopen = () => {
      log("✅ WebSocket подключен");
      sendEvent("all_user_chats", {});
    };

    state.socket.onmessage = (event) => {
      try {
        const message = JSON.parse(event.data);
        log(`📡 Получено: ${JSON.stringify(message)}`);

        switch (message.event_type) {
          case "all_user_chats":
            handlers.onAllUserChats?.(message.data);
            break;
          case "message_history":
            handlers.onMessageHistory?.(message.data);
            break;
          case "message_history_error":
            handlers.onMessageHistoryError?.(message.data);
            break;
          case "add_user_to_chat":
            handlers.onAddUserToChat?.(message.data);
            break;
          case "send_message":
            handlers.onSendMessage?.(message.data);
            break;
          default:
            log(`⚠️ Неизвестный тип события: ${message.event_type}`);
        }
      } catch (error) {
        log(`❌ Ошибка парсинга JSON: ${error.message}`);
        log(`🔹 Полученные данные: ${event.data}`);
      }
    };

    state.socket.onerror = (error) => {
      log(`❌ Ошибка WebSocket: ${error.message || "unknown error"}`);
      setTimeout(() => {
        log("🔄 Попытка переподключения...");
        connectWebSocket(handlers);
      }, 5000);
    };

    state.socket.onclose = (event) => {
      log(`🔌 Соединение закрыто: ${event.code} ${event.reason}`);
      if (event.code !== 1000 && event.code !== 1001) {
        setTimeout(() => {
          log("🔄 Попытка переподключения...");
          connectWebSocket(handlers);
        }, 5000);
      }
    };
  } catch (error) {
    log(`❌ Ошибка подключения: ${error.message}`);
    setTimeout(() => {
      connectWebSocket(handlers);
    }, 5000);
  }
}

export function sendEvent(type, data) {
  if (!state.socket || state.socket.readyState !== WebSocket.OPEN) {
    log(`⚠️ Не могу отправить событие "${type}": соединение не установлено`);
    return;
  }

  const payload = { event_type: type, data };
  log(`📤 Отправлено: ${JSON.stringify(payload)}`);
  state.socket.send(JSON.stringify(payload));
}

export function requestMessageHistory({ login, chatId, limit, before = null }) {
  sendEvent("get_message_history", {
    login,
    chat_id: chatId,
    limit,
    before,
  });
}
