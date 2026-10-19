// SPDX-License-Identifier: MIT; Copyright (c) 2026 Phoobobo
const api = require("./api");
const words = {
  zh: {
    brand: "在哪儿",
    subtitle: "记得放过，就能找到。",
    login: "微信快捷登录",
    createAccount: "首次使用，创建账号",
    bind: "绑定已有账号",
    email: "原账号邮箱",
    password: "原账号密码",
    bindHelp: "只需绑定一次，以后直接微信登录。",
    search: "找什么东西？",
    find: "查找",
    items: "物品",
    locations: "位置",
    add: "添加",
    assistant: "文字助手",
    account: "我的集合",
    empty: "这里还没有记录，添加一件物品试试。",
    more: "加载更多",
    loading: "正在加载…",
    edit: "编辑",
    remove: "删除",
    name: "名称",
    description: "描述",
    location: "存放位置",
    tags: "标签",
    save: "保存",
    photo: "添加照片",
    root: "未指定位置",
    back: "返回",
    send: "发送",
    confirm: "确认保存",
    cancel: "取消",
    saved: "已保存",
    cancelled: "已取消",
    replaced: "已失效",
    logout: "退出登录",
    collection: "当前集合",
    language: "English",
    newName: "给你的集合起个名字",
    error: "操作未完成",
    retry: "重试",
    photoRetry: "记录已保存，请重试照片上传",
    deleteQuestion: "确定删除这条记录？",
    chooseType: "记录类型",
    welcome: "用一句话找东西，或描述要收纳的物品。",
    invitation: "家庭邀请口令",
    join: "加入集合",
    noModel: "请先在网页的集合设置中配置模型",
    sessionExpired: "请重新登录",
    notConfigured: "尚未配置微信登录，请完成小程序注册及服务端配置。",
  },
  en: {
    brand: "Zainaer",
    subtitle: "A place for everything you remember.",
    login: "Continue with WeChat",
    createAccount: "Create a new account",
    bind: "Link existing account",
    email: "Account email",
    password: "Account password",
    bindHelp: "Link once, then sign in with WeChat.",
    search: "What are you looking for?",
    find: "Search",
    items: "Items",
    locations: "Locations",
    add: "Add",
    assistant: "Assistant",
    account: "Collections",
    empty: "No records yet. Add your first item.",
    more: "Load more",
    loading: "Loading…",
    edit: "Edit",
    remove: "Delete",
    name: "Name",
    description: "Description",
    location: "Location",
    tags: "Tags",
    save: "Save",
    photo: "Add photo",
    root: "No parent location",
    back: "Back",
    send: "Send",
    confirm: "Confirm save",
    cancel: "Cancel",
    saved: "Saved",
    cancelled: "Cancelled",
    replaced: "Expired",
    logout: "Sign out",
    collection: "Current collection",
    language: "中文",
    newName: "Name your collection",
    error: "Could not complete the action",
    retry: "Retry",
    photoRetry: "Record saved. Retry photo upload.",
    deleteQuestion: "Delete this record?",
    chooseType: "Record type",
    welcome: "Find belongings or describe what you want to store.",
    invitation: "Family invitation token",
    join: "Join collection",
    noModel: "Configure a model in collection settings on the website first.",
    sessionExpired: "Please sign in again",
    notConfigured: "WeChat sign-in is not configured yet.",
  },
};
function labels() {
  return words[wx.getStorageSync("zainaer-language") || "zh"];
}
function toggle() {
  wx.setStorageSync(
    "zainaer-language",
    wx.getStorageSync("zainaer-language") === "en" ? "zh" : "en",
  );
}
function auth() {
  if (!api.session().token) {
    wx.reLaunch({ url: "/pages/login/index" });
    return false;
  }
  return true;
}
function error(page, e) {
  page.setData({
    error:
      e.status === 503 ? labels().notConfigured : e.message || labels().error,
  });
}
async function run(page, fn) {
  if (page.data.busy) return;
  page.setData({ busy: true, error: "" });
  try {
    await fn();
  } catch (e) {
    error(page, e);
  } finally {
    page.setData({ busy: false });
  }
}
function detail(id) {
  wx.navigateTo({ url: "/pages/detail/index?id=" + encodeURIComponent(id) });
}
module.exports = { labels, toggle, auth, error, run, detail };
