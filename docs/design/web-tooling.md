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

OpenClaw's native browser surface is not part of the default local baseline on this appliance.

Current caveat:

- browser should only be staged intentionally for a separate diagnostic path
- the chosen local Gemma baseline is solid for `web_search` and `web_fetch`, but noisy fetched pages still need human judgment
- for important tool-heavy tasks, treat `test` validation or a stronger model as the reliability boundary
