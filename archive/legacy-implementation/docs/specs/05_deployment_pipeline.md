# Deployment Pipeline

The deployment pipeline is expressed through the domain-specific CLI verbs rather than a separate namespace.

Examples:

- `moltbox host ollama deploy`
- `moltbox runtime test deploy`
- `moltbox runtime prod rollback`
- `moltbox tools update`

Rollback must always target the same object domain as the original deployment.
