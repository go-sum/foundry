package render

import (
	icons "github.com/go-sum/foundry/pkg/componentry/icons"
	"github.com/go-sum/foundry/pkg/componentry/ui/core"
)

// PropsForRegistry returns IconProps for a semantic icon key using an explicit registry.
func PropsForRegistry(r *icons.Registry, key icons.Key, base core.IconProps) core.IconProps {
	if r == nil {
		return base
	}
	ref, ok := r.Resolve(key)
	if !ok {
		return base
	}
	base.Src = ref.Sprite
	base.ID = ref.ID
	return base
}
