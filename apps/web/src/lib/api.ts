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

/**
 * What every page shows when the server can't be reached (S29; Rahul, 25 Oct): the request never
 * got an answer (offline, or the API down: status 0), or something in between answered a 5xx
 * without the API's `error` (S30): the dev proxy's empty 502, or API Gateway's own JSON
 * (`{"message": …}`) on AWS. Never the browser's own "Failed to fetch".
 */
export const NETWORK_ERROR = "We couldn't reach the server. Check your connection and try again.";

export async function api<T>(path: string, init?: RequestInit): Promise<T> {
  let res: Response;
  try {
    res = await fetch(`/api${path}`, {
      ...init,
      headers: { "Content-Type": "application/json", ...init?.headers },
    });
  } catch (e) {
    if (e instanceof DOMException && e.name === "AbortError") throw e; // the caller cancelled it
    throw new ApiError(0, NETWORK_ERROR);
  }
  const body = res.status === 204 ? null : await res.json().catch(() => null);
  if (!res.ok) {
    const fallback = res.status >= 500 ? NETWORK_ERROR : "Something went wrong. Please try again.";
    throw new ApiError(res.status, body?.error ?? fallback, body);
  }
  return body as T;
}
