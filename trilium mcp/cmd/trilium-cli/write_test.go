package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"trilium-mcp/internal/notes"
	"trilium-mcp/internal/trilium"
)

// TestCLIWriteRoundTrip 验证 CLI 新建、追加、片段和整篇更新、重复调用及根笔记保护形成完整闭环。
func TestCLIWriteRoundTrip(t *testing.T) {
	metadata := map[string]trilium.Note{"temp": {NoteID: "temp", Title: "temp", Type: "text", MIME: "text/html", ParentNoteIDs: []string{"root"}}}
	bodies := map[string]string{"temp": "scope root"}
	creates, puts, deletes := 0, 0, 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "test-token" {
			t.Error("missing token")
			w.WriteHeader(401)
			return
		}
		if r.Method == http.MethodPost && r.URL.Path == "/etapi/create-note" {
			creates++
			var input trilium.CreateNoteInput
			if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
				t.Error(err)
				w.WriteHeader(400)
				return
			}
			metadata[input.NoteID] = trilium.Note{NoteID: input.NoteID, Title: input.Title, Type: "text", MIME: "text/html", ParentNoteIDs: []string{input.ParentNoteID}}
			bodies[input.NoteID] = input.Content
			w.WriteHeader(http.StatusCreated)
			return
		}
		path := strings.TrimPrefix(r.URL.Path, "/etapi/notes/")
		id := strings.TrimSuffix(path, "/content")
		note, exists := metadata[id]
		if !exists {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if r.Method == http.MethodGet && path == id {
			json.NewEncoder(w).Encode(note)
			return
		}
		if r.Method == http.MethodGet && path == id+"/content" {
			io.WriteString(w, bodies[id])
			return
		}
		if r.Method == http.MethodPut && path == id+"/content" {
			puts++
			body, _ := io.ReadAll(r.Body)
			bodies[id] = string(body)
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Method == http.MethodDelete && path == id {
			deletes++
			delete(metadata, id)
			delete(bodies, id)
			w.WriteHeader(http.StatusNoContent)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()
	directory := t.TempDir()
	configFile := filepath.Join(directory, ".env")
	if err := os.WriteFile(configFile, nil, 0600); err != nil {
		t.Fatal(err)
	}
	contentFile := filepath.Join(directory, "content.md")
	if err := os.WriteFile(contentFile, []byte("# 标题\n\n正文"), 0600); err != nil {
		t.Fatal(err)
	}
	getenv := func(key string) string {
		switch key {
		case "TRILIUM_URL":
			return server.URL
		case "TRILIUM_TOKEN":
			return "test-token"
		case "TRILIUM_WRITE_ROOT_ID":
			return "temp"
		}
		return ""
	}
	invoke := func(args ...string) ([]byte, error) {
		var out, errOut bytes.Buffer
		configuredArgs := append([]string{args[0], "-env-file", configFile}, args[1:]...)
		err := run(context.Background(), configuredArgs, &out, &errOut, getenv)
		return out.Bytes(), err
	}
	createdJSON, err := invoke("create", "-title", "CLI test", "-file", contentFile, "-operation", "cli-create")
	if err != nil {
		t.Fatal(err)
	}
	var created notes.Result
	if err := json.Unmarshal(createdJSON, &created); err != nil {
		t.Fatal(err)
	}
	readJSON, err := invoke("read", created.NoteID)
	if err != nil {
		t.Fatal(err)
	}
	var read struct {
		ContentSHA256 string `json:"contentSHA256"`
	}
	if err := json.Unmarshal(readJSON, &read); err != nil || read.ContentSHA256 != created.ContentSHA256 {
		t.Fatal("read hash does not match created body")
	}
	appendFile := filepath.Join(directory, "append.md")
	if err := os.WriteFile(appendFile, []byte("**追加**"), 0600); err != nil {
		t.Fatal(err)
	}
	args := []string{"append", "-file", appendFile, "-operation", "cli-append", "-expected-sha256", created.ContentSHA256, created.NoteID}
	for i := 0; i < 2; i++ {
		body, err := invoke(args...)
		if err != nil || !json.Valid(body) {
			t.Fatalf("append command failed: %v", err)
		}
	}
	if creates != 1 || puts != 1 || strings.Count(bodies[created.NoteID], "<strong>追加</strong>") != 1 || bodies["temp"] != "scope root" {
		t.Fatal("CLI duplicated writes or edited the scope root")
	}
	if _, err := os.Stat(filepath.Join(directory, ".trilium-state", "cli-append.json")); err != nil {
		t.Fatal("default state directory was not beside the config file")
	}
	args[len(args)-1] = "temp"
	if _, err := invoke(args...); err == nil || puts != 1 {
		t.Fatal("scope root edit was accepted")
	}

	beforeUpdateJSON, err := invoke("read", created.NoteID)
	if err != nil {
		t.Fatal(err)
	}
	var beforeUpdate struct {
		ContentSHA256 string `json:"contentSHA256"`
	}
	if err := json.Unmarshal(beforeUpdateJSON, &beforeUpdate); err != nil {
		t.Fatal(err)
	}
	replacementFile, matchFile := filepath.Join(directory, "replace.html"), filepath.Join(directory, "match.html")
	if err := os.WriteFile(replacementFile, []byte("<p>更新后的正文</p>"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(matchFile, []byte("<p>正文</p>"), 0600); err != nil {
		t.Fatal(err)
	}
	updateArgs := []string{"update", "-file", replacementFile, "-match-file", matchFile, "-operation", "cli-update-fragment", "-expected-sha256", beforeUpdate.ContentSHA256, created.NoteID}
	var updated notes.Result
	for i := 0; i < 2; i++ {
		body, err := invoke(updateArgs...)
		if err != nil || json.Unmarshal(body, &updated) != nil {
			t.Fatalf("fragment update failed: %v", err)
		}
	}
	if puts != 2 || !updated.AlreadyApplied || !strings.Contains(bodies[created.NoteID], "<p>更新后的正文</p>") || !strings.Contains(bodies[created.NoteID], "<strong>追加</strong>") {
		t.Fatal("fragment update lost surrounding content or repeated PUT")
	}
	const replacement = "<h2>整体更新</h2><p>最终正文</p>"
	if err := os.WriteFile(replacementFile, []byte(replacement), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := invoke("update", "-file", replacementFile, "-operation", "cli-update-full", "-expected-sha256", updated.ContentSHA256, created.NoteID); err != nil {
		t.Fatal(err)
	}
	if puts != 3 || bodies[created.NoteID] != replacement || metadata[created.NoteID].Title != "CLI test" {
		t.Fatal("default HTML update did not replace the body exactly")
	}
	finalReadJSON, err := invoke("read", created.NoteID)
	if err != nil {
		t.Fatal(err)
	}
	var finalRead struct {
		ContentSHA256 string `json:"contentSHA256"`
	}
	if err := json.Unmarshal(finalReadJSON, &finalRead); err != nil {
		t.Fatal(err)
	}
	var deleted notes.DeleteResult
	deleteArgs := []string{"delete", "-operation", "cli-delete", "-expected-sha256", finalRead.ContentSHA256, created.NoteID}
	for i := 0; i < 2; i++ {
		body, err := invoke(deleteArgs...)
		if err != nil || json.Unmarshal(body, &deleted) != nil {
			t.Fatalf("delete command failed: %v", err)
		}
	}
	if deletes != 1 || !deleted.Deleted || !deleted.AlreadyApplied {
		t.Fatalf("delete replay was not idempotent: %+v, deletes=%d", deleted, deletes)
	}
	if _, exists := metadata[created.NoteID]; exists {
		t.Fatal("deleted note remains in the fake ETAPI tree")
	}
}

// TestCLIUpdateRejectsEmptyMatchFile 验证显式提供的空匹配文件不会被解释为整篇替换。
func TestCLIUpdateRejectsEmptyMatchFile(t *testing.T) {
	directory := t.TempDir()
	content, match := filepath.Join(directory, "new.html"), filepath.Join(directory, "empty.html")
	if err := os.WriteFile(content, []byte("<p>new</p>"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(match, nil, 0600); err != nil {
		t.Fatal(err)
	}
	getenv := func(key string) string {
		if key == "TRILIUM_WRITE_ROOT_ID" {
			return "temp"
		}
		return testEnv("http://127.0.0.1:1")(key)
	}
	var out, errOut bytes.Buffer
	args := withEmptyEnvFile(t, []string{"update", "-file", content, "-match-file", match, "-operation", "empty-match", "-expected-sha256", strings.Repeat("0", 64), "note123"})
	err := run(context.Background(), args, &out, &errOut, getenv)
	if err == nil || !strings.Contains(err.Error(), "match file must not be empty") || out.Len() != 0 {
		t.Fatalf("empty match was not rejected safely: %v", err)
	}
}

// TestCLIWriteRequiresExplicitScope 验证未配置写入根时，即使提供正文和操作 ID 也拒绝写入。
func TestCLIWriteRequiresExplicitScope(t *testing.T) {
	var out, errOut bytes.Buffer
	args := withEmptyEnvFile(t, []string{"create", "-title", "no", "-file", "unused.md", "-operation", "no-scope"})
	err := run(context.Background(), args, &out, &errOut, testEnv("http://127.0.0.1:1"))
	if err == nil || !strings.Contains(err.Error(), "TRILIUM_WRITE_ROOT_ID") {
		t.Fatalf("expected explicit scope requirement, got %v", err)
	}
}
