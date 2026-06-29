# KYBa Identity Service Authentication

Status: active fork integration note.
Owner: KYBa Identity / Gitea Fork maintainers.
Last reviewed: 2026-06-29.

## Purpose

This fork delegates user-facing authentication, registration and account recovery to KYBa Identity Service.

Gitea remains the repository UI and Git platform. KYBa Identity Service is the source of truth for who the user is and which identity is allowed to enter the KYBa workspace.

## Current flow

```text
/user/login
  -> KYBa Identity challenge start
  -> /user/identity/challenge
  -> KYBa Identity challenge verify
  -> local Gitea shadow user sync
  -> normal Gitea session
```

Registration:

```text
/user/sign_up
  -> KYBa Identity register
  -> local Gitea shadow user sync
  -> normal Gitea session with grants managed separately
```

Recovery:

```text
/user/forgot_password
  -> KYBa Identity recovery challenge start
```

## UI source

The page structure follows the Lovable KYBa Workspace authentication reference:

- sign in by registered phone;
- delivery method: SMS or voice;
- challenge code screen;
- registration with display name, phone and invitation code;
- account recovery by registered phone.

It is implemented as native Gitea templates, not React.

## Configuration

Add to `app.ini`:

```ini
[kyba.identity]
ENABLED = true
MOCK = false
BASE_URL = https://identity.example.internal
CLIENT_ID = gitea-fork
CLIENT_SECRET =
TIMEOUT = 10s
CHALLENGE_START_PATH = /v1/auth/challenges
CHALLENGE_VERIFY_PATH = /v1/auth/challenges/verify
REGISTER_PATH = /v1/auth/register
RECOVERY_START_PATH = /v1/auth/recovery/start
AUTO_CREATE_USERS = true
REGISTRATION_ENABLED = true
RECOVERY_ENABLED = true
```

Do not commit live secret values.

## Shadow users

Gitea still needs a local user record for sessions, repository permissions and UI rendering.

After KYBa Identity verifies a user, the fork creates or updates a local shadow user. This local user is not the authentication source of truth.

## Security boundary

- Gitea local password login is bypassed when `[kyba.identity] ENABLED=true`.
- Account recovery is delegated to Identity Service.
- Registration is delegated to Identity Service.
- Gitea sessions are still local Gitea sessions after identity verification.
- Repository permissions and future KCP permissions are evaluated by Gitea/KYBa after identity verification.

## Local smoke mode

`MOCK=true` enables deterministic local UI smoke without a live Identity Service.

Use it only for local testing.
