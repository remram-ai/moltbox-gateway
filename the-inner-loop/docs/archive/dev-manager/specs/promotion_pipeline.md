# Promotion Pipeline Specification

## Purpose

Define the release promotion model from development through production.

## Capability Definition

Capability name:

- Release promotion and approval pipeline

Implementation order:

- 7

Depends on:

- `filesystem_layout.md`
- `runtime_model.md`
- `deployment_primitives.md`
- `control_plane.md`
- `tools_container.md`

Delivers:

- staged release movement from DEV to TEST to PROD
- approval boundaries
- rollback prerequisites
- Codex and operator role separation

Primary implementation reference for:

- publish-run orchestration
- approval gating
- release metadata requirements
- rollback trigger and handling logic

Acceptance criteria:

- release-candidate artifact requirements are defined
- TEST and PROD same-commit rule is defined
- human approval boundary is defined
- Codex direct-to-PROD deployment is prohibited

## Responsibilities

- define release-candidate movement between environments
- define approval boundaries
- define validation requirements
- define rollback prerequisites
- define Codex and human operator roles

## Canonical Flow

```text
DEV iteration
-> deploy candidate to TEST
-> validation
-> create PR
-> human merge
-> deploy to PROD
```

## Release Candidate Model

A release candidate is a specific commit plus its rendered deployment manifest
and metadata.

Required release-candidate metadata:

- commit sha
- manifest id
- tool version
- validation report id
- creation time

Rules:

- TEST and PROD must use the same commit sha
- PROD promotion is allowed only after a validated TEST run
- branch names are not promotion artifacts; commits are

## Pipeline Stages

### Stage 1: DEV Iteration

- Codex may deploy feature work to DEV
- DEV is allowed to drift
- no promotion occurs directly from DEV state

### Stage 2: Candidate Selection

- select a commit from the approved source branch, expected to be `main`
- render the deployment manifest for that commit

### Stage 3: TEST Deployment

- deploy selected commit to TEST
- start TEST
- run validation and smoke tests
- store validation report

### Stage 4: Approval Boundary

- create a PR or equivalent approval artifact
- human operator reviews validation output
- human merge is the approval event

### Stage 5: PROD Promotion

- verify approval event
- snapshot PROD
- deploy the same validated commit to PROD
- run post-deploy validation

### Stage 6: Rollback

- if PROD validation fails, restore snapshot
- record rollback event and failure report

## Codex Role

Codex may:

- iterate in DEV
- trigger TEST deployments
- run validation
- read reports and logs

Codex may not:

- deploy directly to PROD
- bypass approval
- invoke unrestricted shell access

## Human Operator Role

Human operator responsibilities:

- review candidate validation
- merge approval PR
- authorize exceptional rollback or recovery actions when required

## Dependencies

- control-plane specification
- deployment primitives specification
- runtime model specification
- filesystem layout specification

## Example Flows

Happy path:

1. feature is validated in DEV
2. commit from `main` is selected
3. commit is deployed to TEST
4. TEST validation passes
5. PR is merged by human
6. same commit is deployed to PROD

Failure path:

1. TEST validation fails
2. diagnostics are collected
3. no PR merge occurs
4. candidate is rejected or revised
