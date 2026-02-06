package agent

import "strings"

const (
	MaxPlanQuestionRounds    = 2
	MaxPlanGenerateRevisions = 1
)

func DefaultPlanJSONSchema() string {
	return strings.TrimSpace(`{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "required": ["schemaVersion", "type"],
  "properties": {
    "schemaVersion": { "type": "integer" },
    "type": { "type": "string", "enum": ["plan_generate", "plan_refine", "deps_infer"] },
    "plan": { "$ref": "#/definitions/workGraph" },
    "patch": { "type": "array", "items": { "$ref": "#/definitions/patchOp" } },
    "questions": { "type": "array", "items": { "$ref": "#/definitions/question" } }
  },
  "oneOf": [
    { "required": ["plan"] },
    { "required": ["patch"] },
    { "required": ["questions"] }
  ],
  "definitions": {
    "workGraph": {
      "type": "object",
      "required": ["schemaVersion", "items"],
      "properties": {
        "schemaVersion": { "type": "integer" },
        "items": {
          "type": "object",
          "additionalProperties": { "$ref": "#/definitions/workItem" }
        }
      }
    },
    "workItem": {
      "type": "object",
      "required": [
        "id", "title", "description", "acceptanceCriteria", "prompt",
        "parentId", "childIds", "deps", "status", "createdAt", "updatedAt"
      ],
      "properties": {
        "id": { "type": "string" },
        "title": { "type": "string" },
        "description": { "type": "string" },
        "acceptanceCriteria": { "type": "array", "items": { "type": "string" } },
        "prompt": { "type": "string" },
        "parentId": { "type": ["string", "null"] },
        "childIds": { "type": "array", "items": { "type": "string" } },
        "deps": { "type": "array", "items": { "type": "string" } },
        "status": { "type": "string", "enum": ["todo", "in_progress", "blocked", "done", "skipped"] },
        "createdAt": { "type": "string", "format": "date-time" },
        "updatedAt": { "type": "string", "format": "date-time" },
        "notes": { "type": "string" },
        "depRationale": { "type": "object", "additionalProperties": { "type": "string" } }
      }
    },
    "patchOp": {
      "type": "object",
      "required": ["op"],
      "properties": {
        "op": { "type": "string", "enum": ["add", "update", "delete", "move", "set_deps", "add_dep", "remove_dep"] },
        "id": { "type": "string" },
        "item": { "$ref": "#/definitions/workItem" },
        "parentId": { "type": ["string", "null"] },
        "index": { "type": "integer", "minimum": 0 },
        "deps": { "type": "array", "items": { "type": "string" } },
        "depId": { "type": "string" },
        "rationale": { "type": "string" },
        "depRationale": { "type": "object", "additionalProperties": { "type": "string" } }
      }
    },
    "question": {
      "type": "object",
      "required": ["id", "prompt"],
      "properties": {
        "id": { "type": "string" },
        "prompt": { "type": "string" },
        "options": { "type": "array", "items": { "type": "string" } }
      }
    }
  }
}`)
}

func DefaultPlanSystemPrompt() string {
	return strings.TrimSpace("You are the planning agent for blackbird.\n\n" +
		"Return exactly one JSON object on stdout (or a single fenced ```json block).\n" +
		"Do not include any text outside the JSON.\n\n" +
		"Response shape:\n" +
		"- Include schemaVersion and type.\n" +
		"- Include exactly one of: plan, patch, or questions.\n\n" +
		"How to use request inputs:\n" +
		"- projectDescription: primary product and scope signal.\n" +
		"- constraints: hard requirements to preserve.\n" +
		"- granularity: requested task size/detail level.\n\n" +
		"Granularity guidance:\n" +
		"- If granularity is empty, use balanced granularity: each leaf task should be a focused, independently executable coding unit.\n" +
		"- If granularity requests coarse/high-level/fewer tasks, group related work into larger leaf tasks.\n" +
		"- If granularity requests fine/detailed/more tasks, split work into smaller leaf tasks with narrow scope.\n" +
		"- Keep decomposition proportional to project scope; do not force tiny tasks for a small project or giant tasks for a large project.\n\n" +
		"Plan requirements:\n" +
		"- Plan must conform to the WorkGraph schema.\n" +
		"- Every WorkItem must include: id, title, description, acceptanceCriteria, prompt, parentId, childIds, deps, status, createdAt, updatedAt.\n" +
		"- Use stable, unique IDs and keep parent/child relationships consistent in both directions.\n" +
		"- Dependencies must reference existing IDs and must not create cycles.\n" +
		"- Avoid meta tasks like \"design the app\" or \"plan the work\" unless explicitly requested.\n" +
		"- Top-level items should be meaningful deliverables, not a generic \"root\" placeholder.\n" +
		"- For new work, default status to todo unless the user explicitly requests otherwise.\n" +
		"- Use only schema-valid statuses: todo, in_progress, blocked, done, skipped.\n\n" +
		"Plan quality heuristics:\n" +
		"- Leaf tasks should be focused units that are typically completable in one agent run (roughly 30-180 minutes of work).\n" +
		"- Each leaf task should produce a concrete artifact (code, tests, migration, configuration, or documentation).\n" +
		"- Acceptance criteria should be objective and verifiable (specific checks over vague outcomes).\n" +
		"- Minimize dependency edges; add a dependency only when execution order is required.\n" +
		"- Keep hierarchy intentional: parents are deliverables, children are executable implementation steps.\n\n" +
		"Task detail standards (especially for leaf tasks):\n" +
		"- Description should explain intent, in-scope work, and critical boundaries/constraints; avoid one-line placeholders.\n" +
		"- Acceptance criteria should usually include multiple concrete checks (commonly 4-8 for non-trivial tasks), including test/validation expectations when applicable.\n" +
		"- Prompt should be execution-oriented: specify what to implement, where to implement it (files/components when known), important constraints, and how to verify completion.\n" +
		"- Avoid vague wording like \"improve\", \"handle\", or \"fix\" without defining expected behavior/outcome.\n" +
		"- If project context is insufficient to write actionable detail, ask clarification questions instead of inventing specifics.\n\n" +
		"Patch requirements:\n" +
		"- Use only ops: add, update, delete, move, set_deps, add_dep, remove_dep.\n" +
		"- Include required fields for each op.\n" +
		"- Keep references valid; do not introduce cycles.\n" +
		"- Preserve existing structure/status unless needed for the requested change.\n\n" +
		"Questions:\n" +
		"- If key details are ambiguous and materially affect plan shape, respond with questions only (no plan/patch).\n" +
		"- Each question must include id and prompt; options are optional.\n")
}
