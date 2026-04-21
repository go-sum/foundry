const registry = {};
const connected = new WeakMap();

export function register(name, definition) {
  registry[name] = definition;
}

function getTargets(el, name) {
  const map = {};
  el.querySelectorAll(`[data-${name}-target]`).forEach((t) => {
    t.getAttribute(`data-${name}-target`).split(/\s+/).forEach((key) => {
      (map[key] ||= []).push(t);
    });
  });
  return map;
}

function parseActions(el) {
  const raw = el.getAttribute("data-action");
  if (!raw) return [];
  return raw.split(/\s+/).map((s) => {
    const [trigger, handler] = s.split("->");
    const [ctrl, method] = handler.split("#");
    return { event: trigger, ctrl, method };
  });
}

function connectController(el, name) {
  const def = registry[name];
  if (!def) return;
  let instances = connected.get(el);
  if (instances?.has(name)) return;
  if (!instances) {
    instances = new Map();
    connected.set(el, instances);
  }
  const instance = Object.create(def);
  instance.el = el;
  instance.targets = getTargets(el, name);
  instance.target = (k) => instance.targets[k]?.[0];
  instance.targetsAll = (k) => instance.targets[k] || [];
  instances.set(name, instance);
  instance.connect?.();
}

function disconnectController(el, name) {
  const instances = connected.get(el);
  if (!instances) return;
  const instance = instances.get(name);
  if (!instance) return;
  instance.disconnect?.();
  instances.delete(name);
}

function connectAll(root) {
  if (!root) return;
  const selector = "[data-controller]";
  const els = root.matches?.(selector) ? [root, ...root.querySelectorAll(selector)] : root.querySelectorAll(selector);
  els.forEach((el) => {
    el.getAttribute("data-controller").split(/\s+/).forEach((name) => connectController(el, name));
  });
}

function disconnectAll(root) {
  if (!root) return;
  const selector = "[data-controller]";
  const els = root.matches?.(selector) ? [root, ...root.querySelectorAll(selector)] : root.querySelectorAll(selector);
  els.forEach((el) => {
    const instances = connected.get(el);
    if (!instances) return;
    instances.forEach((_, name) => disconnectController(el, name));
  });
}

document.addEventListener("click", (e) => {
  const target = e.target.closest("[data-action]");
  if (!target) return;
  parseActions(target).forEach(({ event, ctrl, method }) => {
    if (event !== "click") return;
    const scope = target.closest(`[data-controller~="${ctrl}"]`);
    if (!scope) return;
    const instances = connected.get(scope);
    const instance = instances?.get(ctrl);
    instance?.[method]?.(e);
  });
});

// [data-dismiss]: remove the closest [data-dismissible] ancestor.
document.addEventListener("click", (e) => {
  const btn = e.target.closest("[data-dismiss]");
  if (!btn) return;
  btn.closest("[data-dismissible]")?.remove();
});

// data-toast-trigger: clone a <template> by ID and append to #toast-container.
document.addEventListener("click", (e) => {
  const btn = e.target.closest("[data-toast-trigger]");
  if (!btn) return;
  const tmpl = document.getElementById(btn.getAttribute("data-toast-trigger"));
  if (!tmpl) return;
  const container = document.getElementById("toast-container") ?? document.body;
  container.appendChild(tmpl.content.cloneNode(true));
});

document.addEventListener("DOMContentLoaded", () => connectAll(document.body));

document.addEventListener("htmx:afterSettle", (e) => connectAll(e.detail.elt));
document.addEventListener("htmx:beforeCleanupElement", (e) => disconnectAll(e.detail.elt));

new MutationObserver((mutations) => {
  mutations.forEach((m) => {
    m.addedNodes.forEach((n) => { if (n.nodeType === 1) connectAll(n); });
    m.removedNodes.forEach((n) => { if (n.nodeType === 1) disconnectAll(n); });
  });
}).observe(document.body, { childList: true, subtree: true });
