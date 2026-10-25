import type { CreationJSON, PasskeySummary } from "~/lib/passkeys";
import { BaseAPI, route } from "../base";
import type { APIKeyCreate, APIKeyCreatedOut, APIKeyOut, ChangePassword, UserOut } from "../types/data-contracts";
import type { Result } from "../types/non-generated";

export class UserApi extends BaseAPI {
  public listPasskeys() {
    return this.http.get<PasskeySummary[]>({ url: route("/users/self/passkeys") });
  }
  public beginPasskeyRegistration(name: string, password: string) {
    return this.http.post<object, { publicKey: CreationJSON }>({
      url: route("/users/self/passkeys/begin"),
      body: { name, password },
    });
  }
  public finishPasskeyRegistration(body: object) {
    return this.http.post<object, { ok: boolean }>({ url: route("/users/self/passkeys/finish"), body });
  }
  public deletePasskey(id: string) {
    return this.http.delete<void>({ url: route(`/users/self/passkeys/${id}`) });
  }

  public completeCollectionNaming(name: string) {
    return this.http.put<{ name: string }, void>({ url: route("/users/self/collection-name"), body: { name } });
  }

  public self() {
    return this.http.get<Result<UserOut>>({ url: route("/users/self") });
  }

  public logout() {
    return this.http.post<object, void>({ url: route("/users/logout") });
  }

  public delete() {
    return this.http.delete<void>({ url: route("/users/self") });
  }

  public changePassword(current: string, newPassword: string) {
    return this.http.put<ChangePassword, void>({
      url: route("/users/self/change-password"),
      body: {
        current,
        new: newPassword,
      },
    });
  }

  public getSettings() {
    return this.http.get<Result<Record<string, unknown>>>({
      url: route("/users/self/settings"),
    });
  }

  public setSettings(settings: Record<string, unknown>) {
    return this.http.put<Record<string, unknown>, Result<Record<string, unknown>>>({
      url: route("/users/self/settings"),
      body: settings,
    });
  }

  public listApiKeys() {
    return this.http.get<APIKeyOut[]>({ url: route("/users/self/api-keys") });
  }

  public createApiKey(body: APIKeyCreate) {
    return this.http.post<APIKeyCreate, APIKeyCreatedOut>({
      url: route("/users/self/api-keys"),
      body,
    });
  }

  public deleteApiKey(id: string) {
    return this.http.delete<void>({ url: route(`/users/self/api-keys/${id}`) });
  }
}
