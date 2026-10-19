# Telugu translations — native review queue

Every Telugu string in `locales/te/*.json` was written with machine assistance by the
Frontend Engineer (foundation) agent on 2026-10-12 and **has not yet been reviewed by a
native speaker**. Per doc 17 §10, a native-speaker reviewer must approve them before the
pilot. Tick each namespace when reviewed and note changes in the PR.

| Namespace | Keys | Status | Reviewer | Notes |
|---|---|---|---|---|
| `common` | ~70 | ☐ machine-assisted, needs native review | | Buttons kept short; "లాగిన్ / లాగ్ అవుట్" kept as loanwords |
| `auth` | ~35 | ☐ machine-assisted, needs native review | | Check OTP wording ("కోడ్" vs "OTP"); `wrongApp.title` builds "ఈ యాప్ {{audience}} కోసం" from genitive plural nouns |
| `errors` | ~45 | ☐ machine-assisted, needs native review | | Tone should be polite, simple; avoid formal Sanskritised words |
| `customer` | ~30 | ☐ machine-assisted, needs native review | | "చెక్అవుట్" loanword |
| `restaurant` | ~20 | ☐ machine-assisted, needs native review | | Kitchen staff vocabulary: "ఆర్డర్", "స్టాక్", "షిఫ్ట్" kept as loanwords |
| `rider` | ~20 | ☐ machine-assisted, needs native review | | Step labels must fit one button line |
| `admin` | ~35 | ☐ machine-assisted, needs native review | | Admin V1 is English-first (doc 17 §10); Telugu kept for completeness |

## Glossary used (validate with users, doc 17 §10)

| English | Telugu used | Note |
|---|---|---|
| order | ఆర్డర్ | loanword users expect |
| delivery | డెలివరీ | loanword |
| restaurant | రెస్టారెంట్ | loanword |
| cart | కార్ట్ | loanword |
| payment / payout | చెల్లింపు | native word |
| code (OTP) | కోడ్ | loanword |
| address | చిరునామా | native word |
| vegetarian / non-vegetarian | శాకాహారం / మాంసాహారం | |
| try again | మళ్ళీ ప్రయత్నించండి | |

## Rules for new strings

1. Add the English key in `locales/en/<ns>.json` and the Telugu value in `locales/te/<ns>.json`
   in the same PR. `pnpm i18n:check` fails CI if a Telugu key is missing, empty, missing a
   plural form (`_one` / `_other`) or uses different `{{variables}}`.
2. Add a row (or update the count) above and mark it ☐ until a native speaker approves it.
