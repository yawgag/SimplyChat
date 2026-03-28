import { elements } from "./dom.js";

export function log(message) {
  if (!elements.logs) {
    return;
  }

  elements.logs.innerHTML += `<div>${message}</div>`;
  elements.logs.scrollTop = elements.logs.scrollHeight;
}
