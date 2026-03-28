export const state = {
  socket: null,
  activeChatId: null,
  login: "",
  selectedChatForAddUser: null,
  chats: [],
  selectedFiles: [],
  messageHistoryByChat: {},
};

export const MAX_FILES_PER_MESSAGE = 10;
export const MESSAGE_HISTORY_PAGE_SIZE = 50;

export function createEmptyHistoryState() {
  return {
    items: [],
    itemIds: new Set(),
    nextCursor: null,
    hasMore: true,
    initialized: false,
    isLoadingInitial: false,
    isLoadingOlder: false,
    requestTimeoutId: null,
  };
}

export function ensureHistoryState(chatId) {
  if (!state.messageHistoryByChat[chatId]) {
    state.messageHistoryByChat[chatId] = createEmptyHistoryState();
  }
  return state.messageHistoryByChat[chatId];
}
