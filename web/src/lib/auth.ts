const STORAGE_KEY = "octo_admin_key";

export function getAdminKey(): string | null {
  return localStorage.getItem(STORAGE_KEY);
}

export function setAdminKey(key: string): void {
  localStorage.setItem(STORAGE_KEY, key);
}

export function clearAdminKey(): void {
  localStorage.removeItem(STORAGE_KEY);
}

export function isAuthenticated(): boolean {
  return !!getAdminKey();
}
