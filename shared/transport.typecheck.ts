// SPDX-License-Identifier: MIT; Copyright (c) 2026 Phoobobo
// Compile-only contract: sharing the API must retain browser Response metadata.
import { HouseholdItems } from "./household";
import type { Requests } from "../frontend/lib/requests/requests";
declare const http: Requests;
const api = new HouseholdItems(http, (path) => path);
const browserResponse: Promise<{ response: Response }> = api.get("id");
void browserResponse;
