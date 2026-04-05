# Overview

Moltbox Gateway is the control plane and CLI repo for the Moltbox appliance.

Current design stance:

- this repo is the local design authority for gateway work
- the appliance target is only `gateway`, `caddy`, `ollama`, `openclaw-test`, and `openclaw-prod`
- appliance `dev` is gone
- OpenSearch is gone
- Postgres is deferred to Cortex work
- `prod` is a managed pet
- `test` is the appliance proving lane
- local development happens off-appliance

Core operating model:

- gateway orchestrates the appliance
- OpenClaw owns its native runtime lifecycle
- `moltbox-runtime` owns desired runtime baseline
- `moltbox-services` owns desired service topology
- `remram-skills` owns tracked plugin and skill source
- `remram-cortex` owns the Cortex overlay and bridge-package assets

Primary local model posture:

- provider `ollama`
- model `mistral:7b-instruct-32k`
- context window `32768`

Together escalation must remain available, but it must happen through official OpenClaw provider or plugin behavior rather than hidden gateway routing.

Host posture:

- ZFS-backed critical paths are mandatory
- system-owned paths are mandatory
- human admin access is preserved
- AI access uses restricted SSH identities and forced-command wrappers

Use `../design/README.md` for the full design package.
