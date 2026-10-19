// SPDX-License-Identifier: MIT; Copyright (c) 2026 Phoobobo
const { HouseholdItems, queryString } = require("../shared/household");
const config = require("../config");
const key = "zainaer-session";
const session = () => wx.getStorageSync(key) || {};
const save = (value) => wx.setStorageSync(key, value);
const clear = () => {
  wx.removeStorageSync(key);
  wx.removeStorageSync("zainaer-tenant");
};
const tenant = () => wx.getStorageSync("zainaer-tenant") || "";
function origin() {
  if (!/^https:\/\/[^/?#]+$/.test(config.apiOrigin))
    throw new Error("请先配置 HTTPS 服务地址 / Configure HTTPS API origin");
  return config.apiOrigin;
}
const route = (path, query) => "/api/v1" + path + queryString(query);
function headers() {
  return {
    "Content-Type": "application/json",
    Authorization: session().token || "",
    "X-Tenant": tenant(),
  };
}
function request(method, url, body) {
  return new Promise((resolve, reject) => {
    let base;
    try {
      base = origin();
    } catch (e) {
      reject(e);
      return;
    }
    wx.request({
      url: base + url,
      method,
      data: body,
      header: headers(),
      timeout: 180000,
      success(res) {
        if (res.statusCode === 401 && !url.includes("/users/login")) {
          clear();
          wx.reLaunch({ url: "/pages/login/index" });
        }
        resolve({
          response: res,
          status: res.statusCode,
          error: res.statusCode < 200 || res.statusCode >= 300,
          data: res.data,
        });
      },
      fail: () =>
        reject(new Error("网络连接失败，请重试 / Network request failed")),
    });
  });
}
const transport = Object.fromEntries(
  ["get", "post", "put", "delete"].map((method) => [
    method,
    (args) => request(method.toUpperCase(), args.url, args.body),
  ]),
);
async function unwrap(promise) {
  const res = await promise;
  if (res.error) {
    const e = new Error(
      res.data?.message ||
        res.data?.error ||
        `请求失败 / Request failed (${res.status})`,
    );
    e.data = res.data;
    e.status = res.status;
    throw e;
  }
  return res.data;
}
const call = (method, path, body) => unwrap(request(method, route(path), body));
const items = new HouseholdItems(transport, route);
function code() {
  return new Promise((resolve, reject) =>
    wx.login({
      timeout: 10000,
      success: (r) =>
        r.code ? resolve(r.code) : reject(new Error("微信登录失败")),
      fail: () => reject(new Error("微信登录失败")),
    }),
  );
}
async function login(register = false) {
  const value = await call("POST", "/users/login/wechat", {
    code: await code(),
    register,
  });
  clear();
  save(value);
}
async function bind() {
  return call("POST", "/users/self/wechat", { code: await code() });
}
function photo(item) {
  const id = item.thumbnailId || item.imageId;
  if (!id) return "";
  return (
    origin() +
    route(
      `/entities/${encodeURIComponent(item.id)}/attachments/${encodeURIComponent(id)}`,
      {
        access_token: session().attachmentToken,
        tenant: tenant() || undefined,
      },
    )
  );
}
function upload(id, filePath) {
  return new Promise((resolve, reject) =>
    wx.uploadFile({
      url: origin() + route(`/entities/${encodeURIComponent(id)}/attachments`),
      filePath,
      name: "file",
      header: { Authorization: session().token, "X-Tenant": tenant() },
      formData: { type: "photo", name: "photo.jpg", primary: "true" },
      timeout: 60000,
      success: (r) =>
        r.statusCode >= 200 && r.statusCode < 300
          ? resolve()
          : reject(new Error("照片上传失败 / Photo upload failed")),
      fail: () => reject(new Error("照片上传失败 / Photo upload failed")),
    }),
  );
}
module.exports = {
  call,
  items,
  unwrap,
  login,
  bind,
  session,
  save,
  clear,
  tenant,
  photo,
  upload,
};
