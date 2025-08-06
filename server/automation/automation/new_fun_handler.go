// ui_block_refresh_handler.go
package automation

import (
	"context"
	"fmt"
	"log"
	"strconv"

	"github.com/cortezaproject/corteza/server/automation/types"
	composeTypes "github.com/cortezaproject/corteza/server/compose/types"
	"github.com/cortezaproject/corteza/server/pkg/expr"
)

type (
	uiBlockRefreshHandler struct {
		nsSvc   namespaceService
		pageSvc pageService
		ws      websocketSender
	}

	namespaceService interface {
		FindByHandle(ctx context.Context, handle string) (*composeTypes.Namespace, error)
	}

	pageService interface {
		Find(ctx context.Context, filter composeTypes.PageFilter) (composeTypes.PageSet, composeTypes.PageFilter, error)
	}

	websocketSender interface {
		Send(kind string, payload interface{}, userIDs ...uint64) error
	}

	handlerRegistry interface {
		AddFunctions(...*types.Function)
	}
)

// Register handler
func UIBlockRefreshHandler(reg handlerRegistry, nsSvc namespaceService, pageSvc pageService, ws websocketSender) {
	h := &uiBlockRefreshHandler{
		nsSvc:   nsSvc,
		pageSvc: pageSvc,
		ws:      ws,
	}
	reg.AddFunctions(h.Functions()...)
}

// Expose functions
func (h *uiBlockRefreshHandler) Functions() []*types.Function {
	return []*types.Function{
		{
			Ref:  "refreshUIBlock",
			Kind: "function",
			Labels: map[string]string{
				"category": "ui-interaction",
			},
			Meta: &types.FunctionMeta{
				Short:       "Refresh UI block",
				Description: "Refreshes a specific UI block on a page by sending websocket message",
			},
			Parameters: []*types.Param{
				{Name: "customID", Types: []string{"String"}, Required: true},
				{Name: "pagina", Types: []string{"String"}, Required: true},
				{Name: "Namespace", Types: []string{"String"}, Required: true},
			},
			Results: []*types.Param{
				{Name: "success", Types: []string{"Boolean"}},
				{Name: "message", Types: []string{"String"}},
				{Name: "blockID", Types: []string{"String"}},
			},
			Handler: h.refreshUIBlock,
		},
	}
}

// Main handler logic
func (h *uiBlockRefreshHandler) refreshUIBlock(ctx context.Context, args *expr.Vars) (*expr.Vars, error) {
	log.Println("refreshUIBlock called")

	varMap, ok := args.Get().(map[string]expr.TypedValue)
	if !ok {
		return h.errorResult("invalid arguments", "")
	}

	customID, err := h.extractStringParam(varMap, "customID")
	if err != nil {
		return h.errorResult("customID parameter is required", "")
	}
	pageHandle, err := h.extractStringParam(varMap, "pagina")
	if err != nil {
		return h.errorResult("pagina parameter is required", "")
	}
	namespaceHandle, err := h.extractStringParam(varMap, "Namespace")
	if err != nil {
		return h.errorResult("Namespace parameter is required", "")
	}

	ns, err := h.nsSvc.FindByHandle(ctx, namespaceHandle)
	if err != nil {
		return h.errorResult(fmt.Sprintf("namespace not found: %v", err), "")
	}

	pages, _, err := h.pageSvc.Find(ctx, composeTypes.PageFilter{
		NamespaceID: ns.ID,
		Handle:      pageHandle,
	})
	if err != nil || len(pages) == 0 {
		return h.errorResult("page not found", "")
	}
	page := pages[0]

	var foundBlock *composeTypes.PageBlock
	var foundBlockIndex int
	for i, block := range page.Blocks {
		if h.blockMatchesCustomID(block, customID) {
			foundBlock = &block
			foundBlockIndex = i
			break
		}
	}

	if foundBlock == nil {
		return h.errorResult(fmt.Sprintf("block '%s' not found", customID), "")
	}

	payload := map[string]interface{}{
		"type":        "block-refresh",
		"customID":    customID,
		"blockID":     strconv.FormatUint(foundBlock.BlockID, 10),
		"blockIndex":  foundBlockIndex,
		"blockKind":   foundBlock.Kind,
		"pageID":      strconv.FormatUint(page.ID, 10),
		"pageHandle":  page.Handle,
		"namespaceID": strconv.FormatUint(ns.ID, 10),
		"namespace":   namespaceHandle,
	}

	if err := h.ws.Send("ui-block-refresh", payload); err != nil {
		return h.errorResult(fmt.Sprintf("websocket send failed: %v", err), strconv.FormatUint(foundBlock.BlockID, 10))
	}

	msg := fmt.Sprintf("Refreshed block %d (%s) on page %s", foundBlock.BlockID, foundBlock.Kind, page.Handle)
	log.Println(msg)
	return h.successResult(msg, strconv.FormatUint(foundBlock.BlockID, 10))
}

// Match logic (simplified version of original)
func (h *uiBlockRefreshHandler) blockMatchesCustomID(block composeTypes.PageBlock, customID string) bool {
	if id, err := strconv.ParseUint(customID, 10, 64); err == nil && block.BlockID == id {
		return true
	}
	if block.Options != nil {
		for _, key := range []string{"customID", "id", "blockId", "identifier", "name"} {
			if val, ok := block.Options[key]; ok && fmt.Sprintf("%v", val) == customID {
				return true
			}
		}
	}
	if block.Meta != nil {
		if val, ok := block.Meta["customID"]; ok && fmt.Sprintf("%v", val) == customID {
			return true
		}
	}
	if block.Title == customID {
		return true
	}
	return false
}

// Helpers
func (h *uiBlockRefreshHandler) extractStringParam(varMap map[string]expr.TypedValue, paramName string) (string, error) {
	tv, ok := varMap[paramName]
	if !ok || tv == nil {
		return "", fmt.Errorf("parameter %s is required", paramName)
	}
	switch v := tv.(type) {
	case *expr.String:
		return v.GetValue(), nil
	default:
		return fmt.Sprintf("%v", tv), nil
	}
}

func (h *uiBlockRefreshHandler) successResult(msg, blockID string) (*expr.Vars, error) {
	out := &expr.Vars{}
	out.Set("success", true)
	out.Set("message", msg)
	out.Set("blockID", blockID)
	return out, nil
}

func (h *uiBlockRefreshHandler) errorResult(msg, blockID string) (*expr.Vars, error) {
	out := &expr.Vars{}
	out.Set("success", false)
	out.Set("message", msg)
	out.Set("blockID", blockID)
	return out, nil
}
