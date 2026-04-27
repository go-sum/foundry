package validate

import (
	"fmt"
	"mime"
	"sync"

	"github.com/gorilla/schema"

	"github.com/go-sum/foundry/pkg/web"
)

var (
	formDecoder     *schema.Decoder
	formDecoderOnce sync.Once
)

func getFormDecoder() *schema.Decoder {
	formDecoderOnce.Do(func() {
		d := schema.NewDecoder()
		d.SetAliasTag("form")
		d.IgnoreUnknownKeys(true)
		d.ZeroEmpty(true)
		formDecoder = d
	})
	return formDecoder
}

// Bind parses the request body into dest and validates it using v.
// Dispatches on Content-Type:
//   - application/json              → req.JSON(dest)
//   - application/x-www-form-urlencoded | multipart/form-data → form decode
//   - anything else                 → web.ErrUnsupportedMedia
//
// After decoding, v.Struct(dest) is called. Validation errors are returned as
// a *web.Error via ToWebError; schema decode errors are mapped by mapSchemaError.
func Bind(v Validator, req web.Request, dest any) error {
	ct := req.Headers.Get("Content-Type")
	mediaType, _, err := mime.ParseMediaType(ct)
	if err != nil {
		return web.ErrUnsupportedMedia(fmt.Sprintf("invalid Content-Type: %s", ct))
	}

	switch mediaType {
	case "application/json":
		if err := req.JSON(dest); err != nil {
			return err
		}

	case "application/x-www-form-urlencoded", "multipart/form-data":
		fd, err := req.FormData()
		if err != nil {
			return err
		}
		if err := getFormDecoder().Decode(dest, fd.Values); err != nil {
			return mapSchemaError(err)
		}

	default:
		return web.ErrUnsupportedMedia(fmt.Sprintf("unsupported Content-Type: %s", mediaType))
	}

	if verr := v.Struct(dest); verr != nil {
		if we := ToWebError(verr); we != nil {
			return we
		}
		return verr
	}
	return nil
}

// mapSchemaError converts gorilla/schema decode errors into *web.Error values.
func mapSchemaError(err error) error {
	if me, ok := err.(schema.MultiError); ok {
		var errs Errors
		for k := range me {
			errs = append(errs, FieldError{
				Field:   k,
				Tag:     "conversion",
				Message: "invalid value",
			})
		}
		return errs.ToWebError()
	}
	if ce, ok := err.(schema.ConversionError); ok {
		return web.ErrBadRequest(fmt.Sprintf("invalid value for field %s", ce.Key))
	}
	return web.ErrBadRequest(err.Error())
}
