import { register } from "../runtime.js";

register("dropdown", {
  connect() {
    this._outsideClick = (e) => {
      if (!this.el.contains(e.target)) this.el.open = false;
    };
    this._keydown = (e) => {
      if (e.key === "Escape") this.el.open = false;
    };
    document.addEventListener("click", this._outsideClick);
    document.addEventListener("keydown", this._keydown);
  },
  disconnect() {
    document.removeEventListener("click", this._outsideClick);
    document.removeEventListener("keydown", this._keydown);
  },
});
