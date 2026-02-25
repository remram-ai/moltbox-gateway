# moltbox

**Local hardware appliance for sovereign AI infrastructure.**

Moltbox is the physical control plane for Remram.

It is a local-first hardware environment designed to give you ownership of your context, memory, and AI routing layer.

This is what makes your AI stack sovereign.

---

## What Moltbox Is

Moltbox is a dedicated local machine that runs:

- `remram-os` (orchestration layer)
- `remram-recall` (RAM — retrieval)
- `remram-encode` (REM — consolidation)
- OpenClaw integration
- OpenSearch (memory substrate)
- Local routing model
- Cloud model connectors

The local model is not intended for deep reasoning.
It is optimized for:

- Intent routing
- Tool selection
- Context packaging
- Guardrail enforcement
- Short reasoning loops

Heavy cognition can escalate to cloud models.

---

## Hardware Philosophy

Moltbox separates **control** from **cognition**.

- Local hardware owns memory and routing.
- External models provide stateless expert reasoning.

This ensures:

- Persistent identity
- Data ownership
- Deterministic orchestration
- Token-efficient cloud usage

---

## Minimum System Requirements

Designed around a practical baseline:

- **GPU:** RTX 5060 Ti (16GB VRAM) or equivalent
- **System RAM:** 64GB recommended (128GB ideal for heavy OpenSearch use)
- **CPU:** 12–16 cores preferred
- **Storage:** Dedicated NVMe for OpenSearch index

The 16GB GPU tier supports a local routing model and embedding workloads.
It is not sized for large-scale deep reasoning models.

---

## Recommended Configuration

- 16GB+ VRAM GPU
- 128GB system RAM
- Separate NVMe drives:
  - OS / containers
  - OpenSearch data
- Reliable 850W+ PSU
- Strong cooling and sustained load stability

Moltbox prioritizes stability over peak benchmark performance.

---

## Repository Contents

This repository includes:

- Docker and container configurations
- OpenSearch setup and tuning
- Model runtime configuration
- Environment variables and deployment scripts
- Hardware tuning notes
- Optional configuration profiles

This repository does not contain memory logic.
It deploys and wires the system that runs it.

---

## Architecture Position

```
User
  ↓
Moltbox (local control plane)
  ↓
remram-os
  ↓
RAM / REM
  ↓
Cloud Cognition (optional)
```

Moltbox ensures your memory remains local and sovereign.

