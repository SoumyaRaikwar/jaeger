# Architecture Decision Record: Configuration Schema Generation

## Status
Proposed

## Context
As part of Jaeger V2's migration to rely heavily on the OpenTelemetry Collector architecture, we need a robust mechanism to document, validate, and expose the configuration surface of Jaeger-specific components (extensions, receivers, storage plugins). 

Currently, configuration structures are defined purely in Go structs. This leads to drift between code and documentation and completely lacks machine-readable definitions that UIs or CI pipelines can consume.

We have adopted the `schemagen` tool from `opentelemetry-collector-contrib` (introduced in PR #7947 and applied to internal extensions in PR #7952) to automate the generation of JSON schemas from our Go configuration structs. However, questions remain regarding how these generated schemas will be consumed, how external references (e.g., to core OTel types) are resolved, and how this relates to upstream OpenTelemetry initiatives.

### Upstream OpenTelemetry Context (Issue #14548)
OpenTelemetry Collector PR #14548 (`[cmd/mdatagen] Add component config JSON schema generation`) introduces schema generation directly via `mdatagen`, utilizing a `config` section within `metadata.yaml` to support JSON Schema draft 2020-12, validation constraints, and schema composition.

While `mdatagen` focuses on metadata-driven schema definition, `schemagen` (our current approach) generates schemas dynamically via Go reflection on the actual configuration structs.

## Decision

### 1. Schema Generation Approach (Go Reflection vs. mdatagen)
We will continue to use `schemagen` for Jaeger V2 components.
**Argument:** Jaeger's configuration heavily utilizes complex, shared structs (especially around storage backends and tenancy) that map cleanly using Go reflection. While `mdatagen` is progressing in upstream OTel, requiring developers to manually define schemas in `metadata.yaml` duplicates effort and risks drift from the actual Go structs. `schemagen` guarantees the schema exactly matches the code.

*Note: Once `mdatagen` matures to support Go struct reflection natively, we will evaluate migrating to it to fully align with upstream.*

### 2. Handling External References ($ref)
The fundamental question raised in PR #7952 is: *How would an external reference be used?* 

Example from `expvar` schema:
```yaml
$ref: go.opentelemetry.io/collector/config/confighttp.server_config
```

**Decision on References:**
Generated schemas will leave external `$ref` pointers intact as logical URIs pointing to the Go module path.

**How they will be used:**
We will implement a build-time **Schema Resolver/Bundler** (likely a lightweight script added to the Makefile). This resolver will be responsible for:
1. Downloading or locating the corresponding schema files from upstream dependency repositories (e.g., fetching the `confighttp.server_config` schema from the Collector's repo).
2. "Bundling" or "Dereferencing" the schema prior to final packaging or documentation generation, ensuring the end-user receives a fully resolved, self-contained JSON schema.

Until upstream repositories consistently publish their generated schemas alongside their releases, our bundler may need to maintain a local cache or mapping of common OTel configuration schemas.

### 3. Usage & Integration Points
The generated schemas will serve three primary purposes:

1. **Automated Documentation (Phase 3):** We will build a generator that converts the bundled JSON schemas into markdown documentation, ensuring our docs are always 100% accurate regarding available configuration fields, types, and defaults.
2. **CI Check (`check-generate`):** A strictly enforced CI target will fail if a developer modifies a configuration struct without regenerating the corresponding schema, identical to how `mockery` and `protobuf` are enforced today.
3. **Editor/IDE Support:** The fully resolved root schema will be exposed for IDEs (e.g., via `yaml-language-server`) to provide autocomplete and validation for `jaeger-v2.yaml` configuration files.

## Consequences

**Positive:**
* Configuration documentation becomes automated and guaranteed accurate.
* IDEs can provide rich autocomplete for Jaeger config files.
* CI ensures schemas and code never drift.

**Negative:**
* Requires maintaining a Schema Bundler/Resolver to handle cross-repository `$ref` resolution until the broader OTel ecosystem standardizes schema publishing.
* Temporary divergence from the `mdatagen`-centric approach being explored upstream, requiring future reconciliation.
