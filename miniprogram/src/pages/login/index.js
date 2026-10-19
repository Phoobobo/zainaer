// SPDX-License-Identifier: MIT; Copyright (c) 2026 Phoobobo
const api = require("../../lib/api"),
  ui = require("../../lib/ui");
Page({
  data: {
    t: ui.labels(),
    busy: false,
    error: "",
    unbound: false,
    registrationAllowed: false,
    link: false,
    email: "",
    password: "",
  },
  onShow() {
    this.setData({ t: ui.labels() });
    if (api.session().token) wx.reLaunch({ url: "/pages/home/index" });
  },
  input(e) {
    this.setData({ [e.currentTarget.dataset.key]: e.detail.value });
  },
  language() {
    ui.toggle();
    this.setData({ t: ui.labels() });
  },
  link() {
    this.setData({ link: !this.data.link, error: "" });
  },
  login(e) {
    ui.run(this, async () => {
      try {
        await api.login(e.currentTarget.dataset.register === true);
        wx.reLaunch({ url: "/pages/home/index" });
      } catch (err) {
        if (err.data?.code === "wechat_unbound") {
          this.setData({
            unbound: true,
            registrationAllowed: err.data.registrationAllowed,
          });
          return;
        }
        throw err;
      }
    });
  },
  bind() {
    ui.run(this, async () => {
      const token = await api.call("POST", "/users/login", {
        username: this.data.email.trim(),
        password: this.data.password,
        stayLoggedIn: true,
      });
      api.save(token);
      try {
        await api.bind();
        this.setData({ password: "" });
        wx.reLaunch({ url: "/pages/home/index" });
      } catch (e) {
        try {
          await api.call("POST", "/users/logout");
        } catch (_) {}
        api.clear();
        throw e;
      }
    });
  },
});
