import type { components, paths } from './schema';

/** JSON body of a 2xx response for `path` + `method`, derived from the OpenAPI contract. */
type Json<R> = R extends { content: { 'application/json': infer B } } ? B : never;
type Responses<P extends keyof paths, M extends keyof paths[P]> = paths[P][M] extends {
  responses: infer R;
}
  ? R
  : never;
export type SuccessBody<P extends keyof paths, M extends keyof paths[P]> = {
  [S in keyof Responses<P, M>]: S extends 200 | 201 | 202 ? Json<Responses<P, M>[S]> : never;
}[keyof Responses<P, M>];

export type RequestBody<P extends keyof paths, M extends keyof paths[P]> = paths[P][M] extends {
  requestBody?: { content: { 'application/json': infer B } };
}
  ? B
  : never;

export type Me = SuccessBody<'/me', 'get'>;
export type RoleName = NonNullable<components['schemas']['Role']>;
export type OtpChallenge = SuccessBody<'/auth/otp/request', 'post'>;
export type OtpVerifyResult = SuccessBody<'/auth/otp/verify', 'post'>;
export type AdminMfaChallenge = SuccessBody<'/auth/admin/login', 'post'>;
