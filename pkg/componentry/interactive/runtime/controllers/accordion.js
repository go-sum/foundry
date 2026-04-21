import { register } from "../runtime.js";

register("accordion", {
  connect() {
    if (!this.el.hasAttribute("data-accordion-exclusive")) return;
    this.el.addEventListener("toggle", (e) => {
      if (!e.target.open) return;
      this.el.querySelectorAll("details").forEach((d) => {
        if (d !== e.target) d.open = false;
      });
    }, true);
  },
});
