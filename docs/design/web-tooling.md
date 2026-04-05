# Web Tooling

This document defines the current web-tooling baseline.

## Gold Baseline

The steady-state appliance baseline provides:

- `web_search`
- `web_fetch`
- `browser`

## Tool Roles

`web_search`:

- built-in OpenClaw tool
- backed by appliance `searxng`
- used to discover current or uncertain external information

`web_fetch`:

- built-in OpenClaw tool
- used to fetch a known URL cheaply

`browser`:

- built-in OpenClaw browser surface
- backed by headless Chromium inside the OpenClaw runtime container
- used for JS-heavy pages or interactive workflows that `web_fetch` cannot handle

## Removed Detour

Removed from the baseline:

- the appliance `playwright` service
- the old custom `web_browser` detour

That path is not documented as steady-state behavior anymore.

## Native Browser Status

OpenClaw's native browser surface is enabled and healthy on this appliance through the official OpenClaw browser path.

Current caveat:

- the runtime/browser integration is proven
- the chosen local model is still inconsistent at deciding to use `web_fetch` or `browser` through ordinary chat
- for important tool-heavy tasks, treat `test` validation or a stronger model as the reliability boundary
