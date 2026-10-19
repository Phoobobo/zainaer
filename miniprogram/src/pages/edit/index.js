// SPDX-License-Identifier: MIT; Copyright (c) 2026 Phoobobo
const api = require("../../lib/api"),
  ui = require("../../lib/ui");
const { householdEntityTypes } = require("../../shared/household");
Page({
  data: {
    t: ui.labels(),
    name: "",
    description: "",
    locations: [],
    locationNames: [],
    locationIndex: 0,
    tags: [],
    tagIds: [],
    types: [],
    typeNames: [],
    typeIndex: 0,
    busy: false,
    error: "",
    photoPath: "",
    ready: false,
  },
  onLoad(q) {
    this.id = q.id || "";
    this.isLocation = q.location === "true";
    if (ui.auth()) this.load();
  },
  load() {
    ui.run(this, async () => {
      const [types, tags] = await Promise.all([
        api.call("GET", "/entity-types"),
        api.call("GET", "/tags"),
      ]);
      let locations = [],
        page = 1;
      while (true) {
        const data = await api.unwrap(
          api.items.getAll({ isLocation: true, page, pageSize: 100 }),
        );
        locations.push(...data.items);
        if (locations.length >= data.total || !data.items.length) break;
        page++;
      }
      locations = locations.filter((x) => x.id !== this.id);
      const selected = householdEntityTypes(types);
      this.setData({
        types: selected,
        typeNames: selected.map((t) =>
          t.isLocation ? this.data.t.locations : this.data.t.items,
        ),
        typeIndex: Math.max(
          0,
          selected.findIndex((t) => t.isLocation === this.isLocation),
        ),
        tags: tags.map((t) => ({ ...t, checked: false })),
        locations,
        locationNames: [this.data.t.root, ...locations.map((l) => l.name)],
      });
      if (this.id) {
        this.original = await api.unwrap(api.items.get(this.id));
        const x = this.original;
        if (x.entityType && !selected.some((t) => t.id === x.entityType.id)) {
          selected.push(x.entityType);
          this.setData({
            types: selected,
            typeNames: selected.map((t) => t.name),
          });
        }
        this.setData({
          name: x.name,
          description: x.description,
          typeIndex: Math.max(
            0,
            selected.findIndex((t) => t.id === x.entityType.id),
          ),
          locationIndex: locations.findIndex((l) => l.id === x.parent?.id) + 1,
          tagIds: x.tags.map((t) => t.id),
          tags: this.data.tags.map((t) => ({
            ...t,
            checked: x.tags.some((v) => v.id === t.id),
          })),
        });
      }
      this.setData({ ready: true });
    });
  },
  input(e) {
    this.setData({ [e.currentTarget.dataset.key]: e.detail.value });
  },
  pick(e) {
    this.setData({ [e.currentTarget.dataset.key]: Number(e.detail.value) });
  },
  tags(e) {
    this.setData({ tagIds: e.detail.value });
  },
  photo() {
    wx.chooseMedia({
      count: 1,
      mediaType: ["image"],
      success: (r) => this.setData({ photoPath: r.tempFiles[0].tempFilePath }),
    });
  },
  save() {
    ui.run(this, async () => {
      if (!this.data.ready || !this.data.name.trim()) return;
      const body = {
        ...(this.original || {}),
        name: this.data.name.trim(),
        description: this.data.description,
        entityTypeId: this.data.types[this.data.typeIndex].id,
        parentId: this.data.locations[this.data.locationIndex - 1]?.id || null,
        quantity: this.original?.quantity || 1,
        tagIds: this.data.tagIds,
      };
      const saved = await api.unwrap(
        this.id ? api.items.update(this.id, body) : api.items.create(body),
      );
      this.id = saved.id;
      this.original = saved;
      if (this.data.photoPath) {
        try {
          await api.upload(this.id, this.data.photoPath);
        } catch (_) {
          throw new Error(this.data.t.photoRetry);
        }
      }
      wx.redirectTo({
        url: "/pages/detail/index?id=" + encodeURIComponent(this.id),
      });
    });
  },
});
