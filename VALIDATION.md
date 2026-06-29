# KYBa Gitea Fork Validation

This repository is intended to be built and deployed as a Gitea fork.

## Required checks in an environment with Go 1.26.3 and Node dependencies

```bash
go test ./...
make build
make frontend
```

## KCP repository-native web UI smoke

After starting the fork, open an existing repository and verify the `KYBa KCP` tab appears next to the normal repository tabs.

Example smoke URLs:

```bash
curl -fsS http://127.0.0.1:3000/<owner>/<repo>/kcp
curl -fsS http://127.0.0.1:3000/<owner>/<repo>/kcp/imports
curl -fsS http://127.0.0.1:3000/<owner>/<repo>/kcp/exports
curl -fsS http://127.0.0.1:3000/<owner>/<repo>/kcp/impact
```

Expected: each response renders inside the repository shell and contains `KYBa repository interface`.

## KCP API smoke

Use an admin token:

```bash
curl -fsS -H "Authorization: token <admin-token>" http://127.0.0.1:3000/api/v1/kcp/capsules
curl -fsS -H "Authorization: token <admin-token>" http://127.0.0.1:3000/api/v1/kcp/imports
curl -fsS -H "Authorization: token <admin-token>" http://127.0.0.1:3000/api/v1/kcp/impact
curl -fsS -H "Authorization: token <admin-token>" http://127.0.0.1:3000/api/v1/kcp/export-plan
```

## Local sandbox limitation recorded during assembly

The assembly sandbox had Go 1.23.2 and no network access to download the Go 1.26.3 toolchain required by Gitea 1.26.4. Therefore full `go test ./...` and `make build` must be run by the owner in a proper Gitea build environment.


## KYBa Identity Service auth checks

The fork includes delegated auth pages and route hooks for KYBa Identity Service. After enabling `[kyba.identity]`, verify:

```bash
curl -fsS http://127.0.0.1:3000/user/login
curl -fsS http://127.0.0.1:3000/user/sign_up
curl -fsS http://127.0.0.1:3000/user/forgot_password
```

Expected: each page renders KYBa Identity wording. With `MOCK=true`, sign-in challenge flow can be exercised without a live service.
