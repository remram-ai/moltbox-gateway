# Web Tooling

This document defines the current web-tooling baseline.

## Gold Baseline

The steady-state appliance baseline provides:

- `web_search`
- `web_fetch`

## Tool Roles

`web_search`:

- built-in OpenClaw tool
- backed by appliance `searxng`
- used to discover current or uncertain external information

`web_fetch`:

- built-in OpenClaw tool
- used to fetch a known URL cheaply

## Removed Detour

Removed from the baseline:

- the appliance `playwright` service
- the old custom `web_browser` detour

That path is not documented as steady-state behavior anymore.

## Native Browser Status

OpenClaw’s native browser surface exists upstream, but it is not yet a proven gold-baseline dependency on this appliance.

Until it is proven through official OpenClaw browser flows on the box, it stays out of the baseline and out of the service inventory.
