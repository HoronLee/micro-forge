type RequestType = {
  path: string;
  method: string;
  body: string | null;
};

type RequestMeta = {
  service: string;
  method: string;
};

export type RequestHandler = (
  request: RequestType,
  meta: RequestMeta,
) => Promise<unknown>;

export type RequestHandlerOptions = {
  baseUrl?: string;
  getAccessToken?: () => string | null | undefined;
};

function normalizeBaseUrl(baseUrl: string): string {
  const normalized = baseUrl.trim().replace(/\/+$/, "");
  if (normalized === "" && baseUrl.startsWith("/")) {
    return "/";
  }
  return normalized;
}

function buildUrl(baseUrl: string, path: string): string {
  const safePath = path.replace(/^\/+/, "");
  const normalizedBase = normalizeBaseUrl(baseUrl);
  if (normalizedBase === "" || normalizedBase === "/") {
    return `/${safePath}`;
  }
  return `${normalizedBase}/${safePath}`;
}

function getDefaultBaseUrl(): string {
  return import.meta.env.VITE_API_BASE_URL || "/api";
}

export function createRequestHandler(
  options: RequestHandlerOptions = {},
): RequestHandler {
  const baseUrl = options.baseUrl || getDefaultBaseUrl();

  return async (request, meta) => {
    const token = options.getAccessToken?.();
    const headers = new Headers({ Accept: "application/json" });

    if (request.body != null) {
      headers.set("Content-Type", "application/json");
    }
    if (token) {
      headers.set("Authorization", `Bearer ${token}`);
    }

    const response = await fetch(buildUrl(baseUrl, request.path), {
      method: request.method,
      headers,
      body: request.body,
    });

    const contentType = response.headers.get("content-type") || "";
    const data = contentType.includes("application/json")
      ? await response.json()
      : await response.text();

    if (!response.ok) {
      const message =
        typeof data === "object" &&
        data !== null &&
        "message" in data &&
        typeof (data as { message?: unknown }).message === "string"
          ? (data as { message: string }).message
          : `${meta.service}.${meta.method} 请求失败 (${response.status})`;
      throw new Error(message);
    }

    return data;
  };
}
