# `oc agentic proposal rca` (`cli/proposal/rca.go`)

## Purpose

This code implements the **`rca` subcommand** under `oc agentic proposal`. It is the **CLI integration point** between **OpenShift Lightspeed** (where cluster users describe problems) and **lightspeed-agentic-operator** (which reconciles `Proposal` custom resources and runs the agentic analysis workflow).

When a user triggers root-cause analysis from Lightspeed (e.g. via a **`/rca`-style keyword**), Lightspeed is expected to invoke this command with the user’s text as **`--request`**. That creates a **`Proposal` CR** on the cluster so the operator can run analysis using the **IntelliAide** skills image and structured output schema.

## What it does

1. **Defines a JSON Schema** (`rcaOutputSchemaJSON`) that constrains the LLM/agent output to a shape that matches **Int IntelliAide’s `rca_structured`** format: remediation **`options[]`** plus an **`rcaSummary`** object (executive summary, root causes, recommendations, etc.).

2. **Exposes a Cobra command** `NewRCACmd` with:
   - **`--request`** (required): the problem statement (typically the text from Lightspeed).
   - **`--agent`**: which `Agent` CR to use for analysis (default `smart`).
   - **`--skills-image`**: OCI image mounting IntelliAide skills (default `quay.io/rh-ee-cdate/intelliaide-skills:latest`; overridable for dev).
   - **`--target-namespaces`**: optional list of namespaces to focus the RCA.
   - **`--output` / `-o`**: optional `json` or `yaml` to print the created `Proposal` instead of a short success message.
   - Standard kubectl-style **config flags** (kubeconfig, context, namespace resolution via `ConfigFlags`).

3. **On `Run`**: parses the embedded schema into `apiextensionsv1.JSONSchemaProps`, builds a `Proposal` with:
   - `GenerateName: rca-`
   - Label `agentic.openshift.io/source: intelliaide`
   - `Spec.Request` and optional `TargetNamespaces`
   - `Spec.Analysis` pointing at the chosen agent
   - `Spec.Tools.Skills` with the IntelliAide skills image
   - `Spec.Tools.OutputSchema` set to the RCA schema  
   **No execution/verification steps** are configured in spec—the code comments describe this as **advisory RCA**; humans decide follow-up.

4. **Creates** the `Proposal` with the controller-runtime client, then either marshals it (`-o json|yaml`) or prints the created name plus a hint to approve analysis:  
   `oc agentic proposal approve <name> -n <ns> --stage=analysis`

## Why it exists

- **Lightspeed** surfaces conversational cluster help; it does not replace the operator’s reconciliation loop.
- **Triggering work** is done by creating a **`Proposal`** that the **lightspeed-agentic-operator** already knows how to process (analysis sandbox, skills mount, structured output).
- **IntelliAide** supplies long-running RCA (`run_rca`, poll status, `get_rca_result`) via skills in the image; this CLI **wires Lightspeed’s user text + cluster config into that pipeline** in a consistent, repeatable way.

## Operational flow (high level)

Lightspeed (user `/rca` …) → **`oc agentic proposal rca --request "..."`** → **`Proposal` created** → operator runs analysis with IntelliAide skills + schema → results land in the agentic workflow (e.g. analysis / remediation options per operator behavior) → optional human **`approve`** for analysis stage to start (per printed instructions).

## Related code

- Command registration: `cli/proposal/proposal.go` (`NewRCACmd` added under `proposal`).
- Tests: `cli/proposal/rca_test.go` (if present in your branch).

## Defaults and overrides

- Default skills image is documented in code as built from **Dockerfile.skills** in the IntelliAide deliverables repo; use **`--skills-image`** for custom tags/registries.