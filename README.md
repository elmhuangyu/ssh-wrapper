# ssh-wrapper

A minimal SSH wrapper designed for AI agents running in "yolo mode" — where the agent has full autonomy including ssh.

## Why This Exists

When an AI agent has full shell access, it can accidentally (or intentionally) access SSH keys and connect to arbitrary hosts. This wrapper prevents that by:

- **Blocking unauthorized hosts** — Only hosts in the allowlist can be reached
- **Hiding SSH keys from the agent** — The agent never sees the private key; the wrapper invokes SSH with the key directly
- **Tamper detection** — Any modification to config, key, or SSH binary causes immediate exit

## How It Works

The wrapper replaces `/usr/bin/ssh` inside the container. Every SSH call made by the agent (including those triggered by `git clone`, `git push`, etc.) passes through the wrapper first.

On each invocation it:

1. Verifies that the config file, private key, and the real `ssh` binary are all owned by root with strict permissions — any tampering causes an immediate exit
2. Logs the full command with a timestamp to the configured log path
3. Checks the target host against the allowlist in `config.yaml` — if the host is not listed, the call is denied with `Access Denied` and exits non-zero
4. Clears the environment and re-invokes the real SSH binary using the managed private key, so the agent never has direct access to the key itself

The binary runs with the `setuid` bit set, allowing it to read root-owned secrets even when invoked by an unprivileged user (uid 1000).

## Security Model

| What is protected | How                                                          |
| ----------------- | ------------------------------------------------------------ |
| Private key       | Owned by root, mode 0400, never exposed to the agent process |
| Config file       | Same — root-owned, tamper detection on startup               |
| SSH binary        | Integrity check on startup                                   |
| Unknown hosts     | Denied before any network connection is made                 |
| Environment       | Cleared before exec — no agent-injected env vars reach ssh   |

The agent can only reach hosts explicitly listed in `config.yaml`. Everything else is blocked and logged.

## Configuration

Mount two files into the container as root-owned secrets:

**`/etc/config.yaml`** — the config of the ssh-wrapper, mode 0400, owned by root.

```yaml
logpath: /var/log/ssh-wrapper/ssh-wrapper.log

allowed:
  - host: github.com
    path_prefix:
      - elmhuangyu/
```

**`/etc/key`** — the private key, mode 0400, owned by root.

## Docker Usage

```dockerfile
FROM ghcr.io/AgentDrasil/ssh-wrapper
```

Or build from source:

```bash
docker build -t ssh-wrapper .
```

Run the container with secrets injected by the entrypoint (which must run as root before dropping to uid 1000):

```bash
docker run \
  -v ./my-key:/run/secrets/ssh_key:ro \
  -v ./config.yaml:/run/secrets/ssh_config:ro \
  -v ./logs:/var/log/ssh-wrapper \
  ssh-wrapper
```

The entrypoint copies the secrets to `/etc/key` and `/etc/config.yaml` with correct ownership and permissions, then drops to uid 1000 before handing off to the agent process.

See `test-compose.yaml` for a complete Docker Compose example with secrets handling.

## E2E Tests

Tests run entirely locally via Docker Compose — no secrets are stored anywhere. A fresh SSH key pair is generated on every run.

```bash
uvx pytest -v --log-cli-level=INFO -s
```

The test suite spins up two containers: `test-app` (the wrapper image) and `git-server` (a local SSH git server). It verifies that:

- `git clone`, `git push`, and `git pull` succeed against the allowed host
- SSH to a non-allowlisted host is blocked with `Access Denied`
- All activity is written to the log file

Tests also run in GitHub Actions on every push and pull request, with no secrets required.

## File Structure

```
.
├── main.go                  # wrapper entrypoint
├── lib/
│   ├── command/             # allowlist enforcement
│   ├── config/              # config parsing
│   └── files/               # security verification
├── Dockerfile
├── test-compose.yaml        # e2e test environment
├── docker-entrypoint.sh     # sets permissions, drops to uid 1000
└── e2e/
    └── test_e2e.py          # test runner
```

## License

Apache 2.0

## TODO

- Add s6 overlay
