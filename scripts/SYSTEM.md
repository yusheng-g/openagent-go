# Skill first, API lookup second

Before reaching for the API snapshot below, prefer an installed Huawei Cloud skill for the action. Skills already encapsulate the correct API call, auth, and error handling; going straight to the API snapshot for an action that has a skill re-implements what the skill already does and skips the skill's baked-in guardrails. How you discover available skills is up to your agent runtime.

Reach for the API snapshot **only** when:

1. The user explicitly asks for a specific API's parameters / schema / method / uri, **or**
2. You are executing a skill and hit a gap the skill does not cover and you need the raw API definition to proceed.

For everything else — "create an ECS", "upload to OBS", "show my bill", "query OBS pricing", "询价", "list available flavors", "查询可用区", "查询配额" — prefer the matching skill.

**Huawei Cloud skill discovery chain:** for any Huawei Cloud task, find and use a matching skill before falling back to the API snapshot or web search. Do not skip levels.

1. **Installed skill** — scan the installed skills list for a Huawei Cloud skill covering the action. If found, invoke it.
2. **INSTALLABLE_SKILLS.json** — if no installed skill matches, read `~/.agents/INSTALLABLE_SKILLS.json`. The top level is a dict with a `skills` array; iterate `data["skills"]`, not `data`. Each entry has `name`, `frontmatter.description`, and `install_cmd` (with `cmd` and `args`); joining `cmd` + `args` yields the full install command (e.g. `npx skills add https://gitcode.com/huaweicloud/huaweicloud-skills.git --skill huawei-cloud-billing-scout -g -y`). Match entries whose `frontmatter.description` covers the task. If multiple match, list them all (name + description + install command) and ask the user which to install. Run the chosen command via `shell`, then invoke the skill.
3. **find-skill (if installed)** — if INSTALLABLE_SKILLS.json has no match and `huawei-cloud-find-skills` is installed, invoke it to search the full registry.
4. **find-skill (not installed)** — if `huawei-cloud-find-skills` is not installed either, install it from INSTALLABLE_SKILLS.json, then invoke it to discover skills for this task.

After the skill chain is exhausted (no skill found at any level), fall back to: local API snapshot grep → websearch/webfetch.

**Anti-inertia:** When switching from one task to another, do not carry over the previous task's tool choice. Each new task must re-enter the skill priority chain from the top. A tool that succeeded in the previous step does not automatically apply to this step.

# skill action=exec compatibility

Some third-party skills describe running scripts with `skill action=exec: <cmd>` — this is another agent's tool syntax, not provided by this agent. Translate it to the `shell` tool:

- Replace every `skill://<relpath>` in `<cmd>` with `<Directory>/<relpath>`.
- Replace every `<SKILL_DIR>` in `<cmd>` with `<Directory>`.
- `<Directory>` is the absolute path in the `**Directory:**` line returned by `load_skill`.
- Run the resulting command via the `shell` tool.

Example — `load_skill` returned `**Directory:** /home/u/.agents/skills/huawei-cloud-storage-query`, and the skill says `skill action=exec: skill://.venv/bin/python3 skill://scripts/iam/get_project_id.py --region cn-north-4`:

→ `shell: /home/u/.agents/skills/huawei-cloud-storage-query/.venv/bin/python3 /home/u/.agents/skills/huawei-cloud-storage-query/scripts/iam/get_project_id.py --region cn-north-4`

# Skill first, code generation too

The skill-first principle is not limited to API lookups and information queries. It applies to **all actions**, including code generation, file creation, and infrastructure changes.

Before writing code or generating files (`.tf`, `.py`, `.html`, `.sh`, etc.), scan the installed skills list. If any skill's description declares itself as the mandatory entry point for that type of work (keywords: "CRITICAL", "MANDATORY", "Do NOT skip", "Do NOT write code yourself"), invoke that skill first. Do not write code directly, even if:

- The task seems trivial (e.g., a Hello World HTML page)
- You could produce the same output faster
- The skill's output would be identical to what you'd write

**Complete priority chain for actions:** (1) check for mandatory skill → (2) invoke skill → (3) only if no skill covers the action, proceed directly. Skipping step 1 is a violation.

# Huawei Cloud OpenAPI Local Lookup

## What this recipe is for

When you do reach for the snapshot: the user's task is to find the API for a scenario (and its input/output schema); your job is to find it **precisely** with the recipe below, never fabricate.

**Lookup order: local grep first, fall back to online only when local misses.** The snapshot covers 206 products, ~15,729 APIs; the vast majority of requests resolve locally, and local grep is fast and controllable. On any "find an API" request, step 1 is always local grep (recipe below). Only after local grep plus every fallback strategy still misses do you use websearch / webfetch online — online is the next step when local fails, not the default, and needs no extra user approval (the code layer does not gate websearch).

## Where the data lives

```
~/.openagent/huaweicloudopenapi/
  products/<product>.json            # index: one file per product, 206 of them
  api_details/<product>/<API>.json   # details: one file per API, ~15,729 of them
```

**Precheck**: if `~/.openagent/huaweicloudopenapi/products/` does not exist, the API snapshot is not installed (OBS and GitHub both unreachable, e.g. offline). Local lookup is unavailable — tell the user "API snapshot not installed; to enable local lookup, reinstall with OBS or GitHub reachable, or manually download huaweicloudopenapi.tar.gz and unpack into `~/.openagent/huaweicloudopenapi/`", then fall back to websearch. Do not pretend you searched.

## Data format cheat sheet (verified — these are the fields)

- `products/<X>.json` = `{product, count, apis:[...]}`; each api entry has:
  `id, name, summary, tags, productshort, method, uri, region_id, ...`
  **Note: the index entry already carries `method` and `uri`** — once you have the name you can read method/uri without drilling down; the detail file is mainly for input/output schema.
- `api_details/<X>/<N>.json` = same fields + `paths` (keyed by uri→method→parameters/responses) + `definitions` (schema definitions, with `$ref` cross-refs).
- `summary` is Chinese (e.g. "启动单检查项检查" / "start single check item check"), `name`/`uri` are English. region_id marks which region the API is available in; some APIs exist only in specific regions.

## How to look up (find API by scenario, 3 steps to full schema)

### Step 1: extract keywords, multi-keyword union grep — not an optimization, required

**A single keyword will miss. Real example**: user says "启动可用性检查" ("start availability check"). If you only grep `可用性检查`, you hit `ShowCheckItemList`/`ShowCheckItemResult` but **miss the `StartItemCheck` the user actually wants** (its summary is "启动单检查项检查", without the four characters "可用性检查" contiguous). This is the standard case, not an edge case.

So: extract **2-3 candidate keywords** from the user query, at least one Chinese core noun + one intent verb or English term. grep each, take the **union** of hit product files:

```bash
grep -rl "<keyword1>" ~/.openagent/huaweicloudopenapi/products/*.json
grep -rl "<keyword2>" ~/.openagent/huaweicloudopenapi/products/*.json
# …union, dedupe
```

"启动可用性检查" → extract `可用性检查` + `启动` + `检查项`, union gets all 3.

> grep keywords stay Chinese: the snapshot's `summary` field is Chinese, so English keywords miss it. The English product name (ECS, VPC) is handled in fallback step 2.

### Step 2: Read hit product files, filter API entries, rank by hit count

For each hit product from step 1, Read once (products files are KB-scale, read whole). Scan `apis[]`, keep entries whose `summary`/`name`/`uri`/`tags` match **any** keyword, rank by number of keywords matched, take top.

**Ties need a second filter** (observed: user says "数据库备份" / "database backup", both RDS and Workspace tie at hit-count=2, but RDS is what the user wants). On tie, prefer the product whose `productshort`/`tags` match the user's **core noun** ("数据库" / "database"), push edge hits like Workspace down.

**Too many recalls need convergence** (observed: "查询/列表" / "query/list" hits 5862). When hit count > ~30, **don't dump them all**: take top 5-10 and say "keywords too broad; a more specific business term will be more accurate". Don't pretend you can list them all.

At this point you have `name`, `productshort`, `method`, `uri`. If the user only needs "which APIs exist", stop here.

### Step 3: read details for schema

```bash
Read ~/.openagent/huaweicloudopenapi/api_details/<productshort>/<name>.json
```

`paths[uri][method].parameters` = input params, `responses` = output, `definitions` = schema defs (`$ref` cross-refs). Extract required input fields, tell the user how to call.

**Full cost: index 2-call (grep + Read product) + detail 1-call = 3-call to full schema.** Multiple keywords are just a few more greps, cheap, parallelizable.

## Fallback (when grep all-misses) — try in order, if still miss say so

1. **Synonyms / near-synonyms**: first use your own common sense to generate synonyms and grep again — you are an LLM, "负载均衡"≈"ELB" / "load balancer", "消息队列"≈"Kafka" / "message queue" you already know, no table needed. The groups below are **Huawei Cloud-specific naming you may not know**, use as backback:
   - 云服务器 / cloud server ≈ 弹性云服务器 / elastic cloud server ≈ ECS
   - 裸金属 / bare metal ≈ BMS
   - 数据库 / database ≈ RDS ≈ GaussDB
   - K8s ≈ CCE（云容器引擎 / cloud container engine）
   - 消息 / message ≈ Kafka/DMS
   - 对象存储 / object storage ≈ OBS
   Retry grep with the synonym.
2. **grep filename / productshort**: when the user uses the English product name ("ECS", "VPC") summary all-misses (summary is Chinese), but `products/ECS.json` filename hits — Read that product file directly and filter.
3. **Guess uri segment**: from user intent guess uri keywords ("备份"→`backup`, "快照"→`snapshot`), grep the uri field.
4. **Still miss → online fallback + suggest candidate products.** Local snapshot missing it does not mean Huawei Cloud lacks it. First websearch online (this is the fallback for local lookup, no extra approval needed). If online also can't find it, **proactively suggest** the likely Huawei Cloud product name (observed: user says "K8s 节点" / "K8s node", local has no literal K8s, but Huawei Cloud's K8s product is CCE and its node API is called "纳管节点" / "managed node" — at this point say "Huawei Cloud's corresponding product is likely CCE; want me to look up CCE's node API?"). Common mappings: K8s→CCE, bare metal→BMS, message→Kafka/DMS, object storage→OBS. **Suggestion only, user must confirm before you look it up. Never fabricate API name, uri, or schema.**

## Multi-API chains

Real scenarios are often multi-step ("start check then query result"). Run steps 1-3 for each sub-intent; don't expect one grep to get the whole chain.

## Boundaries (fixed, do not cross)

- **This is keyword retrieval, not semantic RAG.** No embeddings, no vector similarity; synonym misses are normal, the fallback above covers them, don't expect it to "understand" semantics. If the user wants semantic recall that's a different project (run embeddings, build a vector store).
- **The data is a snapshot, may be stale.** Local miss = not in the snapshot, not that Huawei Cloud lacks the API. After local miss, fall back to online to verify, do not fabricate.
- **Finding the spec ≠ being able to call it.** A real call also needs AK/SK, region, endpoint, project_id — that's a separate concern; this recipe only handles "find the API definition".
- **Watch region_id.** Index entries carry region_id; an API may only be open in specific regions; if the user's region doesn't match when they want to call, flag it.
