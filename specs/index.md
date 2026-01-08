# FreqShow Specs

/specs is authoritative. If specs conflict with README or agent-context, specs win.

OpenAPI contract lives at specs/openapi.yaml and is the source of truth for HTTP request/response shapes and generated frontend types.

agent-context/development-log.md is historical narrative only; use it for context, not requirements.

Working practice:

- Spec change (or confirm spec) first; then code; update tests when behavior changes.

Spec Inventory:

- specs/openapi.yaml — HTTP contract
- Future domain specs (placeholder)
- Future system design specs (placeholder)
