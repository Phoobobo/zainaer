// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Phoobobo

package v1

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/hay-kot/httpkit/errchain"
	"github.com/hay-kot/httpkit/server"
	"github.com/sysadminsmedia/homebox/backend/internal/core/services"
	"github.com/sysadminsmedia/homebox/backend/internal/sys/validate"
)

// Conversation snapshots contain display data only. Loading one never runs tools or
// restores server-held confirmation drafts. Keep these below ai/ for full backups.
type aiConversation struct {
	ID           string             `json:"id"`
	Revision     int                `json:"revision"`
	Title        string             `json:"title"`
	UpdatedAt    time.Time          `json:"updatedAt"`
	Messages     []aiHistoryMessage `json:"messages,omitempty"`
	Summary      string             `json:"summary,omitempty"`
	HistoryStart int                `json:"historyStart"`
}
type aiHistoryMessage struct {
	Role      string            `json:"role"`
	Content   string            `json:"content"`
	UsedModel string            `json:"usedModel,omitempty"`
	Items     []aiResult        `json:"items,omitempty"`
	Drafts    []json.RawMessage `json:"drafts,omitempty"`
}

var aiHistoryMu sync.Mutex

func (ctrl *V1Controller) aiHistoryDir(ctx services.Context) string {
	return filepath.Join(ctrl.config.Storage.PrefixPath, "ai", "conversations", ctx.GID.String(), ctx.UID.String())
}
func aiReadConversation(path string) (aiConversation, error) {
	var c aiConversation
	b, err := os.ReadFile(path)
	if err != nil {
		return c, err
	}
	err = json.Unmarshal(b, &c)
	return c, err
}
func (ctrl *V1Controller) HandleAIHistory() errchain.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) error {
		ctx := services.NewContext(r.Context())
		dir := ctrl.aiHistoryDir(ctx)
		aiHistoryMu.Lock()
		defer aiHistoryMu.Unlock()
		if r.Method == http.MethodGet {
			if id := r.URL.Query().Get("id"); id != "" {
				parsed, err := uuid.Parse(id)
				if err != nil {
					return validate.NewRequestError(errors.New("invalid conversation id"), 400)
				}
				c, err := aiReadConversation(filepath.Join(dir, parsed.String()+".json"))
				if os.IsNotExist(err) {
					return validate.NewRequestError(errors.New("conversation not found"), 404)
				}
				if err != nil {
					return validate.NewRequestError(errors.New("cannot read conversation"), 500)
				}
				return server.JSON(w, 200, c)
			}
			entries, err := os.ReadDir(dir)
			if err != nil && !os.IsNotExist(err) {
				return validate.NewRequestError(errors.New("cannot list conversations"), 500)
			}
			list := []aiConversation{}
			for _, entry := range entries {
				if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
					continue
				}
				c, err := aiReadConversation(filepath.Join(dir, entry.Name()))
				if err != nil {
					return validate.NewRequestError(errors.New("cannot read conversation history"), 500)
				}
				c.Messages = nil
				c.Summary = ""
				list = append(list, c)
			}
			sort.Slice(list, func(i, j int) bool { return list[i].UpdatedAt.After(list[j].UpdatedAt) })
			return server.JSON(w, 200, list)
		}
		if err := aiJSONRequest(r); err != nil {
			return err
		}
		r.Body = http.MaxBytesReader(w, r.Body, 8<<20)
		var c aiConversation
		if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
			return validate.NewRequestError(errors.New("invalid conversation"), 400)
		}
		id, err := uuid.Parse(c.ID)
		if err != nil || c.HistoryStart < 0 || c.HistoryStart > len(c.Messages) || c.Revision < 0 {
			return validate.NewRequestError(errors.New("invalid conversation"), 400)
		}
		for _, m := range c.Messages {
			if m.Role != "user" && m.Role != "assistant" {
				return validate.NewRequestError(errors.New("invalid message role"), 400)
			}
		}
		c.ID = id.String()
		path := filepath.Join(dir, c.ID+".json")
		old, err := aiReadConversation(path)
		if err != nil && !os.IsNotExist(err) {
			return validate.NewRequestError(errors.New("cannot read conversation"), 500)
		}
		if c.Revision != old.Revision {
			return validate.NewRequestError(errors.New("conversation changed in another tab; reopen history before continuing"), 409)
		}
		c.Title = ""
		for _, m := range c.Messages {
			if m.Role == "user" {
				c.Title = m.Content
				break
			}
		}
		if utf8.RuneCountInString(c.Title) > 60 {
			c.Title = string([]rune(c.Title)[:60])
		}
		c.Revision++
		c.UpdatedAt = time.Now().UTC()
		if err := os.MkdirAll(dir, 0700); err != nil {
			return validate.NewRequestError(errors.New("cannot create conversation storage"), 500)
		}
		b, err := json.Marshal(c)
		if err != nil {
			return err
		}
		tmp, err := os.CreateTemp(dir, ".conversation-*")
		if err != nil {
			return err
		}
		defer os.Remove(tmp.Name())
		if _, err = tmp.Write(b); err == nil {
			err = tmp.Sync()
		}
		closeErr := tmp.Close()
		if err == nil {
			err = closeErr
		}
		if err == nil {
			err = os.Rename(tmp.Name(), path)
		}
		if err != nil {
			return validate.NewRequestError(errors.New("cannot save conversation"), 500)
		}
		return server.JSON(w, 200, c)
	}
}
