const loginTab = document.getElementById("loginTab");
const registerTab = document.getElementById("registerTab");
const loginForm = document.getElementById("loginForm");
const registerForm = document.getElementById("registerForm");
const authError = document.getElementById("authError");

function activateLoginTab() {
  loginTab.className = "px-4 py-2 flex-1 text-center bg-blue-500 text-white";
  registerTab.className = "px-4 py-2 flex-1 text-center bg-gray-200 text-gray-700 hover:bg-gray-300";
  loginForm.classList.remove("hidden");
  registerForm.classList.add("hidden");
  authError.classList.add("hidden");
}

function activateRegisterTab() {
  registerTab.className = "px-4 py-2 flex-1 text-center bg-blue-500 text-white";
  loginTab.className = "px-4 py-2 flex-1 text-center bg-gray-200 text-gray-700 hover:bg-gray-300";
  registerForm.classList.remove("hidden");
  loginForm.classList.add("hidden");
  authError.classList.add("hidden");
}

function showError(message) {
  authError.textContent = message;
  authError.classList.remove("hidden");
}

async function readErrorMessage(response, fallbackMessage) {
  const rawBody = await response.text();
  if (!rawBody) {
    return fallbackMessage;
  }

  try {
    const payload = JSON.parse(rawBody);
    if (payload?.message) {
      return payload.message;
    }
  } catch (_) {
    // The backend often responds with plain text or an empty body.
  }

  return rawBody || fallbackMessage;
}

loginTab.onclick = () => {
  activateLoginTab();
};

registerTab.onclick = () => {
  activateRegisterTab();
};

loginForm.onsubmit = async (event) => {
  event.preventDefault();
  const login = document.getElementById("loginInput").value;
  const password = document.getElementById("passwordInput").value;

  try {
    const response = await fetch("/login", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ login, password }),
    });

    if (response.ok) {
      localStorage.setItem("userLogin", login);
      window.location.href = "/";
      return;
    }

    const message = await readErrorMessage(response, "Ошибка входа");
    showError(message);
  } catch (_) {
    showError("Ошибка сети");
  }
};

registerForm.onsubmit = async (event) => {
  event.preventDefault();
  const login = document.getElementById("regLoginInput").value;
  const email = document.getElementById("emailInput").value;
  const password = document.getElementById("regPasswordInput").value;

  try {
    const response = await fetch("/register", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ login, email, password }),
    });

    if (response.ok) {
      localStorage.setItem("userLogin", login);
      window.location.href = "/";
      return;
    }

    const message = await readErrorMessage(response, "Ошибка регистрации");
    showError(message);
  } catch (_) {
    showError("Ошибка сети");
  }
};

activateLoginTab();
