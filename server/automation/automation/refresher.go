package automation

import (
	"context"
	"fmt"
	"log"

	"github.com/cortezaproject/corteza/server/automation/types"
	"github.com/cortezaproject/corteza/server/pkg/expr"
)

type (
	//uiBlockRefreshHandler struct{}

	uiBlockRefreshHandlerRegistry interface {
		AddFunctions(...*types.Function)
		//AddFunctions(ff ...*atypes.Function)
		//Type(ref string) expr.Type
	}

	uiBlockRefreshHandler struct {
		reg uiBlockRefreshHandlerRegistry
		// nsSvc   uiBlockRefreshNamespaceService
		// pageSvc uiBlockRefreshPageService
		wsSvc uiBlockRefreshWebsocketService
		// logger  *zap.Logger
	}

	uiBlockRefreshWebsocketService interface {
		Send(msgType string, payload interface{}) error
	}
)

func RefreshUiBlockHandler(reg uiBlockRefreshHandlerRegistry, ws uiBlockRefreshWebsocketService) {
	h := &uiBlockRefreshHandler{
		reg:   reg,
		wsSvc: ws,
	}

	reg.AddFunctions(h.NewFunction()...)
}

// Functions returns all functions this handler provides
func (h uiBlockRefreshHandler) NewFunction() []*types.Function {
	return []*types.Function{
		{
			Ref:  "refreshUIBlock", // Updated to something descriptive
			Kind: "function",
			Labels: map[string]string{
				"category": "ui",
			},
			Meta: &types.FunctionMeta{
				Short:       "Trigger UI block refresh",
				Description: "Sends a websocket message to refresh a specific UI block by customID, pageID, and namespaceID",
			},
			Parameters: []*types.Param{
				{
					Name:     "customID",
					Types:    []string{"String"},
					Required: true,
					Meta: &types.ParamMeta{
						Label:       "Custom ID",
						Description: "Custom ID defined on the metric block",
					},
				},
				{
					Name:     "pageID",
					Types:    []string{"String"},
					Required: true,
					Meta: &types.ParamMeta{
						Label:       "Page ID",
						Description: "ID of the page containing the metric block",
					},
				},
				{
					Name:     "namespaceID",
					Types:    []string{"String"},
					Required: true,
					Meta: &types.ParamMeta{
						Label:       "Namespace ID",
						Description: "ID of the namespace containing the page",
					},
				},
			},
			Results: []*types.Param{
				{
					Name:  "success",
					Types: []string{"Boolean"},
					Meta: &types.ParamMeta{
						Label:       "Success",
						Description: "Indicates if the refresh was triggered successfully",
					},
				},
				{
					Name:  "message",
					Types: []string{"String"},
					Meta: &types.ParamMeta{
						Label:       "Message",
						Description: "Details about the operation result",
					},
				},
			},
			Handler: h.refreshUIBlock,
		},
	}
}

func (h uiBlockRefreshHandler) refreshUIBlock(ctx context.Context, args *expr.Vars) (*expr.Vars, error) {
	log.Println("refreshUIBlock called")

	// Extract the underlying map[string]expr.TypedValue
	varMap, ok := args.Get().(map[string]expr.TypedValue)
	if !ok {
		return nil, fmt.Errorf("invalid args format")
	}

	getString := func(key string) (string, error) {
		tv, exists := varMap[key]
		if !exists || tv == nil {
			return "", fmt.Errorf("%s parameter is required", key)
		}

		switch v := tv.(type) {
		case *expr.String:
			return v.GetValue(), nil
		default:
			return "", fmt.Errorf("%s parameter is not a string", key)
		}
	}

	customID, err := getString("customID")
	if err != nil {
		return nil, err
	}

	pageID, err := getString("pageID")
	if err != nil {
		return nil, err
	}

	namespaceID, err := getString("namespaceID")
	if err != nil {
		return nil, err
	}

	log.Printf("Received params: customID=%s, pageID=%s, namespaceID=%s\n", customID, pageID, namespaceID)

	// do your refresh logic here...

	out := &expr.Vars{}

	// Prepare data
	payload := map[string]interface{}{
		"customID":    customID,
		"pageID":      pageID,
		"namespaceID": namespaceID,
	}

	if err := h.wsSvc.Send("ui-block-refresh", payload); err != nil {
		out.Set("success", false)
		out.Set("message", fmt.Sprintf("failed to send refresh message: %v", err))
		return out, nil
	}

	out.Set("success", true)
	out.Set("message", fmt.Sprintf("Refresh triggered for customID '%s'", customID))
	return out, nil

}
