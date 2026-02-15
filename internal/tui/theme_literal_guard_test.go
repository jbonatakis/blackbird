package tui

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
)

func TestNoRawLipglossColorLiteralsOutsideThemeFiles(t *testing.T) {
	t.Parallel()

	allowedThemeFiles := map[string]struct{}{
		"theme.go":          {},
		"theme_registry.go": {},
		"theme_styles.go":   {},
	}

	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller(0) failed")
	}
	tuiDir := filepath.Dir(thisFile)

	entries, err := os.ReadDir(tuiDir)
	if err != nil {
		t.Fatalf("ReadDir(%q): %v", tuiDir, err)
	}

	fileSet := token.NewFileSet()
	var violations []string
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || filepath.Ext(name) != ".go" || strings.HasSuffix(name, "_test.go") {
			continue
		}

		path := filepath.Join(tuiDir, name)
		parsed, err := parser.ParseFile(fileSet, path, nil, 0)
		if err != nil {
			t.Fatalf("ParseFile(%q): %v", path, err)
		}

		_, allowed := allowedThemeFiles[name]
		ast.Inspect(parsed, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok || len(call.Args) != 1 {
				return true
			}

			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || sel.Sel.Name != "Color" {
				return true
			}
			pkgIdent, ok := sel.X.(*ast.Ident)
			if !ok || pkgIdent.Name != "lipgloss" {
				return true
			}
			arg, ok := call.Args[0].(*ast.BasicLit)
			if !ok || arg.Kind != token.STRING {
				return true
			}

			if !allowed {
				position := fileSet.Position(arg.Pos())
				violations = append(violations, fmt.Sprintf("%s:%d", name, position.Line))
			}
			return true
		})
	}

	if len(violations) == 0 {
		return
	}

	allowedList := make([]string, 0, len(allowedThemeFiles))
	for name := range allowedThemeFiles {
		allowedList = append(allowedList, name)
	}
	sort.Strings(allowedList)
	sort.Strings(violations)
	t.Fatalf(
		"raw lipgloss color-call literals detected outside allowed theme files (%s):\n%s",
		strings.Join(allowedList, ", "),
		strings.Join(violations, "\n"),
	)
}
