export async function checkAuth() {
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

export async function refreshTokens() {
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

export function buildDownloadURL(fileId) {
  return `/files/${encodeURIComponent(fileId)}/download`;
}

export function buildContentURL(fileId) {
  return `/files/${encodeURIComponent(fileId)}/content`;
}

export async function uploadFileMessage(chatId, login, files) {
  const formData = new FormData();
  files.forEach((file) => {
    formData.append("files", file);
  });

  return fetch(`/chats/${chatId}/messages/files?login=${encodeURIComponent(login)}`, {
    method: "POST",
    body: formData,
    credentials: "include",
  });
}
