import { register } from "../runtime.js";

register("tabs", {
  connect() {
    this.el.addEventListener("click", (e) => {
      const btn = e.target.closest("[role='tab']");
      if (!btn || !this.el.contains(btn)) return;
      this.select(btn.getAttribute("data-tab"));
    });
  },
  select(value) {
    if (typeof value !== "string") {
      value = value.target?.closest("[data-tab]")?.getAttribute("data-tab");
    }
    if (!value) return;
    this.el.querySelectorAll("[role='tab']").forEach((btn) => {
      const active = btn.getAttribute("data-tab") === value;
      btn.setAttribute("aria-selected", String(active));
      btn.setAttribute("tabindex", active ? "0" : "-1");
      btn.classList.toggle("bg-background", active);
      btn.classList.toggle("text-foreground", active);
      btn.classList.toggle("shadow", active);
    });
    this.el.querySelectorAll("[role='tabpanel']").forEach((panel) => {
      panel.hidden = panel.getAttribute("data-tab") !== value;
    });
  },
});
