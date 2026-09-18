# Wave 14 route receipts

Bounded task-scoped lookup from each rollout's own `session_meta.id`, exact parent, canonical
agent path, and latest active `turn_context`. Provider attestation is not available or required.

| Lane | Session id | Parent id | Phase | Active runtime route | Evidence class |
|---|---|---|---|---|---|
| A | `01a0b4dc-3b4e-7820-8eed-20aa3569d021` | `01a0b4d9-82c5-7882-b127-d74c6645dc83` | active implementation packet | `gpt-6-astra` / `medium` | verified runtime configuration |
| B | `01a0b4e1-0587-7e12-8e8e-4227337b6e17` | `01a0b4d9-82c5-7882-b127-d74c6645dc83` | active implementation | `gpt-5.6-luna` / `max` | verified runtime configuration |
| C | `01a0b4dc-9301-7d03-910c-ec6c924b4f2a` | `01a0b4d9-82c5-7882-b127-d74c6645dc83` | active implementation | `gpt-5.6-terra` / `high` | verified runtime configuration |
| D | `01a0b4dc-e0ee-7dc0-83eb-c0574bd1ce2c` | `01a0b4d9-82c5-7882-b127-d74c6645dc83` | active implementation | `gpt-5.6-luna` / `max` | verified runtime configuration |
| E | `01a0b4dd-2dad-7370-8282-6109775cfc07` | `01a0b4d9-82c5-7882-b127-d74c6645dc83` | active implementation | `gpt-5.6-terra` / `high` | verified runtime configuration |
| G | `01a0b4dd-87ff-7193-9a49-2385f47e3f10` | `01a0b4d9-82c5-7882-b127-d74c6645dc83` | active design and migration test | `gpt-5.6-sol` / `high` | verified runtime configuration |
| F | `01a0b4f8-919e-7b43-b6f8-38a3144365de` | `01a0b4d9-82c5-7882-b127-d74c6645dc83` | completed fresh adversarial review after root repair | `gpt-6-astra` / `medium` | verified runtime configuration |
| R1 | `01a0b4f8-b3f0-7891-b449-b5c6c905a7e0` | `01a0b4d9-82c5-7882-b127-d74c6645dc83` | completed mapping triage | `gpt-5.6-luna` / `medium` | verified runtime configuration |
