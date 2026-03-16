const STORAGE_KEY = "datasrv.admin.token";

export function getStoredToken() {
  return window.localStorage.getItem(STORAGE_KEY);
}

export function setStoredToken(token: string) {
  window.localStorage.setItem(STORAGE_KEY, token);
}

export function clearStoredToken() {
  window.localStorage.removeItem(STORAGE_KEY);
}
