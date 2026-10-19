/** RFC 9457 problem details with rovo extensions (doc 11 API-D05). */
export interface FieldError {
  field: string;
  code: string;
  message?: string;
}

export interface ProblemDetails {
  type?: string;
  title?: string;
  status?: number;
  detail?: string;
  instance?: string;
  code?: string;
  traceId?: string;
  fieldErrors?: FieldError[];
  [extension: string]: unknown;
}

const STATUS_CODES: Record<number, string> = {
  400: 'VALIDATION_FAILED',
  401: 'UNAUTHENTICATED',
  403: 'FORBIDDEN',
  404: 'NOT_FOUND',
  408: 'TIMEOUT',
  409: 'CONFLICT',
  412: 'VERSION_MISMATCH',
  422: 'VALIDATION_FAILED',
  426: 'CLIENT_UPGRADE_REQUIRED',
  429: 'RATE_LIMITED',
  503: 'SERVICE_UNAVAILABLE',
};

function fallbackCode(status: number): string {
  if (status === 0) return 'NETWORK';
  return STATUS_CODES[status] ?? (status >= 500 ? 'INTERNAL' : 'UNKNOWN');
}

function isRecord(v: unknown): v is Record<string, unknown> {
  return typeof v === 'object' && v !== null && !Array.isArray(v);
}

/**
 * Every failed API call surfaces as an `ApiError`. UI code switches on the stable `code`
 * (translated via `errors:<code>`), never on `title`/`detail` text.
 */
export class ApiError extends Error {
  override name = 'ApiError';
  readonly status: number;
  readonly code: string;
  readonly title?: string;
  readonly detail?: string;
  readonly type?: string;
  readonly traceId?: string;
  readonly fieldErrors: FieldError[];
  readonly retryAfterSeconds?: number;
  /** The full problem body (e.g. `quote` and `diff` on 409 QUOTE_CHANGED). */
  readonly problem: ProblemDetails;

  constructor(status: number, problem: ProblemDetails = {}, init: { retryAfterSeconds?: number; cause?: unknown } = {}) {
    const code = typeof problem.code === 'string' && problem.code ? problem.code : fallbackCode(status);
    super(problem.detail ?? problem.title ?? code, { cause: init.cause });
    this.status = status;
    this.code = code;
    this.title = problem.title;
    this.detail = problem.detail;
    this.type = problem.type;
    this.traceId = problem.traceId;
    this.fieldErrors = Array.isArray(problem.fieldErrors) ? problem.fieldErrors : [];
    this.retryAfterSeconds = init.retryAfterSeconds;
    this.problem = problem;
  }

  get isNetworkError(): boolean {
    return this.status === 0;
  }

  /** Builds an error from an HTTP response and its (possibly already parsed) body. */
  static fromResponse(response: Response, body?: unknown): ApiError {
    const retryAfter = response.headers.get('Retry-After');
    const retryAfterSeconds = retryAfter && /^\d+$/.test(retryAfter) ? Number(retryAfter) : undefined;
    let problem: ProblemDetails = {};
    if (isRecord(body)) problem = body as ProblemDetails;
    else if (typeof body === 'string' && body) {
      try {
        const parsed: unknown = JSON.parse(body);
        if (isRecord(parsed)) problem = parsed as ProblemDetails;
      } catch {
        problem = { detail: undefined };
      }
    }
    return new ApiError(response.status, problem, { retryAfterSeconds });
  }

  /** Reads and parses a problem+json (or any) error response. */
  static async read(response: Response): Promise<ApiError> {
    let body: unknown;
    try {
      const text = await response.text();
      body = text;
    } catch {
      body = undefined;
    }
    return ApiError.fromResponse(response, body);
  }

  static network(cause: unknown): ApiError {
    return new ApiError(0, { code: 'NETWORK', title: 'Network error' }, { cause });
  }
}

export function isApiError(e: unknown): e is ApiError {
  return e instanceof ApiError;
}

/** Stable error code of any thrown value (`UNKNOWN` for non-API errors). */
export function errorCode(e: unknown): string {
  if (isApiError(e)) return e.code;
  if (e instanceof TypeError) return 'NETWORK';
  return 'UNKNOWN';
}
