// SPDX-License-Identifier: MIT; Copyright (c) 2026 Phoobobo
const api = require("../../lib/api"),
  ui = require("../../lib/ui");
Page({
  data: {
    t: ui.labels(),
    groups: [],
    names: [],
    index: 0,
    name: "",
    invitation: "",
    naming: false,
    busy: false,
    error: "",
  },
  onShow() {
    if (ui.auth()) this.load();
  },
  load() {
    ui.run(this, async () => {
      const [groups, self] = await Promise.all([
        api.call("GET", "/groups/all"),
        api.call("GET", "/users/self"),
      ]);
      const user = self.item || self;
      this.setData({
        t: ui.labels(),
        groups,
        names: groups.map((g) => g.name),
        index: Math.max(
          0,
          groups.findIndex(
            (g) => g.id === (api.tenant() || user.defaultGroupId),
          ),
        ),
        naming: !!user.collectionNamingId,
      });
    });
  },
  input(e) {
    this.setData({ [e.currentTarget.dataset.key]: e.detail.value });
  },
  pick(e) {
    const index = Number(e.detail.value);
    wx.setStorageSync("zainaer-tenant", this.data.groups[index].id);
    this.setData({ index });
  },
  language() {
    ui.toggle();
    this.setData({ t: ui.labels() });
  },
  name() {
    ui.run(this, async () => {
      await api.call("PUT", "/users/self/collection-name", {
        name: this.data.name.trim(),
      });
      this.setData({ naming: false });
      wx.navigateBack();
    });
  },
  join() {
    ui.run(this, async () => {
      await api.call(
        "POST",
        "/groups/invitations/" +
          encodeURIComponent(this.data.invitation.trim()),
      );
      wx.navigateBack();
    });
  },
  logout() {
    ui.run(this, async () => {
      await api.call("POST", "/users/logout");
      api.clear();
      wx.reLaunch({ url: "/pages/login/index" });
    });
  },
});
