import { register } from "../runtime.js";

register("dialog", {
  connect() {
    // Open: scoped to this element — only triggers within this controller's tree.
    this.el.addEventListener("click", (e) => {
      const opener = e.target.closest("[data-dialog-open]");
      if (opener && this.el.contains(opener)) {
        e.stopPropagation();
        const id = opener.getAttribute("data-dialog-open");
        document.getElementById(id)?.showModal();
      }
    });
    // Close: must listen at document level because <dialog> in the top-layer
    // does not bubble click events through its DOM ancestors.
    this._docClose = (e) => {
      const closer = e.target.closest("[data-dialog-close]");
      if (closer) closer.closest("dialog")?.close();
    };
    document.addEventListener("click", this._docClose);
  },
  disconnect() {
    document.removeEventListener("click", this._docClose);
  },
});
