// SPDX-License-Identifier: MIT; Copyright (c) 2026 Phoobobo
import type { TreeItem } from "~/lib/api/types/data-contracts";
export type MapNode = { id: string; name: string; count: number; path: string[]; children: MapNode[] };
export function storageMap(tree: TreeItem[], home: string): MapNode {
  function countItems(nodes: TreeItem[]): number {
    return nodes.reduce((sum, node) => sum + (node.type === "location" ? 0 : 1) + countItems(node.children || []), 0);
  }
  function locations(nodes: TreeItem[], path: string[]): MapNode[] {
    return nodes
      .filter(n => n.type === "location")
      .map(n => ({
        id: n.id,
        name: n.name,
        path: [...path, n.name],
        count: countItems(n.children || []),
        children: locations(n.children || [], [...path, n.name]),
      }))
      .sort((a, b) => a.name.localeCompare(b.name, undefined, { numeric: true }));
  }
  return { id: "root", name: home, path: [home], count: countItems(tree), children: locations(tree, [home]) };
}
export function flattenLocations(node: MapNode): MapNode[] {
  return [node, ...node.children.flatMap(flattenLocations)];
}
