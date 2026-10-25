// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Phoobobo
export type VoiceState = "idle" | "checking" | "starting" | "recording" | "transcribing";
type Recognition = {
  processLocally: boolean;
  lang: string;
  continuous: boolean;
  interimResults: boolean;
  onstart: (() => void) | null;
  onend: (() => void) | null;
  onerror: ((event: { error: string }) => void) | null;
  onresult: ((event: { results: ArrayLike<{ isFinal: boolean; 0: { transcript: string } }> }) => void) | null;
  start(): void;
  stop(): void;
  abort(): void;
};
export type LocalRecognition = {
  new (): Recognition;
  available(options: { langs: string[]; processLocally: true }): Promise<string>;
};

// No remote fallback, language-pack installation, audio persistence, or automatic submission.
export function createLocalVoice(options: {
  recognition: () => LocalRecognition | undefined;
  state: (state: VoiceState) => void;
  error: (code: string) => void;
  text: (text: string) => void;
}) {
  let active: Recognition | undefined;
  let token = 0;
  let state: VoiceState = "idle";
  let timer: ReturnType<typeof setTimeout> | undefined;
  const update = (next: VoiceState) => {
    state = next;
    options.state(next);
  };
  function cancel() {
    token++;
    clearTimeout(timer);
    const previous = active;
    active = undefined;
    if (previous) {
      previous.onstart = previous.onend = previous.onerror = previous.onresult = null;
      try {
        previous.abort();
      } catch {
        /* Already ended. */
      }
    }
    update("idle");
  }
  function fail(code: string) {
    cancel();
    options.error(code);
  }
  async function start(lang: string) {
    if (state !== "idle") return;
    const current = ++token;
    options.error("");
    update("checking");
    timer = setTimeout(() => fail("timeout"), 10000);
    try {
      const Constructor = options.recognition();
      if (!Constructor || typeof Constructor.available !== "function") return fail("unsupported");
      const instance = new Constructor();
      if (!("processLocally" in instance)) return fail("unsupported");
      instance.processLocally = true;
      const availability = await Constructor.available({ langs: [lang], processLocally: true });
      if (current !== token) return;
      if (availability !== "available") return fail("language");
      active = instance;
      instance.lang = lang;
      instance.continuous = false;
      instance.interimResults = false;
      let transcript = "";
      instance.onstart = () => {
        if (current === token && state === "starting") update("recording");
      };
      instance.onresult = event => {
        if (current !== token) return;
        transcript = Array.from(event.results)
          .filter(result => result.isFinal)
          .map(result => result[0].transcript)
          .join(" ")
          .trim();
      };
      instance.onerror = event => {
        if (current === token) fail(event.error);
      };
      instance.onend = () => {
        if (current !== token) return;
        cancel();
        if (transcript) options.text(transcript);
        else options.error("no-speech");
      };
      clearTimeout(timer);
      update("starting");
      timer = setTimeout(() => fail("timeout"), 60000);
      instance.start();
    } catch {
      if (current === token) fail("failed");
    }
  }
  function stop() {
    if (!active || state !== "recording") return;
    update("transcribing");
    clearTimeout(timer);
    timer = setTimeout(() => fail("timeout"), 10000);
    try {
      active.stop();
    } catch {
      fail("failed");
    }
  }
  return { start, stop, cancel };
}
