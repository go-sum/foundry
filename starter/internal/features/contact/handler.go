package contact

import (
	"errors"
	"time"

	"github.com/go-sum/foundry/internal/view/page"
	"github.com/go-sum/foundry/internal/view/partial/contactpartial"
	"github.com/go-sum/foundry/pkg/web"
	"github.com/go-sum/foundry/pkg/web/render"
	"github.com/go-sum/foundry/pkg/web/router"
	"github.com/go-sum/foundry/pkg/web/validate"
	viewstate "github.com/go-sum/foundry/pkg/web/viewstate"
)

// Handler serves the contact form endpoints.
type Handler struct {
	rt      *router.Router
	reqOpts []viewstate.RequestOption
	svc     Service
	val     validate.Validator
}

// NewHandler creates a contact Handler.
func NewHandler(rt *router.Router, svc Service, val validate.Validator, opts ...viewstate.RequestOption) *Handler {
	return &Handler{rt: rt, reqOpts: opts, svc: svc, val: val}
}

// Form renders the contact form page.
func (h *Handler) Form(c *web.Context) (web.Response, error) {
	vr := viewstate.NewRequest(c, h.reqOpts...)
	submitURL := h.rt.MustReverse("contact.submit", nil)
	data := contactpartial.FormData{}
	return viewstate.Render(vr, page.ContactPage(vr, submitURL, data), contactpartial.ContactForm(vr, submitURL, data))
}

// Submit processes a contact form POST.
func (h *Handler) Submit(c *web.Context) (web.Response, error) {
	vr := viewstate.NewRequest(c, h.reqOpts...)
	submitURL := h.rt.MustReverse("contact.submit", nil)

	var input ContactInput
	if err := validate.Bind(h.val, c.Request, &input); err != nil {
		if verrs, ok := errors.AsType[validate.Errors](err); ok {
			fieldErrors := make(map[string][]string)
			for _, fe := range verrs {
				fieldErrors[fe.Field] = append(fieldErrors[fe.Field], fe.Message)
			}
			data := contactpartial.FormData{
				Name:    input.Name,
				Email:   input.Email,
				Message: input.Message,
				Errors:  fieldErrors,
			}
			return render.FragmentWithStatus(422, contactpartial.ContactForm(vr, submitURL, data))
		}
		return web.Response{}, err
	}

	if err := h.svc.Submit(c.Context(), input, c.Request.RemoteAddr()); err != nil {
		if errors.Is(err, ErrRateLimited) {
			return web.Response{}, web.ErrTooManyRequests(time.Minute)
		}
		return web.Response{}, web.ErrUnavailable("Unable to send your message right now. Please try again later.", err)
	}

	return render.Fragment(contactpartial.ContactForm(vr, submitURL, contactpartial.FormData{Sent: true}))
}
