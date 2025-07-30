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

// Проверка текущей аутентификации
async function checkAuth() {
  try {
    const response = await fetch('/api/auth/check', {
      method: 'GET',
      credentials: 'include'
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
    const response = await fetch('/api/auth/refresh', {
      method: 'POST',
      credentials: 'include'
    });
    return response.ok;
  } catch (error) {
    console.error("Token refresh failed:", error);
    return false;
  }
}

// Подключение к WebSocket
async function connectWebSocket() {
  // 1. Проверяем аутентификацию
  const isAuth = await checkAuth();
  if (!isAuth) {
    // 2. Пробуем обновить токены
    const tokensRefreshed = await refreshTokens();
    if (!tokensRefreshed) {
      log("❌ Аутентификация не удалась, перенаправляем на страницу входа");
      window.location.href = "/auth.html";
      return;
    }
  }

  // 3. Подключаемся к WebSocket
  try {
    log(`🔄 Подключение к WebSocket с логином: ${login}`);
    socket = new WebSocket(`ws://localhost:8080/chat?login=${encodeURIComponent(login)}`);
    
    socket.onopen = () => {
      log("✅ WebSocket подключен");
      sendEvent("all_user_chats", {});
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
            // Проверяем, активен ли этот чат
            if (msg.data.chat_id === activeChatId) {
              const message = document.createElement("div");
              message.className = "mb-2";
              message.textContent = `${msg.data.sender_login}: ${msg.data.content}`;
              messageHistory.appendChild(message);
              messageHistory.scrollTop = messageHistory.scrollHeight;
            } else {
              // ВЫДЕЛЕНИЕ ЧАТА ПРИ ПОЛУЧЕНИИ СООБЩЕНИЯ (ЗАПРОС 2)
              // Ищем элемент чата по ID
              const chatElement = document.getElementById(`chat-item-${msg.data.chat_id}`);
              if (chatElement) {
                // Добавляем индикатор нового сообщения
                if (!chatElement.querySelector('.new-message-indicator')) {
                  const indicator = document.createElement('span');
                  indicator.className = 'new-message-indicator bg-red-500 text-white text-xs rounded-full px-2 py-1 ml-2';
                  indicator.textContent = 'Новое';
                  chatElement.appendChild(indicator);
                }
                
                // Меняем цвет фона
                chatElement.classList.add('bg-yellow-100');
              }
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
      log("❌ Ошибка WebSocket: " + err.message);
      // Попытка переподключения через 5 секунд
      setTimeout(() => {
        log("🔄 Попытка переподключения...");
        connectWebSocket();
      }, 5000);
    };
    
    socket.onclose = (event) => {
      log(`🔌 Соединение закрыто: ${event.code} ${event.reason}`);
      // Если закрыто не по инициативе пользователя
      if (event.code !== 1000 && event.code !== 1001) {
        log("🔄 Попытка переподключения...");
        setTimeout(() => {
          connectWebSocket();
        }, 5000);
      }
    };
  } catch (err) {
    log("❌ Ошибка подключения: " + err.message);
    // Попытка переподключения через 5 секунд
    setTimeout(() => {
      connectWebSocket();
    }, 5000);
  }
}

// Отправка событий на сервер
function sendEvent(type, data) {
  if (!socket || socket.readyState !== WebSocket.OPEN) {
    log(`⚠️ Не могу отправить событие "${type}": соединение не установлено`);
    return;
  }
  
  const payload = { event_type: type, data };
  log(`📤 Отправлено: ${JSON.stringify(payload)}`);
  socket.send(JSON.stringify(payload));
}

// Рендеринг списка чатов
function renderChatList(chats = []) {
  chatList.innerHTML = "";
  if (chats.length === 0) {
    chatList.innerHTML = "<p class='text-gray-500'>Нет активных чатов</p>";
    return;
  }
  
  chats.forEach(chat => {
    const li = document.createElement("li");
    // ДОБАВЛЕН ID ДЛЯ КАЖДОГО ЧАТА (ВАЖНО ДЛЯ ВЫДЕЛЕНИЯ) (ЗАПРОС 2)
    li.id = `chat-item-${chat.chat_id}`;
    li.className = "p-2 border-b border-gray-200 hover:bg-gray-100 flex justify-between items-center cursor-pointer";
    
    // ОТКРЫТИЕ ЧАТА ПРИ КЛИКЕ НА ВЕСЬ БЛОК (ЗАПРОС 1)
    li.onclick = () => {
      activeChatId = chat.chat_id;
      sendEvent("set_active_chat", { login: login, chat_id: chat.chat_id });
      
      // Сбрасываем выделение
      document.querySelectorAll('#chatList li').forEach(item => {
        item.classList.remove('bg-blue-100', 'bg-yellow-100');
        const indicator = item.querySelector('.new-message-indicator');
        if (indicator) indicator.remove();
      });
      
      // Выделяем текущий чат
      li.classList.add('bg-blue-100');
    };

    const span = document.createElement("span");
    span.textContent = `Чат #${chat.chat_id}`;
    span.className = "flex-1"; // Убираем cursor-pointer из span
    
    const addButton = document.createElement("button");
    addButton.textContent = "+";
    addButton.className = "text-sm bg-blue-100 text-blue-600 px-2 py-1 rounded ml-2";
    addButton.onclick = (e) => {
      e.stopPropagation(); // Останавливаем всплытие события
      selectedChatForAddUser = chat.chat_id;
      addUserModal.classList.remove("hidden");
    };
    
    li.appendChild(span);
    li.appendChild(addButton);
    chatList.appendChild(li);
  });
}

// Рендеринг истории сообщений
function renderMessageHistory(messages) {
  messageHistory.innerHTML = "";
  
  // Проверяем, что messages существует и является массивом
  if (messages && Array.isArray(messages)) {
    messages.forEach(msg => {
      const div = document.createElement("div");
      div.className = "mb-2";
      div.textContent = `${msg.sender_login}: ${msg.content}`;
      messageHistory.appendChild(div);
    });
  } else {
    // Если messages пустой или null, выводим сообщение
    const div = document.createElement("div");
    div.className = "mb-2 text-gray-500";
    div.textContent = "Нет сообщений в этом чате";
    messageHistory.appendChild(div);
  }
  
  messageHistory.scrollTop = messageHistory.scrollHeight;
}

// Инициализация при загрузке страницы
document.addEventListener('DOMContentLoaded', async () => {
  log("🔄 Инициализация чата...");
  
  // 1. Получаем логин из localStorage
  login = localStorage.getItem("userLogin");
  if (!login) {
    log("❌ Логин не найден в localStorage, перенаправляем на страницу входа");
    window.location.href = "/auth.html";
    return;
  }
  
  log(`✅ Логин получен из localStorage: ${login}`);
  userLoginElement.textContent = `Пользователь: ${login}`;
  
  // 2. Подключаемся к WebSocket
  await connectWebSocket();
  
  // 3. Обработчик кнопки выхода
  logoutBtn.onclick = async () => {
    log("🚪 Пользователь нажал кнопку выхода");
    try {
      await fetch("/logout", { 
        method: "POST", 
        credentials: "include" 
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
  
  // 4. Обработчик отправки сообщения
  sendMessageBtn.onclick = () => {
    if (!activeChatId) {
      log("⚠️ Чат не выбран");
      return;
    }
    
    const content = messageInput.value;
    if (!content) {
      log("⚠️ Сообщение пустое");
      return;
    }
    
    const event = {
      chat_id: activeChatId,
      sender_login: login,
      content: content,
      created_at: new Date().toISOString()
    };
    
    // Отправляем сообщение на сервер
    sendEvent("send_message", event);
    
    // Добавляем сообщение в интерфейс
    const message = document.createElement("div");
    message.className = "mb-2";
    message.textContent = `${login}: ${content}`;
    messageHistory.appendChild(message);
    messageHistory.scrollTop = messageHistory.scrollHeight;
    
    // Очищаем поле ввода
    messageInput.value = "";
  };
  
  // ОТПРАВКА СООБЩЕНИЯ ПО НАЖАТИЮ ENTER (ЗАПРОС 3)
  messageInput.addEventListener('keydown', (e) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault(); // Предотвращаем перенос строки
      sendMessageBtn.click(); // Имитируем клик по кнопке отправки
    }
  });
  
  // 5. Обработчики для создания чатов
  newChatBtn.onclick = () => {
    log("🆕 Подготовка к созданию нового чата");
    chatTypeModal.classList.remove("hidden");
  };
  
  publicChatBtn.onclick = () => {
    log("🌐 Создание публичного чата");
    chatTypeModal.classList.add("hidden");
    sendEvent("new_chat", { chat_type: "public" });
  };
  
  privateChatBtn.onclick = () => {
    log("🔒 Создание приватного чата");
    chatTypeModal.classList.add("hidden");
    sendEvent("new_chat", { chat_type: "private" });
  };
  
  // 6. Обработчики для добавления пользователей
  cancelAddUserBtn.onclick = () => {
    log("❌ Отмена добавления пользователя");
    addUserModal.classList.add("hidden");
    addUserInput.value = "";
  };
  
  confirmAddUserBtn.onclick = () => {
    const userLogin = addUserInput.value;
    if (!userLogin || !selectedChatForAddUser) {
      log("⚠️ Не указан логин пользователя или чат");
      return;
    }
    
    log(`👥 Добавление пользователя ${userLogin} в чат #${selectedChatForAddUser}`);
    sendEvent("add_user_to_chat", {
      chat_id: selectedChatForAddUser,
      user_login: userLogin
    });
    
    addUserModal.classList.add("hidden");
    addUserInput.value = "";
  };
  
  // Закрытие модального окна при клике вне его
  addUserModal.addEventListener('click', (e) => {
    if (e.target === addUserModal) {
      log("❌ Отмена добавления пользователя (клик вне окна)");
      addUserModal.classList.add("hidden");
      addUserInput.value = "";
    }
  });
  
  log("✅ Инициализация завершена");
});