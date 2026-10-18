// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Phoobobo

package v1

import (
	"context"
	"encoding/json"
	"github.com/google/uuid"
	"github.com/sysadminsmedia/homebox/backend/internal/core/services"
	"github.com/sysadminsmedia/homebox/backend/internal/data/repo"
	"github.com/sysadminsmedia/homebox/backend/internal/sys/config"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAIHistoryNoTurnLimit(t *testing.T) {
	messages := make([]aiMessage, 101)
	for i := range messages {
		messages[i] = aiMessage{Role: "user", Content: strings.Repeat("记录", 2001)}
	}
	require.NoError(t, validateAIHistory(aiChatInput{Messages: messages}))
}

func TestAIContextSummary(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		var payload struct {
			Messages []aiMessage `json:"messages"`
			Tools    any         `json:"tools"`
		}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
		require.Nil(t, payload.Tools)
		require.Equal(t, aiSummaryPrompt, payload.Messages[0].Content)
		require.Contains(t, payload.Messages[1].Content, "keys")
		_, _ = w.Write([]byte(`{"usage":{"prompt_tokens":123},"choices":[{"message":{"role":"assistant","content":"keys: checked drawer; not saved"}}]}`))
	}))
	defer srv.Close()
	aiEndpoints["summary-test"] = srv.URL
	defer delete(aiEndpoints, "summary-test")
	s := aiStored{Provider: "summary-test", Model: "mock"}
	messages := []aiMessage{{Role: "system", Content: aiSystemPrompt}, {Role: "user", Content: "keys"}, {Role: "assistant", Content: "checking keys"}, {Role: "user", Content: "keep looking"}, {Role: "assistant", ToolCalls: []aiToolCall{{ID: "call"}}}, {Role: "tool", ToolCallID: "call", Content: "keys: drawer empty"}}
	untouched, summary, err := aiCompactContext(context.Background(), s, "test", messages, 200000)
	require.NoError(t, err)
	require.Equal(t, messages, untouched)
	require.Empty(t, summary)
	require.Zero(t, calls)
	compacted, summary, err := aiCompactContext(context.Background(), s, "test", messages, 200001)
	require.NoError(t, err)
	require.Len(t, compacted, 3)
	require.Equal(t, messages[0], compacted[0])
	require.Equal(t, messages[3], compacted[2])
	require.Equal(t, "assistant", compacted[1].Role)
	require.Equal(t, summary, compacted[1].Content)
	require.Equal(t, 1, calls)
	// A subsequent turn carries the summary forward without re-compressing.
	compacted = append(compacted, aiMessage{Role: "assistant", Content: "next"}, aiMessage{Role: "user", Content: "continue"})
	_, summary, err = aiCompactContext(context.Background(), s, "test", compacted, 0)
	require.NoError(t, err)
	require.Empty(t, summary)
	require.Equal(t, 1, calls)
	// Initial history can trigger compaction before any provider usage exists.
	messages[1].Content = strings.Repeat("keys ", 170000)
	_, summary, err = aiCompactContext(context.Background(), s, "test", messages, 0)
	require.NoError(t, err)
	require.NotEmpty(t, summary)
	require.Greater(t, calls, 2)
	// Failure must not replace history with a blank summary.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, _, err = aiCompactContext(ctx, s, "test", messages, 200001)
	require.Error(t, err)
}

func TestAIChatContinuesPastSixRounds(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls <= 7 {
			// Unknown tools return an error to the model without database side effects.
			_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","tool_calls":[{"id":"test","type":"function","function":{"name":"unknown","arguments":"{}"}}]}}]}`))
		} else {
			_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"finished"}}]}`))
		}
	}))
	defer srv.Close()
	aiEndpoints["round-test"] = srv.URL
	defer delete(aiEndpoints, "round-test")
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "ai"), 0700))
	key := []byte(strings.Repeat("k", 32))
	secret, err := aiSeal(key, "synthetic-test-key")
	require.NoError(t, err)
	group := uuid.New()
	data, err := json.Marshal(aiStored{Provider: "round-test", Model: "mock", Secret: secret})
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "ai", "master.key"), key, 0600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "ai", group.String()+".json"), data, 0600))
	user := repo.UserOut{}
	user.ID = uuid.New()
	user.DefaultGroupID = group
	ctx := services.SetTenantCtx(services.SetUserCtx(context.Background(), &user, ""), group)
	ctrl := &V1Controller{config: &config.Config{Storage: config.Storage{PrefixPath: dir}}}
	r := httptest.NewRequest("POST", "/ai/chat", strings.NewReader(`{"messages":[{"role":"user","content":"find keys"}]}`)).WithContext(ctx)
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	require.NoError(t, ctrl.HandleAIChat()(w, r))
	var out aiChatOut
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &out))
	require.Equal(t, "finished", out.Reply)
	require.Equal(t, 8, calls)
}
