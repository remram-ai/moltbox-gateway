# Remram Deployment Architecture Plan

## Purpose

This document defines the operational deployment model for the Remram system. The goal is to provide a safe, repeatable, and debuggable deployment process suitable for a local-first AI appliance architecture built on top of OpenClaw.

The design emphasizes:

- deterministic runtime behavior
- safe promotion between environments
- strong rollback guarantees
- clear separation between development, deployment tooling, and runtime execution

This document establishes the baseline model for development environments, release promotion, and operational tooling.

---

# System Planes

The system is intentionally divided into two major planes.

## Control Plane (Tools)

The control plane manages deployment and operations. It is responsible for:

- deployment orchestration
- runtime diagnostics
- snapshot and restore
- validation
- release promotion

The control plane **mutates runtime environments but is never mutated by them.**

Control plane components:

- deployment engine
- validation engine
- diagnostics collector
- snapshot manager
- MCP interface

The control plane runs on the host system rather than inside a container.


## Execution Plane (Runtimes)

The execution plane contains the OpenClaw runtimes that actually run Remram.

Each runtime is isolated and represents a different environment.

Execution environments:

- DEV runtime
- TEST runtime
- PROD runtime

Each runtime runs its own OpenClaw container with its own runtime configuration root.

Example runtime roots:

```
~/.openclaw/dev
~/.openclaw/test
~/.openclaw/prod
```

---

# Shared Infrastructure Services

Infrastructure services are shared across all environments.

Shared containers:

- Ollama (local model runtime)
- OpenSearch (vector search and index storage)

These services are not duplicated per environment to avoid unnecessary resource usage and model duplication.

Environment isolation is achieved through configuration and namespace separation.

Example OpenSearch index prefixes:

```
remram-dev-*
remram-test-*
remram-prod-*
```

---

# Environment Model

The system contains three runtime environments.

## DEV Environment

Purpose:

Rapid experimentation and development.

Characteristics:

- full shell access
- debugging allowed
- configuration changes allowed
- runtime may drift

DEV is not considered a stable environment and is not part of the formal deployment pipeline.

Developers may deploy feature branches directly into DEV.

Example usage:

```
git checkout feature/router-fix
git pull
restart dev runtime
```

DEV is allowed to break.


## TEST Environment

Purpose:

Validation of release rcs before production promotion.

Characteristics:

- controlled CLI access
- deployment through pipeline only
- validation and smoke tests executed here

The TEST runtime always runs an immutable release rc.


## PROD Environment

Purpose:

Serve real workloads.

Characteristics:

- minimal surface area
- chat endpoints
- diagnostics
- health checks

No direct shell access or manual runtime mutation is allowed in production.

All changes must occur through the deployment pipeline.

---

# Git Workflow

Git is used as the source-of-truth for code history and release rcs.

Branches:

- main
- feature branches

Feature development occurs on temporary branches.

Example:

```
feature/new-router
feature/opensearch-fix
```

Typical development flow:

```
create feature branch
work locally
push branch to GitHub
test branch in DEV runtime
merge into main when stable
```

Deployment always occurs from a **specific commit**, not from a branch.

---

# Release Candidate Model

A release rc represents a specific commit and configuration set.

Example:

```
commit 7e4a12c
```

Deployment pipeline promotes this commit through environments.

Example promotion flow:

```
deploy commit → TEST
validate
approve
deploy same commit → PROD
```

The same commit must be used for both TEST and PROD deployments.

---

# Deployment Pipeline

The deployment pipeline promotes a release rc between environments.

Pipeline stages:

1. resolve release rc
2. preflight validation
3. snapshot production runtime
4. deploy rc to TEST
5. start TEST runtime
6. run validation and smoke tests
7. operator approval
8. promote rc to PROD
9. post-deploy validation

---

# Snapshot and Rollback

Production mutation requires a snapshot to exist.

Snapshot scope:

- runtime configuration
- deployment metadata
- runtime state necessary for restoration

If deployment fails, rollback restores the previous snapshot.

Rollback is operator-mediated except for clearly safe failures during initial startup.

---

# Validation Strategy

Validation occurs in layers.

## Static Validation

- configuration schema
- required files present

## Stack Validation

- containers healthy
- network configuration valid

## Application Validation

- OpenClaw gateway ready
- model provider reachable

## Smoke Tests

- simple chat request
- tool execution

Validation must succeed before production promotion.

---

# Tools System

Deployment tooling is independent from runtime software.

Tool responsibilities:

- start publish run
- snapshot runtime
- deploy release rc
- run validation
- promote rc
- rollback

Tools interact with runtimes through controlled commands.

The runtime itself does not manage its own deployment.

---

# Tools Deployment

Operational tooling follows a separate lifecycle from Remram runtime releases.

It is not a DEV → TEST → PROD promotion pipeline in the same sense as feature delivery. The tools plane is infrastructure, so its deployment model is closer to a controlled replace-and-verify cycle.

## Tools Deployment Flow

The tools deployment sequence is:

1. run tool unit tests
2. snapshot current tools installation
3. deploy updated tools version
4. restart tools service if needed
5. run tools self-check and operational verification
6. restore previous tools version if verification fails

This process is intentionally simpler than the runtime release pipeline because the blast radius is different. A bad tools deployment can impair the ability to operate the system, but it does not directly represent an application feature release to end users.

## Tools Deployment Characteristics

The tools deployment model should have the following properties:

- self-contained and independent from runtime feature deployment
- mandatory backup before mutation
- fast verification after deployment
- immediate rollback path
- no promotion semantics across DEV, TEST, and PROD runtime environments

## Tools Verification

Verification for tools deployment should include:

- command surface loads correctly
- required scripts are callable
- MCP interface responds
- snapshot and diagnostics commands still function
- deployment commands still resolve runtime targets correctly

## Tools Versioning

Tools should support side-by-side versioning so multiple versions can exist simultaneously on the host. This enables safe upgrades, testing of new toolchains, and immediate rollback without reinstalling software.

Recommended structure:

```
~/.remram/tools
 ├─ versions
 │   ├─ v0.3.1
 │   ├─ v0.3.2
 │   └─ v0.4.0
 │
 ├─ stable -> versions/v0.3.2
 └─ rc -> versions/v0.4.0
 └─ latest -> versions/v0.4.0
```

The running tools service should resolve commands through a stable alias rather than referencing a specific version directory.

Example access paths:

```
/tools/stable/
/tools/latest/
```

This ensures developer tools such as VS Code, Codex, and automation scripts always interact with a consistent tool endpoint while still allowing upgrades to occur underneath the alias.

During deployment:

1. new tools version is installed into `versions/`
2. verification runs against that version
3. if successful, the `stable` symlink is updated
4. previous versions remain available for rollback


## Tools Rollback

If tools verification fails, the previous tools version must be restored immediately.

Tools rollback should be treated as a direct restore operation rather than a staged promotion.

---

# High-Level Architecture

```
HOST
│
├─ Control Plane (tools)
│  ├─ deployment engine
│  ├─ validation engine
│  ├─ snapshot manager
│  ├─ diagnostics
│  └─ MCP interface
│
├─ Shared Services
│  ├─ ollama
│  └─ opensearch
│
└─ Execution Plane
   ├─ openclaw-dev
   ├─ openclaw-test
   └─ openclaw-prod
```

This architecture provides strong operational separation between tooling and execution runtimes while supporting safe promotion and rollback of releases.


---

# Deployment Scripts and Pipeline Integration

Deployment scripts are a core part of the control plane. They implement the concrete mechanics of provisioning, deploying, validating, and restoring runtimes. These scripts must be treated as infrastructure primitives and organized into clear, composable units rather than large monolithic scripts.

## Design Principles

Deployment scripts should follow these rules:

- single-purpose operations
- deterministic behavior
- machine-readable output where possible
- safe to run multiple times when practical
- never silently mutate unrelated system state

Scripts should be callable both by MCP tools and directly by operators when necessary.


## Script Categories

Deployment scripts fall into several categories.

### Runtime Provisioning

Responsible for creating or resetting environment runtimes.

Examples:

```
provision_runtime_dev
provision_runtime_test
provision_runtime_prod
```

Responsibilities include:

- creating runtime directories
- preparing configuration roots
- initializing environment namespaces


### Deployment Execution

Responsible for applying a release rc to a runtime.

Examples:

```
deploy_rc_test
promote_rc_prod
```

Responsibilities include:

- pulling the correct commit
- rendering runtime configuration
- updating container definitions


### Stack Lifecycle

Responsible for starting and stopping services.

Examples:

```
start_runtime
stop_runtime
restart_runtime
```

These scripts wrap container orchestration commands.


### Validation

Responsible for automated system checks.

Examples:

```
validate_runtime_stack
validate_openclaw_gateway
run_smoke_tests
```

Validation scripts must return structured pass/warn/fail results.


### Diagnostics

Responsible for collecting debugging information.

Example:

```
collect_moltbox_debug
```

Diagnostics scripts must never mutate system state.


### Snapshot and Restore

Responsible for creating rollback points and restoring them.

Examples:

```
create_runtime_snapshot
restore_runtime_snapshot
```

Snapshots are mandatory before any production mutation.


---

# Container Architecture Stabilization

Before the full deployment pipeline is finalized, the first major project phase is stabilizing the container architecture used by the execution plane.

The goal is to ensure containers are predictable, modular, and compatible with the environment model defined in this document.


## Current Issues

Initial container layouts tend to evolve organically and often contain issues such as:

- environment assumptions embedded in images
- configuration stored inside containers instead of runtime roots
- inconsistent compose definitions
- runtime configuration drift

These issues must be corrected before reliable deployment automation can be implemented.


## Target Container Model

Execution containers should follow a strict separation between runtime configuration and container image behavior.

Container responsibilities:

- run OpenClaw runtime
- read configuration from runtime root
- connect to shared services

Containers must never contain environment-specific configuration baked into the image.


## Target Runtime Containers

The execution plane will contain three OpenClaw runtime containers.

```
openclaw-dev
openclaw-test
openclaw-prod
```

Each container reads configuration from its corresponding runtime root.


Example:

```
~/.openclaw/dev
~/.openclaw/test
~/.openclaw/prod
```


## Shared Infrastructure Containers

Infrastructure containers are shared across environments.

Examples:

```
ollama
opensearch
```

These services must be configured to support environment namespace separation.


## Container Refactor Goals

The container refactor phase should achieve the following:

- predictable compose definitions
- environment-independent container images
- runtime configuration externalization
- stable networking between runtimes and shared services


Once container architecture is stabilized, the deployment pipeline defined earlier in this document can be implemented reliably.


---

# Implementation Roadmap

The following ordered goals describe the execution strategy for building the Remram deployment and operations system. The roadmap intentionally builds the control plane first so the system can eventually manage itself.

## Goal 1 — Stand up the control plane

Create the first **tools container/service** that exposes a minimal API surface for environment management.

Responsibilities:

- deployment orchestration
- MCP interface for Codex
- coordination of host-side scripts
- tool version management

This container becomes the system's **control plane**.


## Goal 2 — Establish the host control-plane filesystem

Create the persistent host layout used by the tools plane.

Example structure:

```
~/.remram
 ├─ tools
 ├─ host-tools
 ├─ runtimes
 ├─ snapshots
 ├─ deploy
 └─ logs
```

This layout should remain stable throughout the life of the system.


## Goal 3 — Implement deployment primitives

Implement clean **host-side primitives from scratch** instead of refactoring existing bootstrap scripts.

Initial primitive set:

- create_runtime
- destroy_runtime
- start_runtime
- stop_runtime
- restart_runtime
- deploy_commit
- validate_runtime
- collect_diagnostics
- snapshot_runtime
- restore_snapshot

Each primitive must:

- perform exactly one operation
- return structured output
- be callable independently
- be safe to compose into higher-level pipelines

All deployment pipelines will be built from these primitives.


## Goal 4 — Use primitives to create environments

Use the tools plane to construct the runtime topology.

Shared infrastructure:

- Ollama
- OpenSearch

Runtime containers:

- openclaw-dev
- openclaw-test
- openclaw-prod

Each runtime reads configuration from its own runtime root.


## Goal 5 — Stabilize the container architecture

Ensure runtime environments are predictable and isolated.

Requirements:

- container images contain no environment-specific configuration
- runtime configuration lives in runtime roots
- shared services are environment-agnostic
- OpenSearch indexes are namespaced per environment

Once complete, Docker should no longer be manipulated manually. All operations should go through the tools plane.


## Goal 6 — Enable the Codex development loop

Allow Codex to iterate safely in the DEV runtime.

Codex capabilities:

- deploy branch to DEV
- restart DEV runtime
- run tests
- inspect logs
- iterate until validation succeeds

No TEST or PROD promotion occurs at this stage.


## Goal 7 — Add controlled TEST deployment

Allow Codex to promote a selected commit to the TEST runtime.

Workflow:

```
commit from main
↓
deploy to TEST
↓
run validation
↓
generate validation report
```

Codex stops here.


## Goal 8 — Production promotion via Pull Request

Replace informal approval with a GitHub PR-based promotion gate.

Workflow:

```
TEST validation passes
↓
PR created (main → test or test → prod depending on policy)
↓
operator reviews and merges
↓
tools plane detects merge
↓
deploy to PROD
```

This creates a hard approval boundary and a permanent audit trail.


## Goal 9 — Add runtime rollback safety

Before any production mutation, create a runtime snapshot.

Rollback capability:

```
snapshot_runtime(prod)
restore_snapshot(prod)
```

Rollback must always be available.


## Goal 10 — Implement tools versioning

Support side-by-side tool versions.

Example layout:

```
tools/
 ├─ versions/
 ├─ stable/
 └─ rc/
```

New tool releases install to `versions/`, then are verified before promoting `stable`.


## Goal 11 — Implement the tools deployment loop

Tools deploy independently from runtime features.

Deployment flow:

```
unit test tools
↓
snapshot existing tools
↓
install new version
↓
verify functionality
↓
promote stable or rollback
```


## Goal 12 — Pin pipeline versions

Each publish run records the tools version and commit used.

Example:

```
commit: 7e4a12c
tools_version: v0.3.2
```

This ensures deployments are reproducible.


## Goal 13 — Close the full development loop

Final workflow:

```
Codex develops feature
↓
deploy to DEV
↓
iterate until passing
↓
deploy to TEST
↓
validation
↓
PR created
↓
operator merges
↓
deploy to PROD
```

At this point the system becomes a fully automated development and deployment environment.

