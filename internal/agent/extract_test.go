package agent

import "testing"

func TestExtractJSONFullOutput(t *testing.T) {
	output := `{"schemaVersion":1,"type":"plan_generate","questions":[{"id":"q1","prompt":"What?"}]}`
	got, err := ExtractJSON(output)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got != output {
		t.Fatalf("expected %q, got %q", output, got)
	}
}

func TestExtractJSONFencedBlock(t *testing.T) {
	output := "note\n```json\n{\n  \"schemaVersion\": 1,\n  \"type\": \"plan_generate\",\n  \"questions\": [{\"id\":\"q1\",\"prompt\":\"Q\"}]\n}\n```\nmore"
	got, err := ExtractJSON(output)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got == "" || got[0] != '{' {
		t.Fatalf("expected JSON object, got %q", got)
	}
}

func TestExtractJSONMultipleObjects(t *testing.T) {
	output := "{}\n{}"
	_, err := ExtractJSON(output)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if err != ErrMultipleJSONFound {
		t.Fatalf("expected ErrMultipleJSONFound, got %v", err)
	}
}

func TestExtractJSONMissing(t *testing.T) {
	output := "nothing here"
	_, err := ExtractJSON(output)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if err != ErrNoJSONFound {
		t.Fatalf("expected ErrNoJSONFound, got %v", err)
	}
}
