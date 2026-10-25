// SPDX-License-Identifier: MIT; Copyright (c) 2026 Phoobobo
// Keep binary conversion here so registration and login share the same browser adapter.
type DescriptorJSON = Omit<PublicKeyCredentialDescriptor, "id"> & { id: string };
export type CreationJSON = Omit<PublicKeyCredentialCreationOptions, "challenge" | "user" | "excludeCredentials"> & {
  challenge: string;
  user: Omit<PublicKeyCredentialUserEntity, "id"> & { id: string };
  excludeCredentials?: DescriptorJSON[];
};
export type RequestJSON = Omit<PublicKeyCredentialRequestOptions, "challenge" | "allowCredentials"> & {
  challenge: string;
  allowCredentials?: DescriptorJSON[];
};
export interface PasskeySummary {
  id: string;
  name: string;
  createdAt: string;
  lastUsedAt: string | null;
}
export function passkeysSupported() {
  return (
    typeof window !== "undefined" && window.isSecureContext && !!window.PublicKeyCredential && !!navigator.credentials
  );
}
function decode(value: string): ArrayBuffer {
  const raw = atob(value.replace(/-/g, "+").replace(/_/g, "/"));
  return Uint8Array.from(raw, c => c.charCodeAt(0)).buffer;
}
function encode(value: ArrayBuffer): string {
  return btoa(String.fromCharCode(...new Uint8Array(value)))
    .replace(/\+/g, "-")
    .replace(/\//g, "_")
    .replace(/=+$/, "");
}
function serialize(credential: Credential | null) {
  if (!credential) throw new DOMException("Cancelled", "NotAllowedError");
  const key = credential as PublicKeyCredential;
  const response = key.response;
  const common = {
    id: key.id,
    rawId: encode(key.rawId),
    type: key.type,
    clientExtensionResults: key.getClientExtensionResults(),
    authenticatorAttachment: key.authenticatorAttachment,
  };
  if ("attestationObject" in response) {
    const attestation = response as AuthenticatorAttestationResponse;
    return {
      ...common,
      response: {
        clientDataJSON: encode(response.clientDataJSON),
        attestationObject: encode(attestation.attestationObject),
        transports: attestation.getTransports?.() ?? [],
      },
    };
  }
  const assertion = response as AuthenticatorAssertionResponse;
  return {
    ...common,
    response: {
      clientDataJSON: encode(response.clientDataJSON),
      authenticatorData: encode(assertion.authenticatorData),
      signature: encode(assertion.signature),
      userHandle: assertion.userHandle ? encode(assertion.userHandle) : null,
    },
  };
}
export async function createPasskey(options: CreationJSON) {
  return serialize(
    await navigator.credentials.create({
      publicKey: {
        ...options,
        challenge: decode(options.challenge),
        user: { ...options.user, id: decode(options.user.id) },
        excludeCredentials: options.excludeCredentials?.map(c => ({ ...c, id: decode(c.id) })),
      },
    })
  );
}
export async function getPasskey(options: RequestJSON) {
  return serialize(
    await navigator.credentials.get({
      publicKey: {
        ...options,
        challenge: decode(options.challenge),
        allowCredentials: options.allowCredentials?.map(c => ({ ...c, id: decode(c.id) })),
      },
    })
  );
}
export function passkeyErrorKey(error: unknown) {
  if (error instanceof DOMException && error.name === "NotAllowedError") return "passkeys.cancelled";
  if (error instanceof DOMException && error.name === "InvalidStateError") return "passkeys.already_added";
  return "passkeys.failed";
}
