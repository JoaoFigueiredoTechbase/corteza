package automation

import (
	"context"

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
			Ref:  "yourCustomFunction",
			Kind: "function",
			Labels: map[string]string{
				"category": "your-category",
			},
			Meta: &types.FunctionMeta{
				Short:       "Your function description",
				Description: "Detailed description of what your function does",
			},
			Parameters: []*types.Param{
				{
					Name:     "input",
					Types:    []string{"String"},
					Required: true,
					Meta: &types.ParamMeta{
						Label:       "Input Parameter",
						Description: "Description of the input parameter",
					},
				},
			},
			Results: []*types.Param{
				{
					Name:  "result",
					Types: []string{"String"},
					Meta: &types.ParamMeta{
						Label:       "Result",
						Description: "The result of your function",
					},
				},
			},
			Handler: h.refreshUIBlock,
		},
	}
}

func (h uiBlockRefreshHandler) refreshUIBlock(ctx context.Context, args *expr.Vars) (*expr.Vars, error) {
	out := &expr.Vars{}

	// customID, _ := args.GetString("customID")
	// pageID, _ := args.GetUint64("pageID")
	// namespaceID, _ := args.GetUint64("namespaceID")

	// rArgs := &refreshUIBlockArgs{
	// 	CustomID:    customID,
	// 	pageID:      pageID,
	// 	namespaceID: namespaceID,
	// }

	// // Resolve namespace
	// var namespace *composeTypes.Namespace
	// var err error

	// if args.namespaceID > 0 {
	// 	namespace, err = h.nsSvc.FindByID(ctx, args.namespaceID)
	// } else if args.namespaceHandle != "" {
	// 	namespace, err = h.nsSvc.FindByHandle(ctx, args.namespaceHandle)
	// } else {
	// 	out.Set("success", false)
	// 	out.Set("message", "namespace ID or handle is required")
	// 	return out, nil
	// }

	// if err != nil {
	// 	h.logger.Error("failed to find namespace", zap.Error(err))
	// 	out.Set("success", false)
	// 	out.Set("message", fmt.Sprintf("namespace not found: %v", err))
	// 	return out, nil
	// }

	// // Resolve page
	// var page *composeTypes.Page

	// if args.pageID > 0 {
	// 	page, err = h.pageSvc.FindByID(ctx, namespace.ID, args.pageID)
	// } else if args.pageHandle != "" {
	// 	page, err = h.pageSvc.FindByHandle(ctx, namespace.ID, args.pageHandle)
	// } else {
	// 	out.Set("success", false)
	// 	out.Set("message", "page ID or handle is required")
	// 	return out, nil
	// }

	// if err != nil {
	// 	h.logger.Error("failed to find page", zap.Error(err))
	// 	out.Set("success", false)
	// 	out.Set("message", fmt.Sprintf("page not found: %v", err))
	// 	return out, nil
	// }

	// // Find the block with the custom ID
	// var foundBlock *composeTypes.PageBlock
	// for i := range page.Blocks {
	// 	block := &page.Blocks[i]
	// 	if block.Options != nil {
	// 		if customID, ok := block.Options["customID"].(string); ok && customID == args.CustomID {
	// 			foundBlock = block
	// 			break
	// 		}
	// 	}
	// }

	// if foundBlock == nil {
	// 	out.Set("success", false)
	// 	out.Set("message", fmt.Sprintf("block with customID '%s' not found on page", args.CustomID))
	// 	return out, nil
	// }

	// Send websocket message for block refresh
	// payload := map[string]interface{}{
	// 	"type":        "ui-block-refresh",
	// 	"customID":    args.CustomID,
	// 	"pageID":      args.pageID,
	// 	"namespaceID": args.namespaceID,
	// }

	// if err := h.wsSvc.Send("ui-block-refresh", payload); err != nil {
	// 	out.Set("success", false)
	// 	out.Set("message", fmt.Sprintf("failed to send refresh message: %v", err))
	// 	return out, nil
	// }

	// // h.logger.Info("UI block refresh triggered",
	// // 	zap.String("customID", args.CustomID),
	// // 	zap.Uint64("blockID", foundBlock.BlockID),
	// // 	zap.Uint64("pageID", page.ID),
	// // 	zap.Uint64("namespaceID", namespace.ID))

	// out.Set("success", true)
	// out.Set("message", fmt.Sprintf("UI block refresh triggered for customID '%s'", args.CustomID))
	return out, nil
}
