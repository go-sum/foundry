package ddl

// Filter returns a new Schema containing only the objects whose names appear in mask.
// Used to partition a full database introspection result into per-source-file subsets.
func Filter(full *Schema, mask *Schema) *Schema {
	extSet := make(map[string]bool, len(mask.Extensions))
	for _, e := range mask.Extensions {
		extSet[e] = true
	}
	tableSet := make(map[string]bool, len(mask.Tables))
	for _, t := range mask.Tables {
		tableSet[t.Name] = true
	}
	indexSet := make(map[string]bool, len(mask.Indexes))
	for _, i := range mask.Indexes {
		indexSet[i.Name] = true
	}
	funcSet := make(map[string]bool, len(mask.Functions))
	for _, f := range mask.Functions {
		funcSet[f.Name] = true
	}
	trigSet := make(map[string]bool, len(mask.Triggers))
	for _, t := range mask.Triggers {
		trigSet[t.Name] = true
	}

	result := &Schema{}
	for _, e := range full.Extensions {
		if extSet[e] {
			result.Extensions = append(result.Extensions, e)
		}
	}
	for _, t := range full.Tables {
		if tableSet[t.Name] {
			result.Tables = append(result.Tables, t)
		}
	}
	for _, i := range full.Indexes {
		if indexSet[i.Name] {
			result.Indexes = append(result.Indexes, i)
		}
	}
	for _, f := range full.Functions {
		if funcSet[f.Name] {
			result.Functions = append(result.Functions, f)
		}
	}
	for _, t := range full.Triggers {
		if trigSet[t.Name] {
			result.Triggers = append(result.Triggers, t)
		}
	}
	return result
}
