import { elements } from "./dom.js";
import { state, MAX_FILES_PER_MESSAGE, ensureHistoryState } from "./state.js";
import { log } from "./logger.js";
import { formatFileSize, appendMessageToHistory } from "./messages.js";
import { uploadFileMessage } from "./api.js";

export function renderSelectedFiles() {
  elements.selectedFilesList.innerHTML = "";

  if (state.selectedFiles.length === 0) {
    elements.selectedFilesPanel.classList.add("hidden");
    return;
  }

  elements.selectedFilesPanel.classList.remove("hidden");

  state.selectedFiles.forEach((file, index) => {
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
    elements.selectedFilesList.appendChild(row);
  });
}

export function clearSelectedFiles() {
  state.selectedFiles = [];
  elements.fileInput.value = "";
  renderSelectedFiles();
}

export function removeSelectedFile(index) {
  state.selectedFiles.splice(index, 1);
  renderSelectedFiles();
}

export function addSelectedFiles(files) {
  const nextFiles = [...state.selectedFiles];

  for (const file of files) {
    if (nextFiles.length >= MAX_FILES_PER_MESSAGE) {
      log(`⚠️ Можно выбрать максимум ${MAX_FILES_PER_MESSAGE} файлов`);
      break;
    }
    nextFiles.push(file);
  }

  state.selectedFiles = nextFiles;
  renderSelectedFiles();
}

export async function sendFileMessage() {
  if (!state.activeChatId) {
    log("⚠️ Чат не выбран");
    return;
  }

  if (state.selectedFiles.length === 0) {
    log("⚠️ Нет выбранных файлов");
    return;
  }

  if (state.selectedFiles.length > MAX_FILES_PER_MESSAGE) {
    log(`⚠️ Можно отправить максимум ${MAX_FILES_PER_MESSAGE} файлов`);
    return;
  }

  if (elements.messageInput.value.trim()) {
    log("ℹ️ Подпись к файлам пока не поддерживается, текст будет проигнорирован");
  }

  try {
    const response = await uploadFileMessage(state.activeChatId, state.login, state.selectedFiles);

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
    const chatState = ensureHistoryState(payload.chat_id);
    if (!chatState.itemIds.has(payload.id)) {
      chatState.items.push(payload);
      chatState.items.sort((left, right) => {
        const leftTime = new Date(left.created_at).getTime();
        const rightTime = new Date(right.created_at).getTime();
        if (leftTime !== rightTime) {
          return leftTime - rightTime;
        }
        return (left.id ?? 0) - (right.id ?? 0);
      });
      chatState.itemIds = new Set(chatState.items.map((message) => message.id));
    }
    appendMessageToHistory(payload);
    clearSelectedFiles();
    elements.messageInput.value = "";
    log("✅ Файлы отправлены");
  } catch (error) {
    log(`❌ Ошибка сети при отправке файлов: ${error.message}`);
  }
}

export function initFileControls(sendTextMessage) {
  elements.attachFileBtn.onclick = () => {
    elements.fileInput.click();
  };

  elements.fileInput.onchange = (event) => {
    const files = Array.from(event.target.files || []);
    addSelectedFiles(files);
    elements.fileInput.value = "";
  };

  elements.clearFilesBtn.onclick = () => {
    clearSelectedFiles();
  };

  elements.sendMessageBtn.onclick = async () => {
    if (state.selectedFiles.length > 0) {
      await sendFileMessage();
      return;
    }
    await sendTextMessage();
  };

  elements.messageInput.addEventListener("keydown", async (event) => {
    if (event.key === "Enter" && !event.shiftKey) {
      event.preventDefault();
      elements.sendMessageBtn.click();
    }
  });
}
