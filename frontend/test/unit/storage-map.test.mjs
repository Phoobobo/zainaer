// SPDX-License-Identifier: MIT; Copyright (c) 2026 Phoobobo
import { test } from 'node:test';
import assert from 'node:assert/strict';
import { storageMap, flattenLocations } from '../../components/Location/Map/model.ts';

test('storage map keeps empty locations and excludes belongings from location nodes', () => {
  const item = { id: 'earphones', name: '耳机', type: 'item', children: [] };
  const tree = [{ id: 'room', name: '书房', type: 'location', children: [
    { id: 'drawer', name: '抽屉', type: 'location', children: [item] },
    { id: 'empty', name: '空柜', type: 'location', children: [] },
    { id: 'bag', name: '包', type: 'item', children: [{ ...item, id: 'cable' }] },
  ] }];
  const root = storageMap(tree, '家庭');
  assert.equal(root.count, 3);
  assert.equal(root.children[0].count, 3);
  assert.deepEqual(new Set(flattenLocations(root).map(n => n.id)), new Set(['root', 'room', 'drawer', 'empty']));
  assert.deepEqual(flattenLocations(root).find(n => n.id === 'drawer').path, ['家庭', '书房', '抽屉']);
  assert.equal(flattenLocations(root).find(n => n.id === 'empty').count, 0);
});
test('an empty household still has a selectable root', () => {
  assert.deepEqual(storageMap([], 'Home'), { id: 'root', name: 'Home', count: 0, path: ['Home'], children: [] });
});
