package tools

func (r *Registry) toolSchemas() []map[string]any {
	schemas := make([]map[string]any, 0, len(r.tools))
	for _, tool := range r.List() {
		schemas = append(schemas, toolSchema(tool))
	}
	return schemas
}

func toolSchema(tool Tool) map[string]any {
	return map[string]any{
		"type": "function",
		"function": map[string]any{
			"name":        tool.Name(),
			"description": tool.Description(),
			"parameters":  parameterSchema(tool.Parameters()),
		},
	}
}

func parameterSchema(params []Parameter) map[string]any {
	properties := make(map[string]any)
	required := make([]string, 0)
	for _, param := range params {
		properties[param.Name] = parameterDefinition(param)
		if param.Required {
			required = append(required, param.Name)
		}
	}
	return map[string]any{
		"type":       "object",
		"properties": properties,
		"required":   required,
	}
}

func parameterDefinition(param Parameter) map[string]any {
	def := map[string]any{
		"type":        param.Type,
		"description": param.Description,
	}
	if len(param.Enum) > 0 {
		def["enum"] = param.Enum
	}
	return def
}
