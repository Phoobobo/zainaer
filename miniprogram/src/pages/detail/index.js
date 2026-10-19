// SPDX-License-Identifier: MIT; Copyright (c) 2026 Phoobobo
const api = require("../../lib/api"),
  ui = require("../../lib/ui");
Page({
  data: {
    t: ui.labels(),
    item: null,
    path: "",
    photo: "",
    busy: false,
    error: "",
  },
  onLoad(q) {
    this.id = q.id;
  },
  onShow() {
    if (ui.auth()) this.load();
  },
  load() {
    ui.run(this, async () => {
      const [item, path] = await Promise.all([
        api.unwrap(api.items.get(this.id)),
        api.unwrap(api.items.fullpath(this.id)),
      ]);
      this.setData({
        t: ui.labels(),
        item,
        path: path.map((x) => x.name).join(" › "),
        photo: api.photo(item),
      });
    });
  },
  edit() {
    wx.navigateTo({
      url: "/pages/edit/index?id=" + encodeURIComponent(this.id),
    });
  },
  child(e) {
    ui.detail(e.currentTarget.dataset.id);
  },
  remove() {
    wx.showModal({
      title: this.data.t.remove,
      content: this.data.t.deleteQuestion,
      success: (r) => {
        if (r.confirm)
          ui.run(this, async () => {
            await api.unwrap(api.items.delete(this.id));
            wx.navigateBack();
          });
      },
    });
  },
});
