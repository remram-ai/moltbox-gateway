# Deploy

Phase 1 ends with a direct localhost control path:

```text
host moltbox CLI
  -> http://127.0.0.1:7460
    -> gateway container
      -> /var/run/docker.sock
        -> host Docker Engine
```

Current bootstrap state:

- the host CLI is installed at `~/.local/bin/moltbox`
- the long-running control plane is the `gateway` container
- the bootstrap image tag is `moltbox-gateway:latest`
- the proof command is `moltbox gateway docker run hello-world`

Next implementation threads should extend this scaffold instead of replacing it.

Phase 1 intentionally does not implement full service deployment, dependency containers, runtime environments, or reverse proxy wiring.
