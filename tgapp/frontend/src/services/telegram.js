export const tg = window.Telegram?.WebApp || null;

export function isTelegram() {
  return !!tg?.initData;
}

export function postNativeMessage(name, payload) {
  try {
    const handlers = window.webkit?.messageHandlers;
    const handler = handlers?.[name];
    if (!handler || typeof handler.postMessage !== "function") return false;
    handler.postMessage(payload ?? {});
    return true;
  } catch {
    return false;
  }
}

export function expandTg() {
  if (tg?.expand) tg.expand();
}
