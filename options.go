package buffalo

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/gobuffalo/envy"
	"github.com/gobuffalo/logger"
	"github.com/gorilla/sessions"
	"github.com/thegodwinproject/buffalo/internal/defaults"
	"github.com/thegodwinproject/buffalo/worker"
)

// Options are used to configure and define how your application should run.
type Options struct {
	Name string `json:"name"`
	// Addr is the bind address provided to http.Server. Default is "127.0.0.1:3000"
	// Can be set using ENV vars "ADDR" and "PORT".
	Addr string `json:"addr"`
	// Host that this application will be available at. Default is "http://127.0.0.1:[$PORT|3000]".
	Host string `json:"host"`

	// Env is the "environment" in which the App is running. Default is "development".
	Env string `json:"env"`

	// LogLvl defaults to logger.DebugLvl.
	LogLvl logger.Level `json:"log_lvl"`
	// Logger to be used with the application. A default one is provided.
	Logger Logger `json:"-"`

	// MethodOverride allows for changing of the request method type. See the default
	// implementation at buffalo.MethodOverride
	MethodOverride http.HandlerFunc `json:"-"`

	// SessionStore is the `github.com/gorilla/sessions` store used to back
	// the session. It defaults to use a cookie store and the ENV variable
	// `SESSION_SECRET`.
	SessionStore sessions.Store `json:"-"`
	// SessionName is the name of the session cookie that is set. This defaults
	// to "_buffalo_session".
	SessionName string `json:"session_name"`

	// Timeout in second for ongoing requests when shutdown the server.
	// The default value is 60.
	TimeoutSecondShutdown int `json:"timeout_second_shutdown"`

	// Worker implements the Worker interface and can process tasks in the background.
	// Default is "github.com/gobuffalo/worker.Simple.
	Worker worker.Worker `json:"-"`
	// WorkerOff tells App.Start() whether to start the Worker process or not. Default is "false".
	WorkerOff bool `json:"worker_off"`

	// PreHandlers are http.Handlers that are called between the http.Server
	// and the buffalo Application.
	PreHandlers []http.Handler `json:"-"`
	// PreWare takes an http.Handler and returns an http.Handler
	// and acts as a pseudo-middleware between the http.Server and
	// a Buffalo application.
	PreWares []PreWare `json:"-"`

	// CompressFiles enables gzip compression of static files served by ServeFiles using
	// gorilla's CompressHandler (https://godoc.org/github.com/gorilla/handlers#CompressHandler).
	// Default is "false".
	CompressFiles bool `json:"compress_files"`

	Prefix  string          `json:"prefix"`
	Context context.Context `json:"-"`

	cancel context.CancelFunc
}

// PreWare takes an http.Handler and returns an http.Handler
// and acts as a pseudo-middleware between the http.Server and
// a Buffalo application.
type PreWare func(http.Handler) http.Handler

// NewOptions returns a new Options instance with sensible defaults
func NewOptions() Options {
	return optionsWithDefaults(Options{})
}

func optionsWithDefaults(opts Options) Options {
	opts.Env = defaults.String(opts.Env, envy.Get("GO_ENV", "development"))
	opts.Name = defaults.String(opts.Name, "/")
	addr := "0.0.0.0"
	if opts.Env == "development" {
		addr = "127.0.0.1"
	}
	envAddr := envy.Get("ADDR", addr)

	if strings.HasPrefix(envAddr, "unix:") {
		// UNIX domain socket doesn't have a port
		opts.Addr = envAddr
	} else {
		// TCP case
		opts.Addr = defaults.String(opts.Addr, fmt.Sprintf("%s:%s", envAddr, envy.Get("PORT", "3000")))
	}
	opts.Host = defaults.String(opts.Host, envy.Get("HOST", fmt.Sprintf("http://127.0.0.1:%s", envy.Get("PORT", "3000"))))

	if opts.PreWares == nil {
		opts.PreWares = []PreWare{}
	}
	if opts.PreHandlers == nil {
		opts.PreHandlers = []http.Handler{}
	}

	if opts.Context == nil {
		opts.Context = context.Background()
	}
	opts.Context, opts.cancel = context.WithCancel(opts.Context)

	if opts.Logger == nil {
		if lvl, err := envy.MustGet("LOG_LEVEL"); err == nil {
			opts.LogLvl, err = logger.ParseLevel(lvl)
			if err != nil {
				opts.LogLvl = logger.DebugLevel
			}
		}

		if opts.LogLvl == 0 {
			opts.LogLvl = logger.DebugLevel
		}

		opts.Logger = logger.New(opts.LogLvl)
	}

	if opts.SessionStore == nil {
		secret := envy.Get("SESSION_SECRET", "")

		if secret == "" && (opts.Env == "development" || opts.Env == "test") {
			secret = "buffalo-secret"
		}

		// In production a SESSION_SECRET must be set!
		if secret == "" {
			opts.Logger.Warn("Unless you set SESSION_SECRET env variable, your session storage is not protected!")
		}

		cookieStore := sessions.NewCookieStore([]byte(secret))

		//Cookie secure attributes, see: https://www.owasp.org/index.php/Testing_for_cookies_attributes_(OTG-SESS-002)
		cookieStore.Options.HttpOnly = true
		if opts.Env == "production" {
			cookieStore.Options.Secure = true
		}

		opts.SessionStore = cookieStore
	}
	opts.SessionName = defaults.String(opts.SessionName, "_buffalo_session")

	if opts.Worker == nil {
		w := worker.NewSimpleWithContext(opts.Context)
		w.Logger = opts.Logger
		opts.Worker = w
	}

	opts.TimeoutSecondShutdown = defaults.Int(opts.TimeoutSecondShutdown, 60)

	return opts
}
