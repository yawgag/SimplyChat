import { elements } from "./dom.js";
import { state } from "./state.js";

export function openChatTypeModal() {
  elements.chatTypeModal.classList.remove("hidden");
  elements.chatTypeModal.classList.add("flex");
}

export function closeChatTypeModal() {
  elements.chatTypeModal.classList.add("hidden");
  elements.chatTypeModal.classList.remove("flex");
}

export function openAddUserModal(chatId) {
  state.selectedChatForAddUser = chatId;
  elements.addUserModal.classList.remove("hidden");
  elements.addUserModal.classList.add("flex");
}

export function closeAddUserModal() {
  elements.addUserModal.classList.add("hidden");
  elements.addUserModal.classList.remove("flex");
  elements.addUserInput.value = "";
}
