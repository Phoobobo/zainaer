// SPDX-License-Identifier: MIT; Copyright (c) 2026 Phoobobo
// No DOM, Vue, Nuxt, fetch, or wx dependencies: both clients use this module.
import type {
  EntityCreate,
  EntityListResult,
  EntityOut,
  EntityPath,
  EntityUpdate,
  EntityTypeSummary,
} from "../frontend/lib/api/types/data-contracts";

export type QueryValue =
  | string
  | string[]
  | number
  | number[]
  | boolean
  | null
  | undefined;
export type ItemsQuery = {
  orderBy?: string;
  includeArchived?: boolean;
  page?: number;
  pageSize?: number;
  parentIds?: string[];
  tags?: string[];
  negateTags?: boolean;
  onlyWithoutPhoto?: boolean;
  onlyWithPhoto?: boolean;
  q?: string;
  fields?: string[];
  isLocation?: boolean;
};
export function queryString(params: Record<string, QueryValue> = {}) {
  const pairs: string[] = [];
  for (const [key, value] of Object.entries(params)) {
    if (value === undefined || value === null) continue;
    for (const item of Array.isArray(value) ? value : [value])
      pairs.push(
        `${encodeURIComponent(key)}=${encodeURIComponent(String(item))}`,
      );
  }
  return pairs.length ? "?" + pairs.join("&") : "";
}
export function householdEntityTypes(types: EntityTypeSummary[]) {
  const sorted = [...types].sort(
    (a, b) => new Date(a.createdAt).getTime() - new Date(b.createdAt).getTime(),
  );
  return [false, true].flatMap((isLocation) => {
    const type = sorted.find((t) => t.isLocation === isLocation);
    return type ? [type] : [];
  });
}
export interface APIResponse<T, R = unknown> {
  response: R;
  status: number;
  error: boolean;
  data: T;
}
export interface Transport<R = unknown> {
  get<T>(args: { url: string }): Promise<APIResponse<T, R>>;
  post<T, U>(args: { url: string; body?: T }): Promise<APIResponse<U, R>>;
  put<T, U>(args: { url: string; body?: T }): Promise<APIResponse<U, R>>;
  delete<T>(args: { url: string }): Promise<APIResponse<T, R>>;
}
export class HouseholdItems<R = unknown> {
  constructor(
    private http: Transport<R>,
    private route: (path: string, query?: Record<string, QueryValue>) => string,
  ) {}
  getAll(query: ItemsQuery = {}) {
    return this.http.get<EntityListResult>({
      url: this.route("/entities", query),
    });
  }
  get(id: string) {
    return this.http.get<EntityOut>({
      url: this.route(`/entities/${encodeURIComponent(id)}`),
    });
  }
  fullpath(id: string) {
    return this.http.get<EntityPath[]>({
      url: this.route(`/entities/${encodeURIComponent(id)}/path`),
    });
  }
  create(body: EntityCreate) {
    return this.http.post<EntityCreate, EntityOut>({
      url: this.route("/entities"),
      body,
    });
  }
  update(id: string, body: EntityUpdate) {
    return this.http.put<EntityUpdate, EntityOut>({
      url: this.route(`/entities/${encodeURIComponent(id)}`),
      body,
    });
  }
  delete(id: string) {
    return this.http.delete<void>({
      url: this.route(`/entities/${encodeURIComponent(id)}`),
    });
  }
}
