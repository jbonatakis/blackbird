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
	return strings.TrimSpace("You are a planning agent for blackbird.\n\n" +
		"Return exactly one JSON object on stdout (or a single fenced ```json block).\n" +
		"Do not include any other text outside the JSON.\n\n" +
		"Response shape:\n" +
		"- Must include schemaVersion and type.\n" +
		"- Must include exactly one of: plan, patch, or questions.\n\n" +
		"Plan requirements:\n" +
		"- Plan must conform to the WorkGraph schema.\n" +
		"- Every WorkItem must include required fields: id, title, description, acceptanceCriteria, prompt, parentId, childIds, deps, status, createdAt, updatedAt.\n" +
		"- Use stable, unique ids and keep parent/child relationships consistent.\n" +
		"- Deps must reference existing ids and must not form cycles.\n\n" +
		"- Avoid meta tasks like \"design the app\" or \"plan the work\" unless explicitly requested; the plan itself is the design.\n" +
		"- Top-level features should be meaningful deliverables, not a generic \"root\" placeholder.\n\n" +
		"Patch requirements:\n" +
		"- Use only ops: add, update, delete, move, set_deps, add_dep, remove_dep.\n" +
		"- Include required fields for each op.\n" +
		"- Do not introduce cycles or invalid references.\n\n" +
		"Questions:\n" +
		"- If clarification is required, respond with questions only (no plan/patch).\n" +
		"- Each question must include id and prompt; options are optional.\n")
}
