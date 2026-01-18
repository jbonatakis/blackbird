package agent

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"

	"github.com/jbonatakis/blackbird/internal/plan"
)

const SchemaVersion = 1

type RequestType string

const (
	RequestPlanGenerate RequestType = "plan_generate"
	RequestPlanRefine   RequestType = "plan_refine"
	RequestDepsInfer    RequestType = "deps_infer"
)

type RequestMetadata struct {
	Provider       string   `json:"provider,omitempty"`
	Model          string   `json:"model,omitempty"`
	MaxTokens      *int     `json:"maxTokens,omitempty"`
	Temperature    *float64 `json:"temperature,omitempty"`
	ResponseFormat string   `json:"responseFormat,omitempty"`
	JSONSchema     string   `json:"jsonSchema,omitempty"`
}

type Answer struct {
	ID    string `json:"id"`
	Value string `json:"value"`
}

type Request struct {
	SchemaVersion      int             `json:"schemaVersion"`
	Type               RequestType     `json:"type"`
	SystemPrompt       string          `json:"systemPrompt,omitempty"`
	ProjectDescription string          `json:"projectDescription,omitempty"`
	Constraints        []string        `json:"constraints,omitempty"`
	Granularity        string          `json:"granularity,omitempty"`
	ChangeRequest      string          `json:"changeRequest,omitempty"`
	Plan               *plan.WorkGraph `json:"plan,omitempty"`
	Answers            []Answer        `json:"answers,omitempty"`
	Metadata           RequestMetadata `json:"metadata,omitempty"`
}

type Question struct {
	ID      string   `json:"id"`
	Prompt  string   `json:"prompt"`
	Options []string `json:"options,omitempty"`
}

type PatchOpType string

const (
	PatchAdd       PatchOpType = "add"
	PatchUpdate    PatchOpType = "update"
	PatchDelete    PatchOpType = "delete"
	PatchMove      PatchOpType = "move"
	PatchSetDeps   PatchOpType = "set_deps"
	PatchAddDep    PatchOpType = "add_dep"
	PatchRemoveDep PatchOpType = "remove_dep"
)

type PatchOp struct {
	Op           PatchOpType       `json:"op"`
	ID           string            `json:"id,omitempty"`
	Item         *plan.WorkItem    `json:"item,omitempty"`
	ParentID     *string           `json:"parentId,omitempty"`
	Index        *int              `json:"index,omitempty"`
	Deps         []string          `json:"deps,omitempty"`
	DepID        string            `json:"depId,omitempty"`
	Rationale    string            `json:"rationale,omitempty"`
	DepRationale map[string]string `json:"depRationale,omitempty"`
}

type Response struct {
	SchemaVersion int             `json:"schemaVersion"`
	Type          RequestType     `json:"type"`
	Plan          *plan.WorkGraph `json:"plan,omitempty"`
	Patch         []PatchOp       `json:"patch,omitempty"`
	Questions     []Question      `json:"questions,omitempty"`
}

type ValidationError struct {
	Path    string
	Message string
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Path, e.Message)
}

func DecodeRequest(data []byte) (Request, error) {
	var req Request
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		return Request{}, err
	}
	if err := ensureEOF(dec); err != nil {
		return Request{}, err
	}
	return req, nil
}

func EncodeRequest(req Request) ([]byte, error) {
	return json.Marshal(req)
}

func DecodeResponse(data []byte) (Response, error) {
	var resp Response
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&resp); err != nil {
		return Response{}, err
	}
	if err := ensureEOF(dec); err != nil {
		return Response{}, err
	}
	return resp, nil
}

func EncodeResponse(resp Response) ([]byte, error) {
	return json.Marshal(resp)
}

func ValidateRequest(req Request) []ValidationError {
	var errs []ValidationError
	if req.SchemaVersion == 0 {
		errs = append(errs, ValidationError{Path: "$.schemaVersion", Message: "required"})
	} else if req.SchemaVersion != SchemaVersion {
		errs = append(errs, ValidationError{
			Path:    "$.schemaVersion",
			Message: fmt.Sprintf("unsupported schemaVersion %d (expected %d)", req.SchemaVersion, SchemaVersion),
		})
	}

	switch req.Type {
	case RequestPlanGenerate, RequestPlanRefine, RequestDepsInfer:
	default:
		errs = append(errs, ValidationError{Path: "$.type", Message: "invalid or missing request type"})
	}

	switch req.Type {
	case RequestPlanGenerate:
		if req.ProjectDescription == "" {
			errs = append(errs, ValidationError{Path: "$.projectDescription", Message: "required"})
		}
	case RequestPlanRefine:
		if req.ChangeRequest == "" {
			errs = append(errs, ValidationError{Path: "$.changeRequest", Message: "required"})
		}
		if req.Plan == nil {
			errs = append(errs, ValidationError{Path: "$.plan", Message: "required"})
		}
	case RequestDepsInfer:
		if req.Plan == nil {
			errs = append(errs, ValidationError{Path: "$.plan", Message: "required"})
		}
	}

	for i, ans := range req.Answers {
		path := fmt.Sprintf("$.answers[%d]", i)
		if ans.ID == "" {
			errs = append(errs, ValidationError{Path: path + ".id", Message: "required"})
		}
		if ans.Value == "" {
			errs = append(errs, ValidationError{Path: path + ".value", Message: "required"})
		}
	}

	return errs
}

func ValidateResponse(resp Response) []ValidationError {
	var errs []ValidationError
	if resp.SchemaVersion == 0 {
		errs = append(errs, ValidationError{Path: "$.schemaVersion", Message: "required"})
	} else if resp.SchemaVersion != SchemaVersion {
		errs = append(errs, ValidationError{
			Path:    "$.schemaVersion",
			Message: fmt.Sprintf("unsupported schemaVersion %d (expected %d)", resp.SchemaVersion, SchemaVersion),
		})
	}

	switch resp.Type {
	case RequestPlanGenerate, RequestPlanRefine, RequestDepsInfer:
	default:
		errs = append(errs, ValidationError{Path: "$.type", Message: "invalid or missing request type"})
	}

	if resp.Plan != nil && len(resp.Patch) != 0 {
		errs = append(errs, ValidationError{Path: "$", Message: "response must include either plan or patch, not both"})
	}
	if resp.Plan == nil && len(resp.Patch) == 0 && len(resp.Questions) == 0 {
		errs = append(errs, ValidationError{Path: "$", Message: "response must include plan, patch, or questions"})
	}

	if resp.Plan != nil {
		if perrs := plan.Validate(*resp.Plan); len(perrs) != 0 {
			for _, pe := range perrs {
				errs = append(errs, ValidationError{Path: "$.plan" + pe.Path[1:], Message: pe.Message})
			}
		}
	}

	if len(resp.Patch) != 0 {
		errs = append(errs, validatePatchOps(resp.Patch)...)
	}

	for i, q := range resp.Questions {
		path := fmt.Sprintf("$.questions[%d]", i)
		if q.ID == "" {
			errs = append(errs, ValidationError{Path: path + ".id", Message: "required"})
		}
		if q.Prompt == "" {
			errs = append(errs, ValidationError{Path: path + ".prompt", Message: "required"})
		}
		for j, opt := range q.Options {
			if opt == "" {
				errs = append(errs, ValidationError{
					Path:    fmt.Sprintf("%s.options[%d]", path, j),
					Message: "option must be non-empty",
				})
			}
		}
	}

	return errs
}

func validatePatchOps(ops []PatchOp) []ValidationError {
	var errs []ValidationError
	for i, op := range ops {
		path := fmt.Sprintf("$.patch[%d]", i)
		switch op.Op {
		case PatchAdd:
			if op.Item == nil {
				errs = append(errs, ValidationError{Path: path + ".item", Message: "required for add"})
				break
			}
			errs = append(errs, validateWorkItem(path+".item", *op.Item)...)
		case PatchUpdate:
			if op.Item == nil {
				errs = append(errs, ValidationError{Path: path + ".item", Message: "required for update"})
				break
			}
			if op.ID != "" && op.Item.ID != "" && op.ID != op.Item.ID {
				errs = append(errs, ValidationError{Path: path + ".id", Message: "must match item.id when both provided"})
			}
			errs = append(errs, validateWorkItem(path+".item", *op.Item)...)
		case PatchDelete:
			if op.ID == "" {
				errs = append(errs, ValidationError{Path: path + ".id", Message: "required for delete"})
			}
		case PatchMove:
			if op.ID == "" {
				errs = append(errs, ValidationError{Path: path + ".id", Message: "required for move"})
			}
			if op.Index != nil && *op.Index < 0 {
				errs = append(errs, ValidationError{Path: path + ".index", Message: "must be >= 0"})
			}
		case PatchSetDeps:
			if op.ID == "" {
				errs = append(errs, ValidationError{Path: path + ".id", Message: "required for set_deps"})
			}
			if op.Deps == nil {
				errs = append(errs, ValidationError{Path: path + ".deps", Message: "required (use [] if none)"})
			}
			errs = append(errs, validateDepRationale(path, op.Deps, op.DepRationale)...)
		case PatchAddDep:
			if op.ID == "" {
				errs = append(errs, ValidationError{Path: path + ".id", Message: "required for add_dep"})
			}
			if op.DepID == "" {
				errs = append(errs, ValidationError{Path: path + ".depId", Message: "required for add_dep"})
			}
			if op.DepRationale != nil {
				if _, ok := op.DepRationale[op.DepID]; !ok {
					errs = append(errs, ValidationError{
						Path:    path + ".depRationale",
						Message: "depRationale must include depId key when provided",
					})
				}
			}
		case PatchRemoveDep:
			if op.ID == "" {
				errs = append(errs, ValidationError{Path: path + ".id", Message: "required for remove_dep"})
			}
			if op.DepID == "" {
				errs = append(errs, ValidationError{Path: path + ".depId", Message: "required for remove_dep"})
			}
		default:
			errs = append(errs, ValidationError{Path: path + ".op", Message: "invalid patch op"})
		}
	}
	return errs
}

func validateWorkItem(path string, it plan.WorkItem) []ValidationError {
	var errs []ValidationError
	if it.ID == "" {
		errs = append(errs, ValidationError{Path: path + ".id", Message: "required"})
	}
	if it.Title == "" {
		errs = append(errs, ValidationError{Path: path + ".title", Message: "required"})
	}
	if it.AcceptanceCriteria == nil {
		errs = append(errs, ValidationError{Path: path + ".acceptanceCriteria", Message: "required (use [] if none)"})
	}
	if it.ChildIDs == nil {
		errs = append(errs, ValidationError{Path: path + ".childIds", Message: "required (use [] if none)"})
	}
	if it.Deps == nil {
		errs = append(errs, ValidationError{Path: path + ".deps", Message: "required (use [] if none)"})
	}
	if !isValidStatus(it.Status) {
		errs = append(errs, ValidationError{Path: path + ".status", Message: fmt.Sprintf("invalid status %q", it.Status)})
	}
	if it.CreatedAt.IsZero() {
		errs = append(errs, ValidationError{Path: path + ".createdAt", Message: "required (RFC3339 timestamp)"})
	}
	if it.UpdatedAt.IsZero() {
		errs = append(errs, ValidationError{Path: path + ".updatedAt", Message: "required (RFC3339 timestamp)"})
	}
	if !it.CreatedAt.IsZero() && !it.UpdatedAt.IsZero() && it.UpdatedAt.Before(it.CreatedAt) {
		errs = append(errs, ValidationError{Path: path + ".updatedAt", Message: "must be >= createdAt"})
	}
	errs = append(errs, validateDepRationale(path, it.Deps, it.DepRationale)...)
	return errs
}

func validateDepRationale(path string, deps []string, rationale map[string]string) []ValidationError {
	var errs []ValidationError
	if len(rationale) == 0 {
		return errs
	}
	depSet := map[string]bool{}
	for _, depID := range deps {
		depSet[depID] = true
	}
	for depID, reason := range rationale {
		if depID == "" {
			errs = append(errs, ValidationError{Path: path + ".depRationale", Message: "depRationale keys must be non-empty"})
			continue
		}
		if !depSet[depID] {
			errs = append(errs, ValidationError{Path: path + ".depRationale", Message: fmt.Sprintf("depRationale key %q must appear in deps", depID)})
		}
		if reason == "" {
			errs = append(errs, ValidationError{Path: path + ".depRationale", Message: fmt.Sprintf("depRationale[%q] must be non-empty", depID)})
		}
	}
	return errs
}

func isValidStatus(s plan.Status) bool {
	switch s {
	case plan.StatusTodo, plan.StatusInProgress, plan.StatusBlocked, plan.StatusDone, plan.StatusSkipped:
		return true
	default:
		return false
	}
}

func ensureEOF(dec *json.Decoder) error {
	var extra any
	if err := dec.Decode(&extra); err != nil {
		if err == io.EOF {
			return nil
		}
		return err
	}
	return fmt.Errorf("unexpected trailing JSON tokens")
}
