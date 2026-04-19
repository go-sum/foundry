package render

import (
	icons "github.com/go-sum/componentry/icons"
	"github.com/go-sum/componentry/ui/core"
)

// PropsFor returns IconProps for a semantic icon key using the default registry.
func PropsFor(key icons.Key, base core.IconProps) core.IconProps {
	return PropsForRegistry(icons.Default, key, base)
}

// PropsForRegistry returns IconProps for a semantic icon key using an explicit registry.
func PropsForRegistry(r *icons.Registry, key icons.Key, base core.IconProps) core.IconProps {
	ref, ok := r.Resolve(key)
	if !ok {
		return base
	}
	base.Src = ref.Sprite
	base.ID = ref.ID
	return base
}
