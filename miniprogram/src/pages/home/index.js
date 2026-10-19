// SPDX-License-Identifier: MIT; Copyright (c) 2026 Phoobobo
const api = require("../../lib/api"),
  ui = require("../../lib/ui");
Page({
  data: {
    t: ui.labels(),
    q: "",
    mode: false,
    items: [],
    total: 0,
    page: 0,
    busy: false,
    error: "",
  },
  onLoad(q) {
    if (q.q) this.setData({ q: q.q });
  },
  onShow() {
    if (ui.auth()) {
      this.setData({ t: ui.labels() });
      this.load(true);
    }
  },
  input(e) {
    this.setData({ q: e.detail.value });
  },
  search() {
    this.load(true);
  },
  mode(e) {
    if (this.data.busy) return;
    this.setData({ mode: e.currentTarget.dataset.location === true });
    this.load(true);
  },
  load(reset) {
    ui.run(this, async () => {
      const page = reset ? 1 : this.data.page + 1;
      const data = await api.unwrap(
        api.items.getAll({
          q: this.data.q,
          page,
          pageSize: 30,
          isLocation: this.data.mode,
        }),
      );
      const rows = data.items.map((i) => ({ ...i, photo: api.photo(i) }));
      this.setData({
        items: reset ? rows : this.data.items.concat(rows),
        total: data.total,
        page,
      });
    });
  },
  more() {
    this.load(false);
  },
  detail(e) {
    ui.detail(e.currentTarget.dataset.id);
  },
  add() {
    wx.navigateTo({ url: "/pages/edit/index?location=" + this.data.mode });
  },
  assistant() {
    wx.navigateTo({ url: "/pages/assistant/index" });
  },
  account() {
    wx.navigateTo({ url: "/pages/account/index" });
  },
});
