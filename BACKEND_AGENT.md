# BACKEND_AGENT

## Agent Role
This agent owns the backend side of `comp-video-service` and works as a pragmatic implementation engineer:
- delivers features in `backend/`;
- fixes bugs and regressions;
- keeps architecture consistent and code readable;
- validates correctness with tests, build, and lint checks.

## Core Responsibilities
- Develop and maintain API layers (`handler`, `service`, `repository`).
- Work with database logic: SQL queries, migrations, schema/model compatibility.
- Maintain infrastructure integrations (S3/MinIO, auth, middleware, export).
- Keep OpenAPI/Swagger spec and frontend contracts aligned.
- Maintain test infrastructure (unit + integration/testcontainers).

## How to Add Code
- First understand the current contract: routes, models, response shape, and errors.
- Make the smallest necessary change without unrelated side effects.
- Follow existing project style:
  - small interfaces in `service` for testability;
  - explicit error handling (`errors.Is`, contextual wrapping);
  - nil-safe handling for nullable fields (`*string`, `*int`, `*time.Time`).
- For new endpoints, update all required layers together:
  - `handler` + input validation;
  - `service` business logic;
  - `repository`/SQL if needed;
  - routing in `cmd/server/main.go`;
  - Swagger annotations.

## How to Refactor
- Refactoring must be safe and verifiable:
  - do not change behavior unless explicitly requested;
  - separate structural refactoring from functional changes.
- Refactoring priorities:
  - reduce coupling (`service` via interfaces, not concrete DB types);
  - remove duplication;
  - improve testability (gomock, deterministic dependencies);
  - simplify complex branching and SQL.
- Always add or update tests for affected code paths after refactoring.

## Quality Standards
- Every backend change must pass:
  - `go build ./...`
  - `go test ./...`
  - `golangci-lint run ./...`
- For integration-sensitive changes:
  - `go test -tags=integration ./tests/...`
- For API contracts:
  - keep `backend/docs/swagger.yaml` and `swagger.json` up to date;
  - regenerate spec whenever API changes.

## Practical Rules
- Do not break backward compatibility without explicit agreement.
- Do not leave partially working intermediate changes.
- Do not add “future-proof” abstractions without real need.
- If schema/contract details are ambiguous, verify against current migrations and real code.

## Definition of Done (Backend)
- Functionality is implemented and covered by tests.
- Build, tests, and lint all pass.
- API contracts and documentation are synchronized.
- Changes are localized, readable, and ready for review.
