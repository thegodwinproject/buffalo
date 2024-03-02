package mail

import (
	"html/template"
	"net/http"
	"testing"

	"github.com/gobuffalo/httptest"
	"github.com/stretchr/testify/require"
	"github.com/thegodwinproject/buffalo"
	"github.com/thegodwinproject/buffalo/render"
)

func Test_NewFromData(t *testing.T) {
	r := require.New(t)
	m := NewFromData(map[string]interface{}{
		"foo": "bar",
	})
	r.Equal("bar", m.Data["foo"])
}

func Test_New(t *testing.T) {
	r := require.New(t)

	var m Message
	app := buffalo.New(buffalo.Options{})
	app.GET("/", func(c buffalo.Context) error {
		c.Set("foo", "bar")
		m = New(c)
		return c.Render(http.StatusOK, render.String(""))
	})
	w := httptest.New(app)
	w.HTML("/").Get()

	r.NotNil(m)
	r.Equal("bar", m.Data["foo"])
	rp, ok := m.Data["rootPath"].(buffalo.RouteHelperFunc)
	r.True(ok)
	x, err := rp(map[string]interface{}{})
	r.NoError(err)
	r.Equal(template.HTML("/"), x)
}
