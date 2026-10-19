// SPDX-License-Identifier: MIT; Copyright (c) 2026 Phoobobo
const api = require("../../lib/api"),
  ui = require("../../lib/ui");
Page({
  data: {
    t: ui.labels(),
    messages: [],
    input: "",
    configured: false,
    busy: false,
    error: "",
  },
  onLoad() {
    this.summary = "";
    this.historyStart = 0;
    this.closed = false;
    if (ui.auth()) this.load();
  },
  onUnload() {
    this.closed = true;
  },
  load() {
    ui.run(this, async () => {
      const status = await api.call("GET", "/ai/status");
      if (!this.closed)
        this.setData({
          configured: status.configured,
          error: status.configured ? "" : this.data.t.noModel,
        });
    });
  },
  input(e) {
    this.setData({ input: e.detail.value });
  },
  send() {
    ui.run(this, async () => {
      const content = this.data.input.trim();
      if (!content || !this.data.configured) return;
      const messages = this.data.messages.map((m) => ({
        ...m,
        drafts: (m.drafts || []).map((d) => ({
          ...d,
          state: d.state || "replaced",
        })),
      }));
      messages.push({ role: "user", content });
      this.setData({ messages, input: "" });
      const history = messages
        .slice(this.historyStart)
        .map((m) => ({
          role: m.role,
          content:
            m.content +
            (m.drafts || [])
              .map(
                (d) =>
                  "\n" +
                  d.proposal.name +
                  "; " +
                  d.parentPath +
                  "; " +
                  (d.state || "awaiting confirmation"),
              )
              .join(""),
        }));
      if (this.summary)
        history.unshift({ role: "assistant", content: this.summary });
      try {
        const result = await api.call("POST", "/ai/chat", {
          messages: history,
        });
        if (this.closed) return;
        if (result.summary) {
          this.summary = result.summary;
          this.historyStart = messages.length - 1;
        }
        messages.push({
          role: "assistant",
          content: result.reply,
          usedModel: result.usedModel,
          items: result.items || [],
          drafts: result.drafts || [],
        });
        this.setData({ messages });
      } catch (e) {
        messages.pop();
        if (!this.closed) this.setData({ messages, input: content });
        throw e;
      }
    });
  },
  confirm(e) {
    ui.run(this, async () => {
      const { message, draft, cancel } = e.currentTarget.dataset;
      const messages = this.data.messages;
      const d = messages[message].drafts[draft];
      if (d.state) return;
      const result = await api.call("POST", "/ai/confirm", {
        draftId: d.id,
        cancel: !!cancel,
      });
      if (this.closed) return;
      d.state = cancel ? "cancelled" : "saved";
      if (!cancel)
        messages.push({
          role: "assistant",
          content: this.data.t.saved + ": " + result.name,
          items: [result],
        });
      this.setData({ messages });
    });
  },
  detail(e) {
    if (e.currentTarget.dataset.kind === "tag") {
      wx.navigateTo({
        url:
          "/pages/home/index?q=" +
          encodeURIComponent(e.currentTarget.dataset.name),
      });
    } else {
      ui.detail(e.currentTarget.dataset.id);
    }
  },
});
