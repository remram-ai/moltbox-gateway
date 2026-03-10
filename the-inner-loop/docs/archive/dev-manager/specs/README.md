# Remram Dev Manager Specifications

This directory contains the formal specification set for the new `dev-manager`
control-plane architecture.

These documents are intended to be the primary implementation reference
artifacts for the new system.

Reference rules:

- each implementation phase must name the spec it is satisfying
- each spec is an independently deliverable capability
- each capability depends only on the capabilities listed before it
- implementation should not jump ahead of unresolved earlier capabilities

Review order:

1. `filesystem_layout.md`
2. `repository_structure.md`
3. `runtime_model.md`
4. `deployment_primitives.md`
5. `control_plane.md`
6. `tools_container.md`
7. `promotion_pipeline.md`

Implementation should not begin until these documents are reviewed together and
the interfaces between them are accepted.

Capability ladder:

1. Filesystem layout:
   establishes the host-state contract all later capabilities depend on
2. Repository structure:
   establishes where implementation artifacts live in the repo
3. Runtime model:
   establishes the execution-plane objects managed by the system
4. Deployment primitives:
   establishes the host-side operations available to the control plane
5. Control plane:
   establishes orchestration, registry, and Codex-facing tool contracts
6. Tools container:
   establishes the packaging and versioned runtime for the control plane
7. Promotion pipeline:
   establishes staged release movement and approval boundaries

Each document should be treated as complete enough to implement on its own once
all earlier capabilities are accepted.
