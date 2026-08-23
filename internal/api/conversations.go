package api

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/shiv-source/thoth/internal/store"
)

func listConversations(c echo.Context, d Deps) error {
	convs, err := d.Store.ListConversations()
	if err != nil {
		return internalError(c, d, "list conversations", err)
	}
	if convs == nil {
		convs = []store.Conversation{} // empty list serializes as [], never null — the client types it as an array
	}
	return c.JSON(http.StatusOK, map[string]any{"conversations": convs})
}

func createConversation(c echo.Context, d Deps) error {
	var body struct {
		Title string `json:"title"`
	}
	if err := c.Bind(&body); err != nil || body.Title == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "title is required"})
	}
	id, err := d.Store.CreateConversation(body.Title)
	if err != nil {
		return internalError(c, d, "create conversation", err)
	}
	return c.JSON(http.StatusOK, map[string]string{"id": id, "title": body.Title})
}

// deleteConversation removes the conversation and its messages; deleting a
// missing conversation is a no-op (idempotent).
func deleteConversation(c echo.Context, d Deps) error {
	if err := d.Store.DeleteConversation(c.Param("id")); err != nil {
		return internalError(c, d, "delete conversation", err)
	}
	return c.JSON(http.StatusOK, map[string]bool{"ok": true})
}

func getConversation(c echo.Context, d Deps) error {
	convs, err := d.Store.ListConversations()
	if err != nil {
		return internalError(c, d, "list conversations", err)
	}
	var conv *store.Conversation
	for i := range convs {
		if convs[i].ID == c.Param("id") {
			conv = &convs[i]
			break
		}
	}
	if conv == nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "conversation not found"})
	}
	msgs, err := d.Store.Messages(conv.ID)
	if err != nil {
		return internalError(c, d, "conversation messages", err)
	}
	return c.JSON(http.StatusOK, map[string]any{"conversation": conv, "messages": msgs})
}
