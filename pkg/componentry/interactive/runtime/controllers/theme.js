import { register } from "../runtime.js";

register("theme", {
  cycle() {
    const themes = ["light", "dark", "system"];
    const cur = localStorage.getItem("themePreference") || "system";
    const next = themes[(themes.indexOf(cur) + 1) % themes.length];
    localStorage.setItem("themePreference", next);
    document.documentElement.setAttribute("data-theme-preference", next);
    const prefersDark = window.matchMedia("(prefers-color-scheme: dark)").matches;
    if (next === "dark" || (next === "system" && prefersDark)) {
      document.documentElement.classList.add("dark");
    } else {
      document.documentElement.classList.remove("dark");
    }
  },
});
