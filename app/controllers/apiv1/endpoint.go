package apiv1

import (
	"maps"
	"net/http"
	"reflect"
	"regexp"

	"github.com/AlexGalhardo/vukem/api"
	"github.com/AlexGalhardo/vukem/router"
	"github.com/AlexGalhardo/vukem/web"
	"github.com/AlexGalhardo/vukem/z"
)

var paramRe = regexp.MustCompile(`\{([A-Za-z_][A-Za-z0-9_]*)\}`)

// handle registers a typed endpoint, like api.Register, but writes the
// output with web.JSONWith so z also checks every response against Out in
// development and tests (a contract break is a 500, never bad data).
// In is read from the query string (GET, DELETE) or the JSON body, plus
// the path parameters, and validated with z.For[In].
//
// workaround(vukem v0.23.0): api.Register writes c.JSON, so responses are not checked like web.JSONWith.
func handle[In, Out any](g *router.Group, method, path string, op api.Op, fn func(*web.Ctx, In) (Out, error)) *router.Route {
	inType, outType := reflect.TypeFor[In](), reflect.TypeFor[Out]()
	noInput := inType.NumField() == 0
	noOutput := outType.NumField() == 0
	bodyless := method == http.MethodGet || method == http.MethodDelete

	var in z.Schema[In]
	if !noInput {
		in = z.For[In]()
	}

	out := z.For[Out]()

	status := op.Status
	switch {
	case status != 0:
	case noOutput:
		status = http.StatusNoContent
	case method == http.MethodPost:
		status = http.StatusCreated
	default:
		status = http.StatusOK
	}

	var params []string

	h := func(c *web.Ctx) error {
		var v In

		if !noInput {
			input := maps.Clone(c.QueryInput())
			if !bodyless {
				var err error
				if input, err = c.Input(); err != nil {
					return err
				}
			}

			for _, p := range params {
				input[p] = c.Param(p)
			}

			var err error
			if v, err = web.ParseInput(in, input); err != nil {
				return err
			}
		}

		res, err := fn(c, v)
		if err != nil {
			return err
		}

		if noOutput {
			return c.NoContent()
		}

		return web.JSONWith(c, status, out, res)
	}

	rt := g.Match([]string{method}, path, h)[0].Name(op.Name)
	for _, m := range paramRe.FindAllStringSubmatch(rt.Path(), -1) {
		params = append(params, m[1])
	}

	op.Status = status

	return rt.Doc(&api.Operation{Op: op, Method: method, Path: rt.Path(), Params: params, In: inType, Out: outType})
}
