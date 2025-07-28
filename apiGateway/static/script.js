let socket = null;
let activeChatId = null;
let login = "";
let selectedChatForAddUser = null;
let chats = [];

// DOM элементы
const logs = document.getElementById("logs");
const chatList = document.getElementById("chatList");
const messageHistory = document.getElementById("messageHistory");
const userLoginElement = document.getElementById("userLogin");
const logoutBtn = document.getElementById("logoutBtn");
const sendMessageBtn = document.getElementById("sendMessageBtn");
const messageInput = document.getElementById("messageInput");
const newChatBtn = document.getElementById("newChatBtn");
const chatTypeModal = document.getElementById("chatTypeModal");
const publicChatBtn = document.getElementById("publicChatBtn");
const privateChatBtn = document.getElementById("privateChatBtn");
const addUserModal = document.getElementById("addUserModal");
const addUserInput = document.getElementById("addUserInput");
const cancelAddUserBtn = document.getElementById("cancelAddUserBtn");
const confirmAddUserBtn = document.getElementById("confirmAddUserBtn");

function log(msg) {
  logs.innerHTML += `<div>${msg}</div>`;
  logs.scrollTop = logs.scrollHeight;
}

// Автоматическое подключение при загрузке страницы
document.addEventListener('DOMContentLoaded', () => {
  login = localStorage.getItem("userLogin");
  if (login) {
    userLoginElement.textContent = `Пользователь: ${login}`;
    connectWebSocket();
  } else {
    // Если нет логина - перенаправляем на страницу аутентификации
    window.location.href = "/auth.html";
  }
});

function connectWebSocket() {
  socket = new WebSocket(`ws://localhost:8081/chat?login=${encodeURIComponent(login)}`);
  
  socket.onopen = () => {
    log("WebSocket подключен");
    sendEvent("all_user_chats", {}); // Запрашиваем список чатов
  };
  
  socket.onmessage = (event) => {
    try {
      const rawMsg = event.data;
      const msg = JSON.parse(rawMsg);
      log(`📡 Получено: ${JSON.stringify(msg)}`);
      
      switch (msg.event_type) {
        case "all_user_chats":
          if (msg.data === null || !Array.isArray(msg.data)) {
            chats = [];
          } else {
            chats = msg.data;
          }
          log(`📄 Получены чаты: ${JSON.stringify(chats)}`);
          renderChatList(chats);
          break;
          
        case "message_history":
          log(`📬 История сообщений: ${JSON.stringify(msg.data)}`);
          renderMessageHistory(msg.data);
          break;
          
        case "add_user_to_chat":
          log(`👥 Пользователь добавлен: ${JSON.stringify(msg.data)}`);
          const newChat = { chat_id: msg.data.chat_id };
          if (!chats.some(chat => chat.chat_id === newChat.chat_id)) {
            chats.push(newChat);
            renderChatList(chats);
          }
          break;
          
        case "new_chat":
          log(`🆕 Создан чат: ${JSON.stringify(msg.data)}`);
          break;
          
        case "set_active_chat":
          log(`👁️ Активный чат установлен: ${JSON.stringify(msg.data)}`);
          break;
          
        case "send_message":
          log(`📩 Получено сообщение: ${JSON.stringify(msg.data)}`);
          if (msg.data.chat_id === activeChatId) {
            const message = document.createElement("div");
            message.className = "mb-2";
            message.textContent = `${msg.data.sender_login}: ${msg.data.content}`;
            messageHistory.appendChild(message);
            messageHistory.scrollTop = messageHistory.scrollHeight;
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
  
  socket.onerror = (err) => log("❌ Ошибка: " + err);
  socket.onclose = () => log("🔌 Соединение закрыто");
}

function renderChatList(chats = []) {
  chatList.innerHTML = "";
  if (chats.length === 0) {
    chatList.innerHTML = "<p class='text-gray-500'>Нет активных чатов</p>";
    return;
  }
  chats.forEach(chat => {
    const li = document.createElement("li");
    li.className = "p-2 border-b border-gray-200 hover:bg-gray-100 flex justify-between items-center";
    const span = document.createElement("span");
    span.textContent = `Чат #${chat.chat_id}`;
    span.className = "cursor-pointer";
    span.onclick = () => {
      activeChatId = chat.chat_id;
      sendEvent("set_active_chat", { login: login, chat_id: chat.chat_id });
    };
    const addButton = document.createElement("button");
    addButton.textContent = "+";
    addButton.className = "text-sm bg-blue-100 text-blue-600 px-2 py-1 rounded";
    addButton.onclick = () => {
      selectedChatForAddUser = chat.chat_id;
      addUserModal.classList.remove("hidden");
    };
    li.appendChild(span);
    li.appendChild(addButton);
    chatList.appendChild(li);
  });
}

function renderMessageHistory(messages) {
  messageHistory.innerHTML = "";
  if (messages && Array.isArray(messages)) {
    messages.forEach(msg => {
      const div = document.createElement("div");
      div.className = "mb-2";
      div.textContent = `${msg.sender_login}: ${msg.content}`;
      messageHistory.appendChild(div);
    });
  } else {
    const div = document.createElement("div");
    div.className = "mb-2 text-gray-500";
    div.textContent = "Нет сообщений в этом чате";
    messageHistory.appendChild(div);
  }
  messageHistory.scrollTop = messageHistory.scrollHeight;
}

function sendEvent(type, data) {
  if (!socket || socket.readyState !== WebSocket.OPEN) return;
  const payload = { event_type: type, data };
  log(`📤 Отправлено: ${JSON.stringify(payload)}`);
  socket.send(JSON.stringify(payload));
}

// Обработчик выхода
logoutBtn.onclick = async () => {
  try {
    await fetch("/logout", { method: "POST" });
    localStorage.removeItem("userLogin");
    if (socket) socket.close();
    window.location.href = "/auth.html";
  } catch (err) {
    log("Ошибка при выходе: " + err);
  }
};

// Остальные обработчики (отправка сообщений, создание чатов и т.д.)
sendMessageBtn.onclick = () => {
  if (!activeChatId) return;
  const content = messageInput.value;
  if (!content) return;
  
  const event = {
    chat_id: activeChatId,
    sender_login: login,
    content: content,
    created_at: new Date().toISOString()
  };
  
  sendEvent("send_message", event);
  
  // Добавляем сообщение в интерфейс
  const message = document.createElement("div");
  message.className = "mb-2";
  message.textContent = `${login}: ${content}`;
  messageHistory.appendChild(message);
  messageHistory.scrollTop = messageHistory.scrollHeight;
  
  messageInput.value = "";
};

// Обработчики для создания чатов и добавления пользователей
newChatBtn.onclick = () => chatTypeModal.classList.remove("hidden");
publicChatBtn.onclick = () => {
  chatTypeModal.classList.add("hidden");
  sendEvent("new_chat", { chat_type: "public" });
};
privateChatBtn.onclick = () => {
  chatTypeModal.classList.add("hidden");
  sendEvent("new_chat", { chat_type: "private" });
};

cancelAddUserBtn.onclick = () => {
  addUserModal.classList.add("hidden");
  addUserInput.value = "";
};
confirmAddUserBtn.onclick = () => {
  const userLogin = addUserInput.value;
  if (!userLogin || !selectedChatForAddUser) return;
  
  sendEvent("add_user_to_chat", {
    chat_id: selectedChatForAddUser,
    user_login: userLogin
  });
  
  addUserModal.classList.add("hidden");
  addUserInput.value = "";
};

addUserModal.addEventListener('click', (e) => {
  if (e.target === addUserModal) {
    addUserModal.classList.add("hidden");
    addUserInput.value = "";
  }
});