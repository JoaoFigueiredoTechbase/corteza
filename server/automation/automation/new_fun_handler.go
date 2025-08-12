// your_handler.go
package automation

import (
	"context"
	"fmt"
	"log"

	"github.com/cortezaproject/corteza/server/automation/types"
	"github.com/cortezaproject/corteza/server/pkg/expr"
)

type (
	yourHandler struct{}

	// Registry interface - matches what's expected
	handlerRegistry interface {
		AddFunctions(...*types.Function)
	}
)

// YourHandler registers your custom handler functions
func YourHandler(reg handlerRegistry) {
	h := &yourHandler{}

	// Register your functions
	reg.AddFunctions(h.Functions()...)
}

// Functions returns all functions this handler provides
func (h yourHandler) Functions() []*types.Function {
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
			Handler: h.yourCustomFunction,
		},
	}
}

func (h yourHandler) yourCustomFunction(ctx context.Context, args *expr.Vars) (*expr.Vars, error) {
	log.Println("yourCustomFunction called")

	// Get the map of variables
	varMap, ok := args.Get().(map[string]expr.TypedValue)
	if !ok {
		log.Println("Error: could not extract variables map from args")
		return nil, fmt.Errorf("internal error: invalid arguments")
	}

	// Get the "input" parameter as expr.TypedValue
	tv, ok := varMap["input"]
	if !ok || tv == nil {
		log.Println("Error: 'input' parameter is required")
		return nil, fmt.Errorf("input parameter is required")
	}

	// Try to extract the string value
	var input string
	switch v := tv.(type) {
	case *expr.String:
		input = v.GetValue()
	default:
		// fallback: try to use fmt.Sprintf
		input = fmt.Sprintf("%v", tv)
	}

	log.Printf("Received input: %s\n", input)

	// Do something with the input
	result := "Processed: " + input
	log.Printf("Generated result: %s\n", result)

	// Return results
	out := &expr.Vars{}
	out.Set("result", result)
	log.Println("Returning result from yourCustomFunction")

	return out, nil
}
