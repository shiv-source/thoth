package tools

import (
	"context"
	"reflect"
	"strings"
	"testing"
)

// toolStub is a minimal Tool implementation for registry tests.
type toolStub struct {
	name string
}

func (t toolStub) Name() string        { return t.name }
func (t toolStub) Description() string { return "stub " + t.name }
func (t toolStub) Schema() map[string]any {
	return map[string]any{"type": "object", "properties": map[string]any{}}
}
func (t toolStub) Run(ctx context.Context, args map[string]any) (string, error) {
	return t.name, nil
}

func TestRegistryRegisterGetList(t *testing.T) {
	r := NewRegistry()
	for _, name := range []string{"write_file", "read_file", "list"} {
		if err := r.Register(toolStub{name: name}); err != nil {
			t.Fatalf("Register(%s): %v", name, err)
		}
	}

	got := r.List()
	var names []string
	for _, tl := range got {
		names = append(names, tl.Name())
	}
	want := []string{"list", "read_file", "write_file"}
	if !reflect.DeepEqual(names, want) {
		t.Fatalf("List order = %v, want %v", names, want)
	}

	tl, err := r.Get("read_file")
	if err != nil {
		t.Fatalf("Get(read_file): %v", err)
	}
	if tl.Name() != "read_file" {
		t.Fatalf("Get returned %q", tl.Name())
	}
}

func TestRegistryDuplicateErrors(t *testing.T) {
	r := NewRegistry()
	if err := r.Register(toolStub{name: "read_file"}); err != nil {
		t.Fatal(err)
	}
	err := r.Register(toolStub{name: "read_file"})
	if err == nil {
		t.Fatal("registering a duplicate name succeeded, want error")
	}
	if !strings.Contains(err.Error(), "already registered") {
		t.Fatalf("duplicate error = %q", err)
	}
}

func TestRegistryRejectsNilAndEmptyName(t *testing.T) {
	r := NewRegistry()
	if err := r.Register(nil); err == nil {
		t.Fatal("registering nil succeeded, want error")
	}
	if err := r.Register(toolStub{name: ""}); err == nil {
		t.Fatal("registering an empty name succeeded, want error")
	}
}

func TestRegistryGetUnknown(t *testing.T) {
	r := NewRegistry()
	_, err := r.Get("grep")
	if err == nil {
		t.Fatal("Get(unknown) succeeded, want error")
	}
	if !strings.Contains(err.Error(), "unknown tool") {
		t.Fatalf("unknown error = %q", err)
	}
}

func TestRegistryEmptyList(t *testing.T) {
	if got := NewRegistry().List(); len(got) != 0 {
		t.Fatalf("List on empty registry = %v, want none", got)
	}
}

func TestToolSchemas(t *testing.T) {
	fs, err := NewOSFS(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	tools := []Tool{
		NewReadFile(fs, 0),
		NewWriteFile(fs),
		NewList(fs),
		NewSearch(func(context.Context, string, int) ([]Result, error) { return nil, nil }, 0),
		NewGetTime(nil),
		NewEditFile(fs),
		NewAppendFile(fs),
		NewRenameFile(fs),
		NewDeleteFile(fs),
		NewGrep(fs, 0),
	}
	for _, tl := range tools {
		name := tl.Name()
		if name == "" || tl.Description() == "" {
			t.Fatalf("tool %q missing name or description", name)
		}
		schema := tl.Schema()
		if schema["type"] != "object" {
			t.Fatalf("%s schema type = %v, want object", name, schema["type"])
		}
	}
	names := map[string]bool{}
	for _, tl := range tools {
		names[tl.Name()] = true
	}
	for _, want := range []string{
		"read_file", "write_file", "list", "search", "get_time", "edit_file",
		"append_file", "rename_file", "delete_file", "grep",
	} {
		if !names[want] {
			t.Fatalf("missing tool %q", want)
		}
	}
	if _, ok := tools[0].Schema()["required"].([]string); !ok {
		t.Fatal("read_file schema missing required")
	}
	if _, ok := tools[1].Schema()["required"].([]string); !ok {
		t.Fatal("write_file schema missing required")
	}
	if _, ok := tools[3].Schema()["required"].([]string); !ok {
		t.Fatal("search schema missing required")
	}
}

func TestToolNamesAreStable(t *testing.T) {
	// Model prompts and any persisted tool references depend on these names.
	fs, err := NewOSFS(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]bool{
		NewReadFile(fs, 0).Name(): true,
		NewWriteFile(fs).Name():   true,
		NewList(fs).Name():        true,
		NewSearch(func(context.Context, string, int) ([]Result, error) { return nil, nil }, 0).Name(): true,
	}
	for _, want := range []string{"read_file", "write_file", "list", "search"} {
		if !got[want] {
			t.Fatalf("missing stable tool name %q", want)
		}
	}
}

func TestStringArg(t *testing.T) {
	if _, err := StringArg(map[string]any{}, "path"); err == nil {
		t.Fatal("missing arg succeeded, want error")
	}
	if _, err := StringArg(map[string]any{"path": 42}, "path"); err == nil {
		t.Fatal("non-string arg succeeded, want error")
	}
	got, err := StringArg(map[string]any{"path": "x"}, "path")
	if err != nil || got != "x" {
		t.Fatalf("StringArg = %q, %v", got, err)
	}
	def, err := StringArgDefault(map[string]any{}, "path", ".")
	if err != nil || def != "." {
		t.Fatalf("StringArgDefault = %q, %v", def, err)
	}
}

func TestSliceAndIntArgs(t *testing.T) {
	tags, err := StringSliceArg(map[string]any{"tags": []any{"a", "b"}}, "tags")
	if err != nil || !reflect.DeepEqual(tags, []string{"a", "b"}) {
		t.Fatalf("StringSliceArg []any = %v, %v", tags, err)
	}
	tags, err = StringSliceArg(map[string]any{"tags": []string{"a"}}, "tags")
	if err != nil || !reflect.DeepEqual(tags, []string{"a"}) {
		t.Fatalf("StringSliceArg []string = %v, %v", tags, err)
	}
	if tags, err := StringSliceArg(map[string]any{}, "tags"); err != nil || tags != nil {
		t.Fatalf("StringSliceArg missing = %v, %v", tags, err)
	}
	if _, err := StringSliceArg(map[string]any{"tags": []any{1}}, "tags"); err == nil {
		t.Fatal("StringSliceArg with non-string element succeeded")
	}
	if _, err := StringSliceArg(map[string]any{"tags": 42}, "tags"); err == nil {
		t.Fatal("StringSliceArg with non-slice succeeded")
	}

	n, err := IntArg(map[string]any{"limit": 5.0}, "limit")
	if err != nil || n != 5 {
		t.Fatalf("IntArg float = %d, %v", n, err)
	}
	n, err = IntArgDefault(map[string]any{}, "limit", 7)
	if err != nil || n != 7 {
		t.Fatalf("IntArgDefault = %d, %v", n, err)
	}
	if _, err := IntArg(map[string]any{}, "limit"); err == nil {
		t.Fatal("IntArg missing succeeded")
	}
	if _, err := IntArg(map[string]any{"limit": "x"}, "limit"); err == nil {
		t.Fatal("IntArg non-number succeeded")
	}
}
