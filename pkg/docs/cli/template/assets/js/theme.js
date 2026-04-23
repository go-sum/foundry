(function () {
  function preferredTheme() {
    return localStorage.getItem("themePreference") || "system";
  }

  function prefersDark() {
    return window.matchMedia("(prefers-color-scheme: dark)").matches;
  }

  function applyTheme(preference) {
    var html = document.documentElement;
    html.dataset.themePreference = preference;
    html.classList.toggle("dark", preference === "dark" || (preference === "system" && prefersDark()));
  }

  function cycleTheme() {
    var order = ["light", "dark", "system"];
    var current = document.documentElement.dataset.themePreference || "system";
    var next = order[(order.indexOf(current) + 1) % order.length];
    localStorage.setItem("themePreference", next);
    applyTheme(next);
  }

  applyTheme(preferredTheme());

  document.addEventListener("click", function (event) {
    var toggle = event.target.closest("[data-theme-toggle]");
    if (!toggle) {
      return;
    }
    cycleTheme();
  });

  window.matchMedia("(prefers-color-scheme: dark)").addEventListener("change", function () {
    if (preferredTheme() === "system") {
      applyTheme("system");
    }
  });
})();
