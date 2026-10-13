// Thin fetch wrapper for the Go API. Errors carry the API's plain-language message, and the
// whole response body for the few refusals that say more (e.g. a 409's `code`).
export class ApiError extends Error {
  constructor(
    public status: number,
    message: string,
    public body: unknown = null,
  ) {
    super(message);
  }
}

export async function api<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(`/api${path}`, {
    ...init,
    headers: { "Content-Type": "application/json", ...init?.headers },
  });
  const body = res.status === 204 ? null : await res.json().catch(() => null);
  if (!res.ok) throw new ApiError(res.status, body?.error ?? "Something went wrong. Please try again.", body);
  return body as T;
}
