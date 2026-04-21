import { register } from "../runtime.js";

register("toast", {
  connect() {
    const duration = parseInt(this.el.getAttribute("data-toast-duration") || "0", 10);
    if (duration > 0) {
      this._timer = setTimeout(() => this._remove(), duration);
    }
  },
  disconnect() {
    clearTimeout(this._timer);
  },
  dismiss() {
    this._remove();
  },
  _remove() {
    this.el.style.transition = "opacity 150ms ease";
    this.el.style.opacity = "0";
    setTimeout(() => this.el.remove(), 150);
  },
});
