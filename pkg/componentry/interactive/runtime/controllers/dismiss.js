import { register } from "../runtime.js";

register("dismiss", {
  remove() {
    this.el.closest("[data-dismissible]")?.remove();
  },
});
