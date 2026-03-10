# Deployment Pipeline

## Purpose

Enable controlled promotion from development through testing into production.

This domain defines the validated path from DEV iteration to TEST candidate
validation to PROD deployment, including approval boundaries and rollback
capability.

## Prerequisites

This capability depends on earlier domains:

- Control Plane
- Container Deployment System
- Runtime Stack
- Runtime Tooling for validation execution and reporting

## Capability Provided

This capability provides:

- candidate registration from immutable Git commits
- stateful promotion from DEV to TEST to PROD
- TEST deployment and validation tied to one exact commit
- production deployment gated by explicit human approval through pull request
  merge
- production snapshot and restore capability

This is the domain that turns a working inner loop into a real deployment
system.

## Owned Components

The deployment pipeline owns:

- candidate registration and promotion rules
- candidate progression tracking
- TEST validation reports
- approval requirements for PROD deployment
- production deployment execution rules
- rollback and restore rules

It does not own:

- the base control-plane process
- generic deployment rendering
- generic runtime creation
- shared-service setup
- the existence of shared services or runtime roots

Those capabilities are prerequisites from earlier domains.

## Release Candidate Contract

The deployment candidate is an immutable Git commit.

Rules:

- pipeline commands operate on an explicit commit hash
- a candidate must not be identified by branch name, `HEAD`, tag, or other
  moving reference
- the CLI must reject ambiguous or moving refs as deployment inputs
- the same commit validated in TEST is the only commit eligible for PROD
  deployment

## Canonical Interfaces

Representative command surface:

- `remram candidate start <commit>`
- `remram deploy test --commit <commit>`
- `remram validate test --commit <commit>`
- `remram deploy prod --commit <commit>`
- `remram rollback prod`

Rules:

- the CLI surface remains available across environments
- policy controls whether a given caller may invoke a command
- Codex may operate DEV and TEST flows through MCP-backed CLI calls
- Codex must not bypass production approval boundaries
- commands operate on explicit candidate identity
- promotion is stateful and path-constrained
- the CLI must reject promotion of unknown or untracked commits
- the CLI must reject attempts to bypass required earlier stages
- CLI errors must be informative and machine-usable
- errors must identify the candidate commit, state the missing prerequisite, and
  tell the caller what to do next

## Candidate Progression Contract

The pipeline must track, at minimum, the following candidate progression:

- registered in DEV
- validated for TEST eligibility
- deployed to TEST
- validated in TEST
- approved for PROD through merged pull request
- deployed to PROD

Rules:

- a candidate must first be registered or started in DEV
- TEST promotion is allowed only after required DEV-stage conditions are
  satisfied
- PROD promotion is allowed only after the same commit has been deployed to TEST
  and passed TEST validation
- progression is commit-specific and must not transfer across different commits

## DEV To TEST Contract

The inner loop is considered closed when a developer can:

```text
edit -> deploy -> restart -> validate -> iterate
```

in DEV and then prepare a candidate for TEST.

The CLI must enforce that a candidate is known to the pipeline before TEST
promotion is attempted.

TEST validation responsibilities:

- deploy a selected candidate
- run validation
- persist a validation report tied to the exact candidate commit

## TEST Validation Artifact Contract

TEST validation must produce a persistent validation artifact for the exact
candidate commit.

The artifact must capture, at minimum:

- candidate commit
- target environment
- validation execution record
- endpoint or smoke-test results
- pass/fail outcome

This artifact is part of the promotion packet and is the proof that the TEST
deployment was actually exercised.

Promotion to PROD must fail if this artifact is missing or failed.

## TEST To PROD Contract

PROD deployment requires:

- a candidate commit that has passed through TEST successfully
- a TEST validation artifact for that same commit
- explicit human approval through pull request merge
- enforcement that production deployment is blocked without that approval

The merged pull request is the approval boundary and audit trail.

Codex or other agent-driven validation may operate through TEST validation, but
promotion to PROD still requires explicit human approval through the pull
request process.

The merged pull request authorizes deployment of the exact previously tested
commit.

## Snapshot And Restore Contract

Production mutation requires rollback capability.

Snapshot is mandatory before mutating production.

Minimum rollback capability:

- create a production snapshot
- include runtime configuration and runtime state in that snapshot
- restore that snapshot
- verify observable state after restore

Snapshot and restore remain a separate concern from approval, but both belong to
the production deployment capability.

Snapshot and rollback are production safety requirements only. They are not
required DEV or TEST behaviors in this capability contract.

## Rollback Contract

`remram rollback prod` restores the pre-deploy production snapshot associated
with the current or most recent production deployment attempt.

Rules:

- rollback target selection must be deterministic
- rollback restores the captured production snapshot
- rollback must not resolve to an arbitrary earlier deployment unless explicitly
  documented by the CLI contract

## Validation

This capability exists when:

- a candidate can be registered from an explicit commit hash
- TEST validation report can be generated for that exact commit
- the CLI rejects moving refs and unknown commits
- the CLI rejects promotion attempts that bypass required earlier stages
- production deployment is blocked without merged pull request approval
- production deployment proceeds for the exact previously tested commit when
  approval is present
- snapshot and restore produce observable, verifiable state transitions

## Deliverable Example

```text
remram candidate start <commit>
remram deploy test --commit <commit>
remram validate test --commit <commit>
remram deploy prod --commit <commit>
remram rollback prod
```
