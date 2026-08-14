package api

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/shiv-source/thoth/internal/store"
)

func listConversations(c echo.Context, d Deps) error {
	convs, err := d.Store.ListConversations()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
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
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, map[string]string{"id": id, "title": body.Title})
}

func getConversation(c echo.Context, d Deps) error {
	convs, err := d.Store.ListConversations()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
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
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, map[string]any{"conversation": conv, "messages": msgs})
}
