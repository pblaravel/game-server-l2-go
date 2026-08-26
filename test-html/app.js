const I18N = {
  ru: {
    langName: "Русский",
    loginTitle: "ВХОД В ИГРУ",
    registerTitle: "РЕГИСТРАЦИЯ",
    forgotTitle: "СБРОС ПАРОЛЯ",
    successTitle: "АВТОРИЗАЦИЯ",
    login: "Логин",
    password: "Пароль",
    passwordRepeat: "Повторите пароль",
    email: "Email",
    remember: "Запомнить меня",
    forgot: "Забыли пароль?",
    enter: "ВОЙТИ",
    or: "ИЛИ",
    register: "РЕГИСТРАЦИЯ",
    backToLogin: "ВОЙТИ",
    createAccount: "СОЗДАТЬ АККАУНТ",
    sendLink: "ОТПРАВИТЬ",
    forgotHint: "Укажите email — отправим ссылку для сброса пароля.",
    welcome: "Добро пожаловать",
    serverReady: "Сервер готов к подключению.",
    enterWorld: "В ИГРУ",
    logout: "ВЫЙТИ",
    site: "САЙТ",
    forum: "ФОРУМ",
    support: "ПОДДЕРЖКА",
    settings: "НАСТРОЙКИ",
    errLogin: "Введите логин (минимум 3 символа).",
    errPassword: "Введите пароль (минимум 4 символа).",
    errEmail: "Укажите корректный email.",
    errMatch: "Пароли не совпадают.",
    accountCreated: "Аккаунт создан. Можно войти.",
    mailSent: "Письмо для сброса пароля отправлено.",
    connecting: "Подключение к миру Aden...",
    ticketSent: "Обращение отправлено в поддержку.",
    settingsSaved: "Настройки сохранены.",
    save: "СОХРАНИТЬ",
    siteTitle: "Официальный сайт",
    forumTitle: "Форум",
    supportTitle: "Поддержка",
    settingsTitle: "Настройки",
    news1Title: "Осада замка Giran",
    news1Text: "Регистрация кланов на осаду открыта до воскресенья. Награда — казна и контроль налога.",
    news2Title: "Рейд: Queen Ant",
    news2Text: "Босс появляется каждые 24 часа. Рекомендуемый уровень группы — 40+.",
    topic1: "Гайд для новичков Interlude",
    topic2: "Набор в клан Phoenix",
    topic3: "Баг-репорт: окно обмена",
    supportText: "Опишите проблему с входом, персонажем или донатом — ответим в течение суток.",
    category: "Категория",
    catLogin: "Вход в игру",
    catChar: "Персонаж",
    catOther: "Другое",
    message: "Сообщение",
    send: "ОТПРАВИТЬ",
    music: "Музыка",
    effects: "Эффекты",
    quality: "Качество",
    qualityHigh: "Высокое",
    qualityMedium: "Среднее",
    qualityLow: "Низкое",
    fullscreen: "Полный экран",
  },
  en: {
    langName: "English",
    loginTitle: "LOGIN TO GAME",
    registerTitle: "REGISTRATION",
    forgotTitle: "RESET PASSWORD",
    successTitle: "AUTHORIZATION",
    login: "Login",
    password: "Password",
    passwordRepeat: "Repeat password",
    email: "Email",
    remember: "Remember me",
    forgot: "Forgot password?",
    enter: "LOG IN",
    or: "OR",
    register: "REGISTRATION",
    backToLogin: "LOG IN",
    createAccount: "CREATE ACCOUNT",
    sendLink: "SEND",
    forgotHint: "Enter your email and we will send a reset link.",
    welcome: "Welcome",
    serverReady: "The server is ready to connect.",
    enterWorld: "ENTER WORLD",
    logout: "LOG OUT",
    site: "SITE",
    forum: "FORUM",
    support: "SUPPORT",
    settings: "SETTINGS",
    errLogin: "Enter a login (at least 3 characters).",
    errPassword: "Enter a password (at least 4 characters).",
    errEmail: "Enter a valid email.",
    errMatch: "Passwords do not match.",
    accountCreated: "Account created. You can log in.",
    mailSent: "Password reset email has been sent.",
    connecting: "Connecting to the world of Aden...",
    ticketSent: "Your ticket was sent to support.",
    settingsSaved: "Settings saved.",
    save: "SAVE",
    siteTitle: "Official site",
    forumTitle: "Forum",
    supportTitle: "Support",
    settingsTitle: "Settings",
    news1Title: "Siege of Giran Castle",
    news1Text: "Clan registration is open until Sunday. Reward: treasury and tax control.",
    news2Title: "Raid: Queen Ant",
    news2Text: "The boss spawns every 24 hours. Recommended party level: 40+.",
    topic1: "Interlude newbie guide",
    topic2: "Phoenix clan recruitment",
    topic3: "Bug report: trade window",
    supportText: "Describe a login, character, or donation issue — we reply within a day.",
    category: "Category",
    catLogin: "Login",
    catChar: "Character",
    catOther: "Other",
    message: "Message",
    send: "SEND",
    music: "Music",
    effects: "Effects",
    quality: "Quality",
    qualityHigh: "High",
    qualityMedium: "Medium",
    qualityLow: "Low",
    fullscreen: "Fullscreen",
  },
};

const STORAGE = {
  lang: "l2-lang",
  remember: "l2-remember",
  login: "l2-login",
  settings: "l2-settings",
};

const panel = document.getElementById("auth-panel");
const panelTitle = document.getElementById("panel-title");
const loginForm = document.getElementById("login-form");
const registerForm = document.getElementById("register-form");
const forgotForm = document.getElementById("forgot-form");
const successScreen = document.getElementById("success-screen");
const orDivider = document.getElementById("or-divider");
const switchMode = document.getElementById("switch-mode");
const langBtn = document.getElementById("lang-btn");
const langMenu = document.getElementById("lang-menu");
const overlay = document.getElementById("overlay");
const sheetTitle = document.getElementById("sheet-title");
const sheetBody = document.getElementById("sheet-body");
const toastEl = document.getElementById("toast");
const passwordInput = document.getElementById("password-input");
const togglePassword = document.getElementById("toggle-password");
const remember = document.getElementById("remember");
const loginInput = document.getElementById("login-input");

const state = {
  lang: localStorage.getItem(STORAGE.lang) || "ru",
  screen: "login",
  toastTimer: 0,
};

function t(key) {
  return I18N[state.lang][key] || key;
}

function applyI18n() {
  document.documentElement.lang = state.lang;
  document.querySelectorAll("[data-i18n]").forEach((el) => {
    el.textContent = t(el.dataset.i18n);
  });
  document.querySelectorAll("[data-i18n-placeholder]").forEach((el) => {
    el.placeholder = t(el.dataset.i18nPlaceholder);
  });
  updateChrome();
}

function updateChrome() {
  const titles = {
    login: "loginTitle",
    register: "registerTitle",
    forgot: "forgotTitle",
    success: "successTitle",
  };
  panelTitle.textContent = t(titles[state.screen]);
  const showSwitch = state.screen !== "success";
  orDivider.hidden = !showSwitch;
  switchMode.hidden = !showSwitch;
  switchMode.dataset.i18n = state.screen === "login" ? "register" : "backToLogin";
  switchMode.textContent = t(switchMode.dataset.i18n);
}

function showError(id, message) {
  const el = document.getElementById(id);
  el.hidden = !message;
  el.textContent = message || "";
  if (message) {
    panel.classList.remove("is-shake");
    void panel.offsetWidth;
    panel.classList.add("is-shake");
  }
}

function setScreen(name) {
  state.screen = name;
  [loginForm, registerForm, forgotForm, successScreen].forEach((node) => {
    node.hidden = node.dataset.screen !== name;
  });
  showError("login-error", "");
  showError("register-error", "");
  showError("forgot-error", "");
  updateChrome();
}

function toast(message) {
  toastEl.hidden = false;
  toastEl.textContent = message;
  clearTimeout(state.toastTimer);
  state.toastTimer = window.setTimeout(() => {
    toastEl.hidden = true;
  }, 2600);
}

function isEmail(value) {
  return /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(value);
}

loginForm.addEventListener("submit", (event) => {
  event.preventDefault();
  const login = loginInput.value.trim();
  const password = passwordInput.value;
  if (login.length < 3) {
    showError("login-error", t("errLogin"));
    return;
  }
  if (password.length < 4) {
    showError("login-error", t("errPassword"));
    return;
  }
  if (remember.checked) {
    localStorage.setItem(STORAGE.remember, "1");
    localStorage.setItem(STORAGE.login, login);
  } else {
    localStorage.removeItem(STORAGE.remember);
    localStorage.removeItem(STORAGE.login);
  }
  const submit = loginForm.querySelector("[type=submit]");
  submit.disabled = true;
  window.setTimeout(() => {
    submit.disabled = false;
    document.getElementById("welcome-name").textContent = login;
    setScreen("success");
  }, 450);
});

registerForm.addEventListener("submit", (event) => {
  event.preventDefault();
  const data = new FormData(registerForm);
  const login = String(data.get("login") || "").trim();
  const email = String(data.get("email") || "").trim();
  const password = String(data.get("password") || "");
  const repeat = String(data.get("passwordRepeat") || "");
  if (login.length < 3) return showError("register-error", t("errLogin"));
  if (!isEmail(email)) return showError("register-error", t("errEmail"));
  if (password.length < 4) return showError("register-error", t("errPassword"));
  if (password !== repeat) return showError("register-error", t("errMatch"));
  loginInput.value = login;
  toast(t("accountCreated"));
  setScreen("login");
});

forgotForm.addEventListener("submit", (event) => {
  event.preventDefault();
  const email = String(new FormData(forgotForm).get("email") || "").trim();
  if (!isEmail(email)) return showError("forgot-error", t("errEmail"));
  toast(t("mailSent"));
  setScreen("login");
});

document.getElementById("forgot-link").addEventListener("click", () => setScreen("forgot"));
switchMode.addEventListener("click", () => {
  setScreen(state.screen === "login" ? "register" : "login");
});
document.getElementById("logout-btn").addEventListener("click", () => setScreen("login"));
document.getElementById("enter-world").addEventListener("click", () => toast(t("connecting")));

togglePassword.addEventListener("click", () => {
  const hidden = passwordInput.type === "password";
  passwordInput.type = hidden ? "text" : "password";
  togglePassword.querySelector(".icon-eye").hidden = hidden;
  togglePassword.querySelector(".icon-eye-off").hidden = !hidden;
  togglePassword.setAttribute("aria-label", hidden ? "Скрыть пароль" : "Показать пароль");
});

langBtn.addEventListener("click", () => {
  const open = langMenu.hidden;
  langMenu.hidden = !open;
  langBtn.setAttribute("aria-expanded", String(open));
});

langMenu.addEventListener("click", (event) => {
  const btn = event.target.closest("[data-lang]");
  if (!btn) return;
  state.lang = btn.dataset.lang;
  localStorage.setItem(STORAGE.lang, state.lang);
  langMenu.hidden = true;
  langBtn.setAttribute("aria-expanded", "false");
  applyI18n();
  if (!overlay.hidden) renderSheet(overlay.dataset.view);
});

document.addEventListener("click", (event) => {
  if (!event.target.closest("#lang-wrap")) {
    langMenu.hidden = true;
    langBtn.setAttribute("aria-expanded", "false");
  }
});

function closeOverlay() {
  overlay.hidden = true;
  overlay.dataset.view = "";
}

function renderSheet(view) {
  overlay.dataset.view = view;
  const settings = JSON.parse(localStorage.getItem(STORAGE.settings) || "{}");
  const content = {
    site: {
      title: t("siteTitle"),
      html: `
        <article class="news-card"><h3>${t("news1Title")}</h3><p>${t("news1Text")}</p></article>
        <article class="news-card"><h3>${t("news2Title")}</h3><p>${t("news2Text")}</p></article>
      `,
    },
    forum: {
      title: t("forumTitle"),
      html: `
        <article class="topic"><h3>${t("topic1")}</h3><p>12 ${state.lang === "ru" ? "ответов" : "replies"}</p></article>
        <article class="topic"><h3>${t("topic2")}</h3><p>Aden / Giran</p></article>
        <article class="topic"><h3>${t("topic3")}</h3><p>Interlude</p></article>
      `,
    },
    support: {
      title: t("supportTitle"),
      html: `
        <p>${t("supportText")}</p>
        <form class="support-form" id="support-form">
          <label>${t("category")}
            <select name="category">
              <option>${t("catLogin")}</option>
              <option>${t("catChar")}</option>
              <option>${t("catOther")}</option>
            </select>
          </label>
          <label>${t("email")}
            <input name="email" type="email" required />
          </label>
          <label>${t("message")}
            <textarea name="message" required></textarea>
          </label>
          <button class="btn btn-gold" type="submit">${t("send")}</button>
        </form>
      `,
    },
    settings: {
      title: t("settingsTitle"),
      html: `
        <div class="settings-row"><span>${t("music")}</span><input id="set-music" type="range" min="0" max="100" value="${settings.music ?? 70}" /></div>
        <div class="settings-row"><span>${t("effects")}</span><input id="set-fx" type="range" min="0" max="100" value="${settings.fx ?? 80}" /></div>
        <div class="settings-row"><span>${t("quality")}</span>
          <select id="set-quality">
            <option value="high" ${settings.quality === "high" || !settings.quality ? "selected" : ""}>${t("qualityHigh")}</option>
            <option value="medium" ${settings.quality === "medium" ? "selected" : ""}>${t("qualityMedium")}</option>
            <option value="low" ${settings.quality === "low" ? "selected" : ""}>${t("qualityLow")}</option>
          </select>
        </div>
        <label class="check settings-row">
          <input id="set-fullscreen" type="checkbox" ${settings.fullscreen ? "checked" : ""} />
          <span class="box"></span>
          <span>${t("fullscreen")}</span>
        </label>
        <button class="btn btn-gold" id="save-settings" type="button">${t("save")}</button>
      `,
    },
  };
  sheetTitle.textContent = content[view].title;
  sheetBody.innerHTML = content[view].html;
  overlay.hidden = false;

  const supportForm = document.getElementById("support-form");
  if (supportForm) {
    supportForm.addEventListener("submit", (event) => {
      event.preventDefault();
      closeOverlay();
      toast(t("ticketSent"));
    });
  }

  const saveSettings = document.getElementById("save-settings");
  if (saveSettings) {
    saveSettings.addEventListener("click", async () => {
      const next = {
        music: Number(document.getElementById("set-music").value),
        fx: Number(document.getElementById("set-fx").value),
        quality: document.getElementById("set-quality").value,
        fullscreen: document.getElementById("set-fullscreen").checked,
      };
      localStorage.setItem(STORAGE.settings, JSON.stringify(next));
      if (next.fullscreen && document.documentElement.requestFullscreen) {
        try {
          await document.documentElement.requestFullscreen();
        } catch {
          /* user gesture / permission */
        }
      } else if (!next.fullscreen && document.fullscreenElement) {
        await document.exitFullscreen();
      }
      closeOverlay();
      toast(t("settingsSaved"));
    });
  }
}

document.querySelectorAll(".footer [data-open]").forEach((btn) => {
  btn.addEventListener("click", () => renderSheet(btn.dataset.open));
});
document.getElementById("sheet-close").addEventListener("click", closeOverlay);
overlay.addEventListener("click", (event) => {
  if (event.target === overlay) closeOverlay();
});
document.addEventListener("keydown", (event) => {
  if (event.key === "Escape") {
    closeOverlay();
    langMenu.hidden = true;
  }
});

if (localStorage.getItem(STORAGE.remember) === "1") {
  remember.checked = true;
  loginInput.value = localStorage.getItem(STORAGE.login) || "";
}

applyI18n();
setScreen("login");
