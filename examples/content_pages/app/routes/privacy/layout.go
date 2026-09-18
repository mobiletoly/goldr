// Copyright 2026 Toly Pochkin
// SPDX-License-Identifier: Apache-2.0

package privacy

import (
	"net/http"

	"github.com/a-h/templ"
	"github.com/mobiletoly/goldr"
)

func Layout(r *http.Request, ctx goldr.LayoutContext) templ.Component {
	return LayoutView(requestContext(r), ctx.Child)
}
