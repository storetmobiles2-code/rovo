# Security Policy

We take the security of rovo seriously. rovo handles orders, payments, personal
data and KYC documents, so we are grateful for responsible disclosure.

## Supported versions

rovo is pre-1.0 and ships from `main`. Until `v1.0.0` (Mahabubnagar go-live),
security fixes are applied to the latest release and to `main`. Older tags are
not maintained.

| Version | Supported |
|---|---|
| `main` / latest release | ✅ |
| older pre-1.0 tags | ❌ |

## Reporting a vulnerability

**Please do not open a public issue, pull request or Discussion for a security
problem.** Public disclosure before a fix puts users at risk.

Report privately in either of these ways:

1. **GitHub Private Vulnerability Reporting** (preferred) — open an advisory at
   <https://github.com/storetmobiles2-code/rovo/security/advisories/new>.
2. **Email** — `security@rovo.example` (replace with the project's real contact
   before launch). Encrypt with our PGP key if the report is sensitive.

Please include:

- a description of the issue and its impact,
- steps to reproduce or a proof of concept,
- affected version / commit and configuration,
- any suggested remediation.

## Our commitment (SLAs)

- **Acknowledgement within 72 hours.**
- An initial assessment and severity within 7 days.
- Regular updates until the issue is resolved.
- Credit in the advisory and release notes, if you want it.

## Scope

**In scope:** this repository — the backend API and worker, the four PWAs, the
deployment and infrastructure code, and the CI/CD configuration.

**Out of scope:**

- The production infrastructure of any operator running rovo, unless you have
  that operator's written authorisation.
- Denial-of-service, volumetric or spam testing.
- Social engineering, physical attacks, or attacks on third-party providers.
- Findings that require a compromised device or a privileged local account.

## Rules / safe harbour

- Only test against your **own** local or staging instance (`make dev` /
  `make prod-up`). The whole golden flow runs offline with fakes, so you never
  need to touch anyone's live system.
- **Never test against production payment flows.** The payment provider in this
  repository is a fake simulator; keep it that way for testing.
- Do not access, modify or delete data that is not yours, and do not pivot to
  other systems.
- If you discover personal data, stop, do not download it, and tell us.

Good-faith research that follows this policy is authorised, and we will not
pursue or support legal action against you for it.
