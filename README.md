# remram-gateway

Execution substrate for the Remram system.

Remram Gateway provisions and runs the local OpenClaw control plane that powers Remram. It wires models, registers agents, connects to memory services, and defines the concrete runtime profile for a given machine.

If you want to stand up a Remram runtime, this is the repository you use.

---

## Purpose

`remram-gateway` owns the execution environment in which Remram operates.

It is responsible for:

- Installing and configuring the OpenClaw runtime
- Wiring local and cloud model endpoints
- Provisioning and configuring OpenSearch
- Defining runtime-level escalation policy
- Registering available agents
- Defining appliance execution profiles (e.g., Moltbox)

Gateway declares what the runtime loads and how it runs.

---

## Repository Structure

```
remram-gateway/
  README.md

  .openclaw/
    agents.yaml        # Registry of installed agents
    models.yaml        # Local + cloud model endpoints
    tools.yaml         # Globally registered tools
    escalation.yaml    # Runtime escalation policy
    env.example        # Required environment variables

  moltbox/
    docker-compose.yml
    opensearch.yml
    model-runtime.yml
    bootstrap.sh
    README.md          # Moltbox deployment notes
    design.md          # Moltbox design documentation
```

---

## Directory Responsibilities

### `.openclaw/`

Runtime configuration surface.

This directory contains all configuration that conforms to the OpenClaw specification, including:

- Agent registration
- Tool registration
- Model routing configuration
- Escalation thresholds
- Required runtime environment variables

Behavioral logic and prompts live in `remram-agents`.

### `moltbox/`

Concrete appliance profile.

This directory defines how to run the Remram runtime on a specific local hardware configuration.

It contains:

- Container stack definitions
- Search service configuration
- Model runtime wiring
- Bootstrap utilities
- Deployment notes
- Design documentation

If additional appliance profiles are introduced (e.g., lightweight dev node, alternate hardware tier), they will appear as sibling directories.

Example:

```
remram-gateway/
  .openclaw/
  moltbox/
  sparkbox/
  dev-node/
```

Each profile defines infrastructure assumptions and resource sizing.

---

## What Does Not Live Here

The following concerns belong in other repositories:

- Agent definitions → `remram-agents`
- Knowledge and memory services → `remram-cortex`
- User interface and client applications → `remram-app`
- System-level documentation → `remram`

This repository is strictly the execution substrate.

---

## Usage

To stand up a Remram runtime:

1. Select an appliance profile (e.g., `moltbox/`).
2. Configure required environment variables.
3. Launch the container stack.
4. Ensure OpenClaw loads agents defined in `.openclaw/agents.yaml`.
5. Connect to the runtime via supported interfaces.

Changes to agent behavior should be made in `remram-agents`.

