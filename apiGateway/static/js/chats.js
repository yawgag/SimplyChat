import { elements } from "./dom.js";
import { state } from "./state.js";
import { log } from "./logger.js";
import { openAddUserModal, openChatTypeModal, closeChatTypeModal, closeAddUserModal } from "./modals.js";

export function highlightChatWithNewMessage(chatId) {
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

export function markActiveChat(chatElement) {
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

export function renderChatList(chatItems = [], sendEvent) {
  elements.chatList.innerHTML = "";
  if (chatItems.length === 0) {
    elements.chatList.innerHTML = "<p class='text-gray-500'>Нет активных чатов</p>";
    return;
  }

  chatItems.forEach((chat) => {
    const li = document.createElement("li");
    li.id = `chat-item-${chat.chat_id}`;
    li.className = "p-2 border-b border-gray-200 hover:bg-gray-100 flex justify-between items-center cursor-pointer";

    li.onclick = () => {
      state.activeChatId = chat.chat_id;
      sendEvent("set_active_chat", { login: state.login, chat_id: chat.chat_id });
      markActiveChat(li);
    };

    const span = document.createElement("span");
    span.textContent = `Чат #${chat.chat_id}`;
    span.className = "flex-1";

    const addButton = document.createElement("button");
    addButton.textContent = "⋮";
    addButton.className = "text-sm bg-blue-100 text-blue-600 px-2 py-1 rounded ml-2";
    addButton.onclick = (event) => {
      event.stopPropagation();
      openAddUserModal(chat.chat_id);
    };

    li.appendChild(span);
    li.appendChild(addButton);
    elements.chatList.appendChild(li);
  });
}

export function addChatIfMissing(chatId, sendEvent) {
  const newChat = { chat_id: chatId };
  if (!state.chats.some((chat) => chat.chat_id === newChat.chat_id)) {
    state.chats.push(newChat);
    renderChatList(state.chats, sendEvent);
  }
}

export function initChatControls(sendEvent) {
  elements.newChatBtn.onclick = () => {
    openChatTypeModal();
  };

  elements.publicChatBtn.onclick = () => {
    closeChatTypeModal();
    sendEvent("new_chat", { chat_type: "public" });
  };

  elements.privateChatBtn.onclick = () => {
    closeChatTypeModal();
    sendEvent("new_chat", { chat_type: "private" });
  };

  elements.cancelNewChatBtn.onclick = () => {
    closeChatTypeModal();
  };

  elements.cancelAddUserBtn.onclick = () => {
    closeAddUserModal();
  };

  elements.confirmAddUserBtn.onclick = () => {
    const userLogin = elements.addUserInput.value;
    if (!userLogin || !state.selectedChatForAddUser) {
      log("⚠️ Не указан логин пользователя или чат");
      return;
    }

    sendEvent("add_user_to_chat", {
      chat_id: state.selectedChatForAddUser,
      user_login: userLogin,
    });

    closeAddUserModal();
  };

  elements.addUserModal.addEventListener("click", (event) => {
    if (event.target === elements.addUserModal) {
      closeAddUserModal();
    }
  });

  elements.chatTypeModal.addEventListener("click", (event) => {
    if (event.target === elements.chatTypeModal) {
      closeChatTypeModal();
    }
  });
}
