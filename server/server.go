// Copyright 2019 The LUCI Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package server implements an environment for running LUCI servers.
//
// It interprets command line flags and initializes the serving environment with
// the following base services available via context.Context:
//   * github.com/tetrafolium/luci-go/common/logging: Logging.
//   * github.com/tetrafolium/luci-go/common/trace: Tracing.
//   * github.com/tetrafolium/luci-go/server/caching: Process cache.
//   * github.com/tetrafolium/luci-go/server/auth: Making authenticated calls.
//
// Other functionality is optional and provided by modules (objects implementing
// module.Module interface). They should be passed to the server when it starts.
//
// Usage example:
//
//   func main() {
//     // When copying, please pick only modules you need.
//     modules := []module.Module{
//       gaeemulation.NewModuleFromFlags(),
//       limiter.NewModuleFromFlags(),
//       redisconn.NewModuleFromFlags(),
//       secrets.NewModuleFromFlags(),
//     }
//     server.Main(nil, modules, func(srv *server.Server) error {
//       // Initialize global state, change root context (if necessary).
//       if err := initializeGlobalStuff(srv.Context); err != nil {
//         return err
//       }
//       srv.Context = injectGlobalStuff(srv.Context)
//
//       // Install regular HTTP routes.
//       srv.Routes.GET("/", router.MiddlewareChain{}, func(c *router.Context) {
//         // ...
//       })
//
//       // Install pRPC services.
//       servicepb.RegisterSomeServer(srv.PRPC, &SomeServer{})
//       return nil
//     })
//   }
package server

import (
	"context"
	cryptorand "crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"flag"
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"net/http/pprof"
	"os"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"cloud.google.com/go/profiler"
	"golang.org/x/oauth2"
	"google.golang.org/api/option"
	"google.golang.org/grpc"

	"contrib.go.opencensus.io/exporter/stackdriver"
	"go.opencensus.io/exporter/stackdriver/propagation"
	octrace "go.opencensus.io/trace"

	"github.com/tetrafolium/luci-go/common/clock"
	"github.com/tetrafolium/luci-go/common/data/rand/mathrand"
	"github.com/tetrafolium/luci-go/common/data/stringset"
	"github.com/tetrafolium/luci-go/common/errors"
	luciflag "github.com/tetrafolium/luci-go/common/flag"
	"github.com/tetrafolium/luci-go/common/iotools"
	"github.com/tetrafolium/luci-go/common/logging"
	"github.com/tetrafolium/luci-go/common/logging/gologger"
	"github.com/tetrafolium/luci-go/common/logging/sdlogger"
	"github.com/tetrafolium/luci-go/common/system/signals"
	"github.com/tetrafolium/luci-go/common/trace"
	tsmoncommon "github.com/tetrafolium/luci-go/common/tsmon"
	"github.com/tetrafolium/luci-go/common/tsmon/metric"
	"github.com/tetrafolium/luci-go/common/tsmon/monitor"
	"github.com/tetrafolium/luci-go/common/tsmon/target"

	"github.com/tetrafolium/luci-go/hardcoded/chromeinfra" // should be used ONLY in Main()

	"github.com/tetrafolium/luci-go/grpc/discovery"
	"github.com/tetrafolium/luci-go/grpc/grpcmon"
	"github.com/tetrafolium/luci-go/grpc/grpcutil"
	"github.com/tetrafolium/luci-go/grpc/prpc"

	"github.com/tetrafolium/luci-go/web/gowrappers/rpcexplorer"

	clientauth "github.com/tetrafolium/luci-go/auth"

	"github.com/tetrafolium/luci-go/server/auth"
	"github.com/tetrafolium/luci-go/server/auth/authdb"
	"github.com/tetrafolium/luci-go/server/auth/authdb/dump"
	"github.com/tetrafolium/luci-go/server/auth/signing"
	"github.com/tetrafolium/luci-go/server/caching"
	"github.com/tetrafolium/luci-go/server/experiments"
	"github.com/tetrafolium/luci-go/server/internal"
	"github.com/tetrafolium/luci-go/server/middleware"
	"github.com/tetrafolium/luci-go/server/module"
	"github.com/tetrafolium/luci-go/server/portal"
	"github.com/tetrafolium/luci-go/server/router"
	"github.com/tetrafolium/luci-go/server/secrets"
	"github.com/tetrafolium/luci-go/server/tsmon"
)

const (
	// Path of the health check endpoint.
	healthEndpoint = "/healthz"
	// Log a warning if health check is slower than this.
	healthTimeLogThreshold = 50 * time.Millisecond
)

var (
	versionMetric = metric.NewString(
		"server/version",
		"Version of the running container image (taken from -container-image-id).",
		nil)
)

// cloudRegionFromGAERegion maps GAE region codes (e.g. `s`) to corresponding
// cloud regions (e.g. `us-central1`), which may be defined as regions where GAE
// creates resources associated with the app, such as Task Queues or Flex VMs.
//
// Sadly this mapping is not documented, thus the below map is incomplete. Feel
// free to modify it if you deployed to some new GAE region.
//
// This mapping is unused if `-cloud-region` flag is passed explicitly.
var cloudRegionFromGAERegion = map[string]string{
	"e": "europe-west1",
	"g": "europe-west2",
	"h": "europe-west3",
	"m": "us-west2",
	"p": "us-east1",
	"s": "us-central1",
}

// Main initializes the server and runs its serving loop until SIGTERM.
//
// Registers all options in the default flag set and uses `flag.Parse` to parse
// them. If 'opts' is nil, the default options will be used.
//
// Additionally recognizes GAE_APPLICATION env var as an indicator that the
// server is running on GAE. This slightly tweaks its behavior to match what GAE
// expects from servers.
//
// On errors, logs them and aborts the process with non-zero exit code.
func Main(opts *Options, mods []module.Module, init func(srv *Server) error) {
	mathrand.SeedRandomly()
	if opts == nil {
		opts = &Options{
			ClientAuth: chromeinfra.DefaultAuthOptions(),
		}
	}

	opts.Register(flag.CommandLine)
	flag.Parse()
	opts.FromGAEEnv()

	srv, err := New(context.Background(), *opts, mods)
	if err != nil {
		srv.Fatal(err)
	}
	if init != nil {
		if err = init(srv); err != nil {
			srv.Fatal(err)
		}
	}
	if err = srv.ListenAndServe(); err != nil {
		srv.Fatal(err)
	}
}

// Options are used to configure the server.
//
// Most of them are exposed as command line flags (see Register implementation).
// Some (mostly GAE-specific) are only settable through code or are derived from
// the environment.
type Options struct {
	Prod     bool   // set when running in production (not on a dev workstation)
	GAE      bool   // set when running on GAE, implies Prod
	Hostname string // used for logging and metric fields, default is os.Hostname

	HTTPAddr  string // address to bind the main listening socket to
	AdminAddr string // address to bind the admin socket to, ignored on GAE

	ShutdownDelay time.Duration // how long to wait after SIGTERM before shutting down

	ClientAuth      clientauth.Options // base settings for client auth options
	TokenCacheDir   string             // where to cache auth tokens (optional)
	AuthDBPath      string             // if set, load AuthDB from a file
	AuthServiceHost string             // hostname of an Auth Service to use
	AuthDBDump      string             // Google Storage path to fetch AuthDB dumps from
	AuthDBSigner    string             // service account that signs AuthDB dumps

	CloudProject string // name of the hosting Google Cloud Project
	CloudRegion  string // name of the hosting Google Cloud region

	TraceSampling string // what portion of traces to upload to Stackdriver (ignored on GAE)

	TsMonAccount     string // service account to flush metrics as
	TsMonServiceName string // service name of tsmon target
	TsMonJobName     string // job name of tsmon target

	ProfilingDisable   bool   // set to true to explicitly disable Stackdriver Profiler
	ProfilingServiceID string // service name to associated with profiles in Stackdriver Profiler

	ContainerImageID string // ID of the container image with this binary, for logs (optional)

	EnableExperiments []string // names of github.com/tetrafolium/luci-go/server/experiments to enable

	testSeed           int64                   // used to seed rng in tests
	testStdout         sdlogger.LogEntryWriter // mocks stdout in tests
	testStderr         sdlogger.LogEntryWriter // mocks stderr in tests
	testListeners      map[string]net.Listener // addr => net.Listener, for tests
	testAuthDB         authdb.DB               // AuthDB to use in tests
	testDisableTracing bool                    // don't install a tracing backend
}

// Register registers the command line flags.
func (o *Options) Register(f *flag.FlagSet) {
	if o.HTTPAddr == "" {
		o.HTTPAddr = "127.0.0.1:8800"
	}
	if o.AdminAddr == "" {
		o.AdminAddr = "127.0.0.1:8900"
	}
	if o.ShutdownDelay == 0 {
		o.ShutdownDelay = 15 * time.Second
	}
	f.BoolVar(&o.Prod, "prod", o.Prod, "Switch the server into production mode")
	f.StringVar(&o.HTTPAddr, "http-addr", o.HTTPAddr, "Address to bind the main listening socket to or '-' to disable")
	f.StringVar(&o.AdminAddr, "admin-addr", o.AdminAddr, "Address to bind the admin socket to or '-' to disable")
	f.DurationVar(&o.ShutdownDelay, "shutdown-delay", o.ShutdownDelay, "How long to wait after SIGTERM before shutting down")
	f.StringVar(
		&o.ClientAuth.ServiceAccountJSONPath,
		"service-account-json",
		o.ClientAuth.ServiceAccountJSONPath,
		"Path to a JSON file with service account private key",
	)
	f.StringVar(
		&o.ClientAuth.ActAsServiceAccount,
		"act-as",
		o.ClientAuth.ActAsServiceAccount,
		"Act as this service account",
	)
	f.StringVar(
		&o.TokenCacheDir,
		"token-cache-dir",
		o.TokenCacheDir,
		"Where to cache auth tokens (optional)",
	)
	f.StringVar(
		&o.AuthDBPath,
		"auth-db-path",
		o.AuthDBPath,
		"If set, load AuthDB text proto from this file (incompatible with -auth-service-host)",
	)
	f.StringVar(
		&o.AuthServiceHost,
		"auth-service-host",
		o.AuthServiceHost,
		"Hostname of an Auth Service to use (incompatible with -auth-db-path)",
	)
	f.StringVar(
		&o.AuthDBDump,
		"auth-db-dump",
		o.AuthDBDump,
		"Google Storage path to fetch AuthDB dumps from. Default is gs://<auth-service-host>/auth-db",
	)
	f.StringVar(
		&o.AuthDBSigner,
		"auth-db-signer",
		o.AuthDBSigner,
		"Service account that signs AuthDB dumps. Default is derived from -auth-service-host if it is *.appspot.com",
	)
	f.StringVar(
		&o.CloudProject,
		"cloud-project",
		o.CloudProject,
		"Name of hosting Google Cloud Project (optional)",
	)
	f.StringVar(
		&o.CloudRegion,
		"cloud-region",
		o.CloudRegion,
		"Name of hosting Google Cloud region, e.g. 'us-central1' (optional)",
	)
	f.StringVar(
		&o.TraceSampling,
		"trace-sampling",
		o.TraceSampling,
		"What portion of traces to upload to Stackdriver. Either a percent (i.e. '0.1%') or a QPS (i.e. '1qps'). Ignored on GAE. Default is 0.1qps.",
	)
	f.StringVar(
		&o.TsMonAccount,
		"ts-mon-account",
		o.TsMonAccount,
		"Collect and flush tsmon metrics using this account for auth (disables tsmon if not set)",
	)
	f.StringVar(
		&o.TsMonServiceName,
		"ts-mon-service-name",
		o.TsMonServiceName,
		"Service name of tsmon target (disables tsmon if not set)",
	)
	f.StringVar(
		&o.TsMonJobName,
		"ts-mon-job-name",
		o.TsMonJobName,
		"Job name of tsmon target (disables tsmon if not set)",
	)
	f.BoolVar(
		&o.ProfilingDisable,
		"profiling-disable",
		o.ProfilingDisable,
		"Pass to explicitly disable Stackdriver Profiler",
	)
	f.StringVar(
		&o.ProfilingServiceID,
		"profiling-service-id",
		o.ProfilingServiceID,
		"Service name to associated with profiles in Stackdriver Profiler. Defaults to the value of -ts-mon-job-name.",
	)
	f.StringVar(
		&o.ContainerImageID,
		"container-image-id",
		o.ContainerImageID,
		"ID of the container image with this binary, for logs (optional)",
	)

	// See github.com/tetrafolium/luci-go/server/experiments.
	f.Var(luciflag.StringSlice(&o.EnableExperiments), "enable-experiment",
		`A name of the experiment to enable. May be repeated.`)
}

// FromGAEEnv uses the GAE_APPLICATION env var (and other GAE-specific env vars)
// to configure the server for the GAE environment.
//
// Does nothing if GAE_APPLICATION is not set.
//
// Equivalent to passing the following flags:
//   -prod
//   -http-addr 0.0.0.0:${PORT}
//   -admin-addr -
//   -shutdown-delay 0s
//   -cloud-project ${GOOGLE_CLOUD_PROJECT}
//   -cloud-region <derived from the region code in GAE_APPLICATION>
//   -service-account-json :gce
//   -ts-mon-service-name ${GOOGLE_CLOUD_PROJECT}
//   -ts-mon-job-name ${GAE_SERVICE}
//
// Additionally the hostname and -container-image-id (used in metric and trace
// fields) are derived from available GAE_* env vars to be semantically similar
// to what they represent in the GKE environment.
//
// Note that a mapping between a region code in GAE_APPLICATION and
// the corresponding cloud region is not documented anywhere, so if you see
// warnings when your app starts up either update the code to recognize your
// region code or pass '-cloud-region' argument explicitly in app.yaml.
//
// See https://cloud.google.com/appengine/docs/standard/go/runtime.
func (o *Options) FromGAEEnv() {
	appID := os.Getenv("GAE_APPLICATION") // e.g. "s~something"
	if appID == "" {
		return
	}
	o.GAE = true
	o.Prod = true
	o.Hostname = uniqueGAEHostname()
	o.HTTPAddr = fmt.Sprintf("0.0.0.0:%s", os.Getenv("PORT"))
	o.AdminAddr = "-"
	o.ShutdownDelay = 0
	o.CloudProject = os.Getenv("GOOGLE_CLOUD_PROJECT")
	if o.CloudRegion == "" {
		o.CloudRegion = cloudRegionFromGAERegion[strings.Split(appID, "~")[0]]
	}
	o.ClientAuth.ServiceAccountJSONPath = clientauth.GCEServiceAccount
	o.TsMonServiceName = os.Getenv("GOOGLE_CLOUD_PROJECT")
	o.TsMonJobName = os.Getenv("GAE_SERVICE")
	o.ContainerImageID = fmt.Sprintf("appengine/%s/%s:%s",
		os.Getenv("GOOGLE_CLOUD_PROJECT"),
		os.Getenv("GAE_SERVICE"),
		os.Getenv("GAE_VERSION"),
	)
}

// uniqueGAEHostname uses GAE_* env vars to derive a unique enough string that
// is used as a hostname in monitoring metrics.
func uniqueGAEHostname() string {
	// GAE_INSTANCE is huge, hash it to get a small reasonably unique string.
	id := sha256.Sum256([]byte(os.Getenv("GAE_INSTANCE")))
	return fmt.Sprintf("%s-%s-%s",
		os.Getenv("GAE_SERVICE"),
		os.Getenv("GAE_DEPLOYMENT_ID"),
		hex.EncodeToString(id[:])[:16],
	)
}

// imageVersion extracts image tag or digest from ContainerImageID.
//
// This is eventually reported as a value of 'server/version' metric.
//
// Returns "unknown" if ContainerImageID is empty or malformed.
func (o *Options) imageVersion() string {
	// Recognize "<path>@sha256:<digest>" and "<path>:<tag>".
	idx := strings.LastIndex(o.ContainerImageID, "@")
	if idx == -1 {
		idx = strings.LastIndex(o.ContainerImageID, ":")
	}
	if idx == -1 {
		return "unknown"
	}
	return o.ContainerImageID[idx+1:]
}

// userAgent derives a user-agent like string identifying the server.
func (o *Options) userAgent() string {
	return fmt.Sprintf("LUCI-Server (service: %s; job: %s; ver: %s);", o.TsMonServiceName, o.TsMonJobName, o.imageVersion())
}

// shouldEnableTracing is true if options indicate we should enable tracing.
func (o *Options) shouldEnableTracing() bool {
	switch {
	case o.CloudProject == "":
		return false // nowhere to upload traces to
	case !o.Prod && o.TraceSampling == "":
		return false // in dev mode don't upload samples by default
	default:
		return !o.testDisableTracing
	}
}

// hostOptions constructs HostOptions for module.Initialize(...).
func (o *Options) hostOptions() module.HostOptions {
	return module.HostOptions{
		Prod:         o.Prod,
		GAE:          o.GAE,
		CloudProject: o.CloudProject,
		CloudRegion:  o.CloudRegion,
	}
}

// Server is responsible for initializing and launching the serving environment.
//
// Generally assumed to be a singleton: do not launch multiple Server instances
// within the same process, use AddPort instead if you want to expose multiple
// ports.
//
// Doesn't do TLS. Should be sitting behind a load balancer that terminates
// TLS.
type Server struct {
	// Context is the root context used by all requests and background activities.
	//
	// Can be replaced (by a derived context) before ListenAndServe call, for
	// example to inject values accessible to all request handlers.
	Context context.Context

	// Routes is a router for requests hitting HTTPAddr port.
	//
	// This router is used for all requests whose Host header does not match any
	// specially registered per-host routers (see VirtualHost). Normally, there
	// are no such per-host routers, so usually Routes is used for all requests.
	//
	// This router is also accessible to the server modules and they can install
	// routes into it.
	//
	// Should be populated before ListenAndServe call.
	Routes *router.Router

	// PRPC is pRPC server with APIs exposed on HTTPAddr port via Routes router.
	//
	// Should be populated before ListenAndServe call.
	PRPC *prpc.Server

	// Options is a copy of options passed to New.
	Options Options

	startTime   time.Time    // for calculating uptime for /healthz
	lastReqTime atomic.Value // time.Time when the last request started

	stdout sdlogger.LogEntryWriter // for logging to stdout, nil in dev mode
	stderr sdlogger.LogEntryWriter // for logging to stderr, nil in dev mode

	mainPort *Port // pre-registered main port, see initMainPort

	mu      sync.Mutex    // protects fields below
	ports   []*Port       // all non-dummy ports (each one hosts an HTTP server)
	started bool          // true inside and after ListenAndServe
	stopped bool          // true inside and after Shutdown
	ready   chan struct{} // closed right before starting the serving loop
	done    chan struct{} // closed after Shutdown returns

	// See RegisterUnaryServerInterceptor and ListenAndServe.
	unaryInterceptors []grpc.UnaryServerInterceptor

	rndM sync.Mutex // protects rnd
	rnd  *rand.Rand // used to generate trace and operation IDs

	bgrDone chan struct{}  // closed to stop background activities
	bgrWg   sync.WaitGroup // waits for RunInBackground goroutines to stop

	cleanupM sync.Mutex // protects 'cleanup' and actual cleanup critical section
	cleanup  []func(context.Context)

	tsmon   *tsmon.State    // manages flushing of tsmon metrics
	sampler octrace.Sampler // trace sampler to use for top level spans

	authM        sync.RWMutex
	authPerScope map[string]scopedAuth // " ".join(scopes) => ...
	authDB       atomic.Value          // last known good authdb.DB instance

	runningAs string // email of an account the server runs as
}

// scopedAuth holds TokenSource and Authenticator that produced it.
type scopedAuth struct {
	source oauth2.TokenSource
	authen *clientauth.Authenticator
}

// moduleHostImpl implements module.Host via server.Server.
//
// Just a tiny wrapper to make sure modules consume only curated limited set of
// the server API and do not retain the pointer to the server.
type moduleHostImpl struct {
	srv     *Server
	invalid bool
}

func (h *moduleHostImpl) panicIfInvalid() {
	if h.invalid {
		panic("module.Host must not be used outside of Initialize")
	}
}

func (h *moduleHostImpl) Routes() *router.Router {
	h.panicIfInvalid()
	return h.srv.Routes
}

func (h *moduleHostImpl) RunInBackground(activity string, f func(context.Context)) {
	h.panicIfInvalid()
	h.srv.RunInBackground(activity, f)
}

func (h *moduleHostImpl) RegisterCleanup(cb func(context.Context)) {
	h.panicIfInvalid()
	h.srv.RegisterCleanup(cb)
}

func (h *moduleHostImpl) RegisterUnaryServerInterceptor(intr grpc.UnaryServerInterceptor) {
	h.panicIfInvalid()
	h.srv.RegisterUnaryServerInterceptor(intr)
}

// New constructs a new server instance.
//
// It hosts one or more HTTP servers and starts and stops them in unison. It is
// also responsible for preparing contexts for incoming requests.
//
// The given context will become the root context of the server and will be
// inherited by all handlers.
//
// On errors returns partially initialized server (always non-nil). At least
// its logging will be configured and can be used to report the error. Trying
// to use such partially initialized server for anything else is undefined
// behavior.
func New(ctx context.Context, opts Options, mods []module.Module) (srv *Server, err error) {
	seed := opts.testSeed
	if seed == 0 {
		if err := binary.Read(cryptorand.Reader, binary.BigEndian, &seed); err != nil {
			panic(err)
		}
	}

	// Do this very early, so that various transports created during the
	// initialization are already wrapped with tracing. The rest of the tracing
	// infra (e.g. actual uploads) is initialized later in initTracing.
	if opts.shouldEnableTracing() {
		internal.EnableOpenCensusTracing()
	}

	srv = &Server{
		Context:      ctx,
		Options:      opts,
		startTime:    clock.Now(ctx).UTC(),
		ready:        make(chan struct{}),
		done:         make(chan struct{}),
		rnd:          rand.New(rand.NewSource(seed)),
		bgrDone:      make(chan struct{}),
		authPerScope: map[string]scopedAuth{},
		sampler:      octrace.NeverSample(),
	}

	// Cleanup what we can on failures.
	defer func() {
		if err != nil {
			srv.runCleanup()
		}
	}()

	// Logging is needed to report any errors during the early initialization.
	srv.initLogging()

	logging.Infof(srv.Context, "Server starting...")
	if srv.Options.ContainerImageID != "" {
		logging.Infof(srv.Context, "Container image is %s", srv.Options.ContainerImageID)
	}

	// Need the hostname (e.g. pod name on k8s) for logs and metrics.
	if srv.Options.Hostname == "" {
		srv.Options.Hostname, err = os.Hostname()
		if err != nil {
			return srv, errors.Annotate(err, "failed to get own hostname").Err()
		}
	}

	// On k8s log pod IPs too, this is useful when debugging k8s routing.
	if !srv.Options.GAE {
		logging.Infof(srv.Context, "Running on %s (%s)", srv.Options.Hostname, networkAddrsForLog())
	} else {
		logging.Infof(srv.Context, "Running on %s", srv.Options.Hostname)
		if srv.Options.CloudRegion == "" {
			logging.Warningf(srv.Context, "Could not figure out the primary Cloud region based "+
				"on the region code in GAE_APPLICATION %q, consider passing the region name "+
				"via -cloud-region flag explicitly", os.Getenv("GAE_APPLICATION"))
		} else {
			logging.Infof(srv.Context, "Cloud region is %s", srv.Options.CloudRegion)
		}
	}

	// Log enabled experiments, warn if some of them are unknown now.
	var exps []experiments.ID
	for _, name := range opts.EnableExperiments {
		if exp, ok := experiments.GetByName(name); ok {
			logging.Infof(ctx, "Enabling experiment %q", name)
			exps = append(exps, exp)
		} else {
			logging.Warningf(ctx, "Skipping unknown experiment %q", name)
		}
	}
	srv.Context = experiments.Enable(srv.Context, exps...)

	// Configure base server subsystems by injecting them into the root context
	// inherited later by all requests.
	srv.Context = caching.WithProcessCacheData(srv.Context, caching.NewProcessCacheData())
	if err := srv.initAuth(); err != nil {
		return srv, errors.Annotate(err, "failed to initialize auth").Err()
	}
	if err := srv.initTSMon(); err != nil {
		return srv, errors.Annotate(err, "failed to initialize tsmon").Err()
	}
	if err := srv.initTracing(); err != nil {
		return srv, errors.Annotate(err, "failed to initialize tracing").Err()
	}
	if err := srv.initProfiling(); err != nil {
		return srv, errors.Annotate(err, "failed to initialize profiling").Err()
	}
	if err := srv.initMainPort(); err != nil {
		return srv, errors.Annotate(err, "failed to initialize the main port").Err()
	}
	if err := srv.initAdminPort(); err != nil {
		return srv, errors.Annotate(err, "failed to initialize the admin port").Err()
	}

	// Initialize all extra modules. Note that names aren't really used yet, but
	// we still want to guarantee they are unique to avoid issues in the future.
	seen := stringset.New(len(mods))
	host := moduleHostImpl{srv: srv}
	for _, m := range mods {
		if !seen.Add(m.Name()) {
			return srv, errors.Reason("duplicate module %q", m.Name()).Err()
		}
		switch ctx, err := m.Initialize(srv.Context, &host, srv.Options.hostOptions()); {
		case err != nil:
			return srv, errors.Annotate(err, "failed to initialize module %q", m.Name()).Err()
		case ctx != nil:
			srv.Context = ctx
		}
	}
	host.invalid = true // break `host` to make sure modules do not retain it

	return srv, nil
}

// AddPort prepares an additional serving HTTP port.
//
// Can be used to open more listening HTTP ports (in addition to opts.HTTPAddr
// and opts.AdminAddr). The returned Port object can be used to populate the
// router that serves requests hitting the added port.
//
// If opts.ListenAddr is '-', a dummy port will be added: it is a valid *Port
// object, but it is not actually exposed as a listening TCP socket. This is
// useful to disable listening ports without changing any code.
//
// Should be called before ListenAndServe (panics otherwise).
func (s *Server) AddPort(opts PortOptions) *Port {
	port := &Port{
		Routes: s.newRouter(opts),
		parent: s,
		opts:   opts,
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.started {
		s.Fatal(errors.Reason("the server has already been started").Err())
	}
	if opts.ListenAddr != "-" {
		s.ports = append(s.ports, port)
	}
	return port
}

// VirtualHost returns a router (registering it if necessary) used for requests
// that hit the main port (opts.HTTPAddr) and have the given Host header.
//
// Should be used in rare cases when the server is exposed through multiple
// domain names and requests should be routed differently based on what domain
// was used. If your server is serving only one domain name, or you don't care
// what domain name is used to access it, do not use VirtualHost.
//
// Note that requests that match some registered virtual host router won't
// reach the default router (server.Routes), even if the virtual host router
// doesn't have a route for them. Such requests finish with HTTP 404.
//
// Also the router created by VirtualHost is initially completely empty: the
// server and its modules don't install anything into it (there's intentionally
// no mechanism to do this). For that reason VirtualHost should never by used to
// register a router for the "main" domain name: it will make the default
// server.Routes (and all handlers installed there by server modules) useless,
// probably breaking the server. Put routes for the main server functionality
// directly into server.Routes instead, using VirtualHost only for routes that
// critically depend on Host header.
//
// Should be called before ListenAndServe (panics otherwise).
func (s *Server) VirtualHost(host string) *router.Router {
	return s.mainPort.VirtualHost(host)
}

// newRouter creates a Router with the default middleware chain and routes.
func (s *Server) newRouter(opts PortOptions) *router.Router {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.started {
		s.Fatal(errors.Reason("the server has already been started").Err())
	}

	mw := router.NewMiddlewareChain(
		s.rootMiddleware,            // prepares the per-request context
		middleware.WithPanicCatcher, // transforms panics into HTTP 500
	)
	if s.tsmon != nil && !opts.DisableMetrics {
		mw = mw.Extend(s.tsmon.Middleware) // collect HTTP requests metrics
	}

	// Setup middleware chain used by ALL requests.
	r := router.New()
	r.Use(mw)

	// Mandatory health check/readiness probe endpoint.
	r.GET(healthEndpoint, router.MiddlewareChain{}, func(c *router.Context) {
		c.Writer.Write([]byte(s.healthResponse(c.Context)))
	})

	// Add NotFound handler wrapped in our middlewares so that unrecognized
	// requests are at least logged. If we don't do that they'll be handled
	// completely silently and this is very confusing when debugging 404s.
	r.NotFound(router.MiddlewareChain{}, func(c *router.Context) {
		http.NotFound(c.Writer, c.Request)
	})

	return r
}

// RunInBackground launches the given callback in a separate goroutine right
// before starting the serving loop.
//
// If the server is already running, launches it right away. If the server
// fails to start, the goroutines will never be launched.
//
// Should be used for background asynchronous activities like reloading configs.
//
// All logs lines emitted by the callback are annotated with "activity" field
// which can be arbitrary, but by convention has format "<namespace>.<name>",
// where "luci" namespace is reserved for internal activities.
//
// The context passed to the callback is canceled when the server is shutting
// down. It is expected the goroutine will exit soon after the context is
// canceled.
func (s *Server) RunInBackground(activity string, f func(context.Context)) {
	s.bgrWg.Add(1)
	go func() {
		defer s.bgrWg.Done()

		select {
		case <-s.ready:
			// Construct the context after the server is fully initialized. Cancel it
			// as soon as bgrDone is signaled.
			ctx, cancel := context.WithCancel(s.Context)
			if activity != "" {
				ctx = logging.SetField(ctx, "activity", activity)
			}
			defer cancel()
			go func() {
				select {
				case <-s.bgrDone:
					cancel()
				case <-ctx.Done():
				}
			}()
			f(ctx)

		case <-s.bgrDone:
			// the server is closed, no need to run f() anymore
		}
	}()
}

// RegisterUnaryServerInterceptor registers an grpc.UnaryServerInterceptor
// applied to all unary RPCs that hit the server.
//
// Interceptors are chained in order they are registered, i.e. the first
// registered interceptor becomes the outermost. The initial chain already
// contains some base interceptors (e.g. for monitoring) and all interceptors
// registered by server modules. RegisterUnaryServerInterceptor extends this
// chain.
//
// An interceptor set in server.PRPC.UnaryServerInterceptor (if any) is
// automatically registered as the last (innermost) one right before the server
// starts listening for requests in ListenAndServe.
//
// Should be called before ListenAndServe (panics otherwise).
func (s *Server) RegisterUnaryServerInterceptor(intr grpc.UnaryServerInterceptor) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.started {
		s.Fatal(errors.Reason("the server has already been started").Err())
	}
	s.unaryInterceptors = append(s.unaryInterceptors, intr)
}

// ListenAndServe launches the serving loop.
//
// Blocks forever or until the server is stopped via Shutdown (from another
// goroutine or from a SIGTERM handler). Returns nil if the server was shutdown
// correctly or an error if it failed to start or unexpectedly died. The error
// is logged inside.
//
// Should be called only once. Panics otherwise.
func (s *Server) ListenAndServe() error {
	s.mu.Lock()
	if s.started {
		s.mu.Unlock()
		s.Fatal(errors.Reason("the server has already been started").Err())
	}
	s.started = true

	ports := append(make([]*Port, 0, len(s.ports)), s.ports...)

	// Assemble the interceptor chain. Put our base interceptors in front of
	// whatever interceptors were installed by modules and by the user of Server
	// via public s.PRPC.UnaryServerInterceptor.
	interceptors := []grpc.UnaryServerInterceptor{
		grpcmon.UnaryServerInterceptor,
		grpcutil.UnaryServerPanicCatcherInterceptor,
	}
	interceptors = append(interceptors, s.unaryInterceptors...)
	if s.PRPC.UnaryServerInterceptor != nil {
		interceptors = append(interceptors, s.PRPC.UnaryServerInterceptor)
	}

	s.mu.Unlock()

	// Install the interceptor chain.
	s.PRPC.UnaryServerInterceptor = grpcutil.ChainUnaryServerInterceptors(interceptors...)

	// Catch SIGTERM while inside this function. Upon receiving SIGTERM, wait
	// until the pod is removed from the load balancer before actually shutting
	// down and refusing new connections. If we shutdown immediately, some clients
	// may see connection errors, because they are not aware yet the server is
	// closing: Pod shutdown sequence and Endpoints list updates are racing with
	// each other, we want Endpoints list updates to win, i.e. we want the pod to
	// actually be fully alive as long as it is still referenced in Endpoints
	// list. We can't guarantee this, but we can improve chances.
	stop := signals.HandleInterrupt(func() {
		if s.Options.Prod {
			s.waitUntilNotServing()
		}
		s.Shutdown()
	})
	defer stop()

	// Log how long it took from 'New' to the serving loop.
	logging.Infof(s.Context, "Startup done in %s", clock.Now(s.Context).Sub(s.startTime))

	// Unblock all pending RunInBackground goroutines, so they can start.
	close(s.ready)

	// Run serving loops in parallel.
	errs := make(errors.MultiError, len(ports))
	wg := sync.WaitGroup{}
	wg.Add(len(ports))
	for i, port := range ports {
		logging.Infof(s.Context, "Serving %s", port.nameForLog())
		i := i
		port := port
		go func() {
			defer wg.Done()
			if err := s.serveLoop(port.httpServer()); err != http.ErrServerClosed {
				logging.WithError(err).Errorf(s.Context, "Server %s failed", port.nameForLog())
				errs[i] = err
				s.Shutdown() // close all other servers
			}
		}()
	}
	wg.Wait()

	// Per http.Server docs, we end up here *immediately* after Shutdown call was
	// initiated. Some requests can still be in-flight. We block until they are
	// done (as indicated by Shutdown call itself exiting).
	logging.Infof(s.Context, "Waiting for the server to stop...")
	<-s.done
	logging.Infof(s.Context, "The serving loop stopped, running the final cleanup...")
	s.runCleanup()
	logging.Infof(s.Context, "The server has stopped")

	if errs.First() != nil {
		return errs
	}
	return nil
}

// Shutdown gracefully stops the server if it was running.
//
// Blocks until the server is stopped. Can be called multiple times.
func (s *Server) Shutdown() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.stopped {
		return
	}

	logging.Infof(s.Context, "Shutting down the server...")

	// Tell all RunInBackground goroutines to stop.
	close(s.bgrDone)

	// Stop all http.Servers in parallel. Each Shutdown call blocks until the
	// corresponding server is stopped.
	wg := sync.WaitGroup{}
	wg.Add(len(s.ports))
	for _, port := range s.ports {
		port := port
		go func() {
			defer wg.Done()
			port.httpServer().Shutdown(s.Context)
		}()
	}
	wg.Wait()

	// Wait for all background goroutines to stop.
	s.bgrWg.Wait()

	// Notify ListenAndServe that it can exit now.
	s.stopped = true
	close(s.done)
}

// Fatal logs the error and immediately shuts down the process with exit code 3.
//
// No cleanup is performed. Deferred statements are not run. Not recoverable.
func (s *Server) Fatal(err error) {
	errors.Log(s.Context, err)
	os.Exit(3)
}

// healthResponse prepares text/plan response for the health check endpoints.
//
// It additionally contains some easy to obtain information that may help in
// debugging deployments.
func (s *Server) healthResponse(c context.Context) string {
	maybeEmpty := func(s string) string {
		if s == "" {
			return "<unknown>"
		}
		return s
	}
	return strings.Join([]string{
		"OK",
		"",
		"uptime:  " + clock.Now(c).Sub(s.startTime).String(),
		"image:   " + maybeEmpty(s.Options.ContainerImageID),
		"",
		"service: " + maybeEmpty(s.Options.TsMonServiceName),
		"job:     " + maybeEmpty(s.Options.TsMonJobName),
		"host:    " + s.Options.Hostname,
		"",
	}, "\n")
}

// serveLoop binds the socket and launches the serving loop.
//
// Basically srv.ListenAndServe with some testing helpers.
func (s *Server) serveLoop(srv *http.Server) error {
	// If not running tests, let http.Server bind the socket as usual.
	if s.Options.testListeners == nil {
		return srv.ListenAndServe()
	}
	// In test mode the listener MUST be prepared already.
	if l, _ := s.Options.testListeners[srv.Addr]; l != nil {
		return srv.Serve(l)
	}
	return errors.Reason("test listener is not set").Err()
}

// waitUntilNotServing is called during the graceful shutdown and it tries to
// figure out when the traffic stops flowing to the server (i.e. when it is
// removed from the load balancer).
//
// It's a heuristic optimization for the case when the load balancer keeps
// sending traffic to a terminating Pod for some time after the Pod entered
// "Terminating" state. It can happen due to latencies in Endpoints list
// updates. We want to keep the listening socket open as long as there are
// incoming requests (but no longer than 1 min).
func (s *Server) waitUntilNotServing() {
	logging.Infof(s.Context, "Received SIGTERM, waiting for the traffic to stop...")

	// When the server is idle the loop below exits immediately and the server
	// enters the shutdown path, rejecting new connections. Since we gave
	// Kubernetes no time to update the Endpoints list, it is possible someone
	// still might send a request to the server (and it will be rejected).
	// To avoid that we always sleep a bit here to give Kubernetes a chance to
	// propagate the Endpoints list update everywhere. The loop below then
	// verifies clients got the update and stopped sending requests.
	time.Sleep(s.Options.ShutdownDelay)

	deadline := clock.Now(s.Context).Add(time.Minute)
	for {
		now := clock.Now(s.Context)
		lastReq, ok := s.lastReqTime.Load().(time.Time)
		if !ok || now.Sub(lastReq) > 15*time.Second {
			logging.Infof(s.Context, "No requests received in the last 15 sec, proceeding with the shutdown...")
			break
		}
		if now.After(deadline) {
			logging.Warningf(s.Context, "Gave up waiting for the traffic to stop, proceeding with the shutdown...")
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
}

// RegisterCleanup registers a callback that is run in ListenAndServe after the
// server has exited the serving loop.
//
// Registering a new cleanup callback from within a cleanup causes a deadlock,
// don't do that.
func (s *Server) RegisterCleanup(cb func(context.Context)) {
	s.cleanupM.Lock()
	defer s.cleanupM.Unlock()
	s.cleanup = append(s.cleanup, cb)
}

// runCleanup runs all registered cleanup functions (sequentially in reverse
// order).
func (s *Server) runCleanup() {
	s.cleanupM.Lock()
	defer s.cleanupM.Unlock()
	for i := len(s.cleanup) - 1; i >= 0; i-- {
		s.cleanup[i](s.Context)
	}
}

// genUniqueID returns pseudo-random hex string of given even length.
func (s *Server) genUniqueID(l int) string {
	b := make([]byte, l/2)
	s.rndM.Lock()
	s.rnd.Read(b)
	s.rndM.Unlock()
	return hex.EncodeToString(b)
}

var cloudTraceFormat = propagation.HTTPFormat{}

// rootMiddleware prepares the per-request context.
func (s *Server) rootMiddleware(c *router.Context, next router.Handler) {
	started := clock.Now(s.Context)

	// Wrap the request in a tracing span. The span is closed in the defer below
	// (where we know the response status code). If this is a health check, open
	// the span nonetheless, but do not record it (health checks are spammy and
	// not interesting). This way the code is simpler ('span' is always non-nil
	// and has TraceID). Additionally if some of health check code opens a span
	// of its own, it will be ignored (as a child of not-recorded span).
	healthCheck := isHealthCheckRequest(c.Request)
	ctx, span := s.startRequestSpan(s.Context, c.Request, healthCheck)

	// This is used in waitUntilNotServing.
	if !healthCheck {
		s.lastReqTime.Store(started)
	}

	// Associate all logs with the span via its Trace ID. Use the full ID if we
	// can derive it. This is important to groups logs generated by us with logs
	// generated by the GAE service itself (which uses the full trace ID), when
	// running on GAE. Outside of GAE it doesn't really matter and works either
	// way.
	spanCtx := span.SpanContext()
	traceID := hex.EncodeToString(spanCtx.TraceID[:])
	if s.Options.CloudProject != "" {
		traceID = fmt.Sprintf("projects/%s/traces/%s", s.Options.CloudProject, traceID)
	}

	// Track how many response bytes are sent and what status is set.
	rw := iotools.NewResponseWriter(c.Writer)
	c.Writer = rw

	// Observe maximum emitted severity to use it as an overall severity for the
	// request log entry.
	severityTracker := sdlogger.SeverityTracker{Out: s.stdout}

	// Log the overall request information when the request finishes. Use TraceID
	// to correlate this log entry with entries emitted by the request handler
	// below.
	defer func() {
		now := clock.Now(s.Context)
		latency := now.Sub(started)
		statusCode := rw.Status()

		if healthCheck {
			// Do not log fast health check calls AT ALL, they just spam logs.
			if latency < healthTimeLogThreshold {
				return
			}
			// Emit a warning if the health check is slow, this likely indicates
			// high CPU load.
			logging.Warningf(c.Context, "Health check is slow: %s > %s", latency, healthTimeLogThreshold)
		}

		// When running behind Envoy, log its request IDs to simplify debugging.
		var extraFields logging.Fields
		if xrid := c.Request.Header.Get("X-Request-Id"); xrid != "" {
			extraFields = logging.Fields{"requestId": xrid}
		}

		entry := sdlogger.LogEntry{
			Severity:     severityTracker.MaxSeverity(),
			Timestamp:    sdlogger.ToTimestamp(now),
			TraceID:      traceID,
			TraceSampled: span.IsRecordingEvents(),
			SpanID:       spanCtx.SpanID.String(), // the top-level span ID
			Fields:       extraFields,
			RequestInfo: &sdlogger.RequestInfo{
				Method:       c.Request.Method,
				URL:          getRequestURL(c.Request),
				Status:       statusCode,
				RequestSize:  fmt.Sprintf("%d", c.Request.ContentLength),
				ResponseSize: fmt.Sprintf("%d", rw.ResponseSize()),
				UserAgent:    c.Request.UserAgent(),
				RemoteIP:     getRemoteIP(c.Request),
				Latency:      fmt.Sprintf("%fs", latency.Seconds()),
			},
		}
		if s.Options.Prod {
			// Skip writing the root request log entry on GAE, since GAE writes it
			// itself (in "appengine.googleapis.com/request_log" log). See also
			// comments for initLogging(...).
			if !s.Options.GAE {
				s.stderr.Write(&entry)
			}
		} else {
			logging.Infof(s.Context, "%d %s %q (%s)",
				entry.RequestInfo.Status,
				entry.RequestInfo.Method,
				entry.RequestInfo.URL,
				entry.RequestInfo.Latency,
			)
		}
		span.AddAttributes(
			octrace.Int64Attribute("/http/status_code", int64(statusCode)),
			octrace.Int64Attribute("/http/request/size", c.Request.ContentLength),
			octrace.Int64Attribute("/http/response/size", rw.ResponseSize()),
		)
		span.End()
	}()

	ctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()

	// Make the request logger emit log entries associated with the tracing span.
	if s.Options.Prod {
		annotateWithSpan := func(ctx context.Context, e *sdlogger.LogEntry) {
			// Note: here 'span' is some inner span from where logging.Log(...) was
			// called. We annotate log lines with spans that emitted them.
			if span := octrace.FromContext(ctx); span != nil {
				e.SpanID = span.SpanContext().SpanID.String()
			}
		}
		ctx = logging.SetFactory(ctx, sdlogger.Factory(&severityTracker, sdlogger.LogEntry{
			TraceID:   traceID,
			Operation: &sdlogger.Operation{ID: s.genUniqueID(32)},
		}, annotateWithSpan))
	}

	c.Context = caching.WithRequestCache(ctx)
	next(c)
}

// initLogging initializes the server logging.
//
// Called very early during server startup process. Many server fields may not
// be initialized yet, be careful.
//
// When running in production uses the ugly looking JSON format that is hard to
// read by humans but which is parsed by google-fluentd and GAE hosting
// environment.
//
// To support per-request log grouping in Stackdriver Logs UI there must be
// two different log streams:
//   * A stream with top-level HTTP request entries (conceptually like Apache's
//     access.log, i.e. with one log entry per request).
//   * A stream with logs produced within requests (correlated with HTTP request
//     logs via the trace ID field).
//
// Both streams are expected to have a particular format and use particular
// fields for Stackdriver UI to display them correctly. This technique is
// primarily intended for GAE Flex, but it works in many Google environments:
// https://cloud.google.com/appengine/articles/logging#linking_app_logs_and_requests
//
// On GKE we use 'stderr' stream for top-level HTTP request entries and 'stdout'
// stream for logs produced by requests.
//
// On GAE, the stream with top-level HTTP request entries is produced by the GAE
// runtime itself (as 'appengine.googleapis.com/request_log'). So we emit only
// logs produced within requests (also to 'stdout', just like on GKE).
//
// In all environments 'stderr' stream is used to log all global activities that
// happens outside of any request handler (stuff like initialization, shutdown,
// background goroutines, etc).
//
// In non-production mode we use the human-friendly format and a single 'stderr'
// log stream for everything.
func (s *Server) initLogging() {
	if !s.Options.Prod {
		s.Context = gologger.StdConfig.Use(s.Context)
		s.Context = logging.SetLevel(s.Context, logging.Debug)
		return
	}

	if s.Options.testStdout != nil {
		s.stdout = s.Options.testStdout
	} else {
		s.stdout = &sdlogger.Sink{Out: os.Stdout}
	}

	if s.Options.testStderr != nil {
		s.stderr = s.Options.testStderr
	} else {
		s.stderr = &sdlogger.Sink{Out: os.Stderr}
	}

	s.Context = logging.SetFactory(s.Context,
		sdlogger.Factory(s.stderr, sdlogger.LogEntry{
			Operation: &sdlogger.Operation{
				ID: s.genUniqueID(32), // correlate all global server logs together
			},
		}, nil),
	)
	s.Context = logging.SetLevel(s.Context, logging.Debug)
}

// initAuth initializes auth system by preparing the context, pre-warming caches
// and verifying auth tokens can actually be minted (i.e. supplied credentials
// are valid).
func (s *Server) initAuth() error {
	// Make a transport that appends information about the server as User-Agent.
	ua := s.Options.userAgent()
	rootTransport := clientauth.NewModifyingTransport(http.DefaultTransport, func(req *http.Request) error {
		newUA := ua
		if cur := req.UserAgent(); cur != "" {
			newUA += " " + cur
		}
		req.Header.Set("User-Agent", newUA)
		return nil
	})

	// Initialize the state in the context.
	s.Context = auth.Initialize(s.Context, &auth.Config{
		DBProvider: func(context.Context) (authdb.DB, error) {
			db, _ := s.authDB.Load().(authdb.DB) // refreshed asynchronously in refreshAuthDB
			return db, nil
		},
		Signer:              signerImpl{srv: s},
		AccessTokenProvider: s.getAccessToken,
		AnonymousTransport:  func(context.Context) http.RoundTripper { return rootTransport },
		FrontendClientID:    nil, // TODO(vadimsh): Implement.
		EndUserIP:           getRemoteIP,
		IsDevMode:           !s.Options.Prod,
	})

	// The default value for ClientAuth.SecretsDir is usually hardcoded to point
	// to where the token cache is located on developer machines (~/.config/...).
	// This location often doesn't exist when running from inside a container.
	// The token cache is also not really needed for production services that use
	// service accounts (they don't need cached refresh tokens). So in production
	// mode totally ignore default ClientAuth.SecretsDir and use whatever was
	// passed as -token-cache-dir. If it is empty (default), then no on-disk token
	// cache is used at all.
	//
	// If -token-cache-dir was explicitly set, always use it (even in dev mode).
	// This is useful when running containers locally: developer's credentials
	// on the host machine can be mounted inside the container.
	if s.Options.Prod || s.Options.TokenCacheDir != "" {
		s.Options.ClientAuth.SecretsDir = s.Options.TokenCacheDir
	}

	// Note: we initialize a token source for one arbitrary set of scopes here. In
	// many practical cases this is sufficient to verify that credentials are
	// valid. For example, when we use service account JSON key, if we can
	// generate a token with *some* scope (meaning Cloud accepted our signature),
	// we can generate tokens with *any* scope, since there's no restrictions on
	// what scopes are accessible to a service account, as long as the private key
	// is valid (which we just verified by generating some token).
	au, err := s.initTokenSource(auth.CloudOAuthScopes)
	if err != nil {
		return errors.Annotate(err, "failed to initialize the token source").Err()
	}
	if _, err := au.source.Token(); err != nil {
		return errors.Annotate(err, "failed to mint an initial token").Err()
	}

	// Report who we are running as. Useful when debugging access issues.
	switch email, err := au.authen.GetEmail(); {
	case err == nil:
		logging.Infof(s.Context, "Running as %s", email)
		s.runningAs = email
	case err == clientauth.ErrNoEmail:
		logging.Warningf(s.Context, "Running as <unknown>, cautiously proceeding...")
	case err != nil:
		return errors.Annotate(err, "failed to check the service account email").Err()
	}

	// Now initialize the AuthDB (a database with groups and auth config) and
	// start a goroutine to periodically refresh it.
	if err := s.initAuthDB(); err != nil {
		return errors.Annotate(err, "failed to initialize AuthDB").Err()
	}

	return nil
}

// getAccessToken generates OAuth access tokens to use for requests made by
// the server itself.
//
// It should implement caching internally. This function may be called very
// often, concurrently, from multiple goroutines.
func (s *Server) getAccessToken(c context.Context, scopes []string) (_ *oauth2.Token, err error) {
	key := strings.Join(scopes, " ")

	_, span := trace.StartSpan(c, "github.com/tetrafolium/luci-go/server.GetAccessToken")
	span.Attribute("cr.dev/scopes", key)
	defer func() { span.End(err) }()

	s.authM.RLock()
	au, ok := s.authPerScope[key]
	s.authM.RUnlock()

	if !ok {
		if au, err = s.initTokenSource(scopes); err != nil {
			return nil, err
		}
	}

	return au.source.Token()
}

// initTokenSource initializes a token source for a given list of scopes.
//
// If such token source was already initialized, just returns it and its
// parent authenticator.
func (s *Server) initTokenSource(scopes []string) (scopedAuth, error) {
	key := strings.Join(scopes, " ")

	s.authM.Lock()
	defer s.authM.Unlock()

	au, ok := s.authPerScope[key]
	if ok {
		return au, nil
	}

	// Use ClientAuth as a base template for options, so users of Server{...} have
	// ability to customize various aspects of token generation.
	opts := s.Options.ClientAuth
	opts.Scopes = scopes

	// GAE v2 is very aggressive in caching the token internally (in the metadata
	// server) and refreshing it only when it is very close to its expiration. We
	// need to match this behavior in our in-process cache, otherwise
	// GetAccessToken complains that the token refresh procedure doesn't actually
	// change the token (because the metadata server returned the cached one).
	opts.MinTokenLifetime = 20 * time.Second

	// Note: we are using the root context here (not request-scoped context passed
	// to getAccessToken) because the authenticator *outlives* the request (by
	// being cached), thus it needs a long-living context.
	ctx := logging.SetField(s.Context, "activity", "luci.auth")
	au.authen = clientauth.NewAuthenticator(ctx, clientauth.SilentLogin, opts)
	var err error
	au.source, err = au.authen.TokenSource()

	if err != nil {
		// ErrLoginRequired may happen only when running the server locally using
		// developer's credentials. Let them know how the problem can be fixed.
		if !s.Options.Prod && err == clientauth.ErrLoginRequired {
			if opts.ActAsServiceAccount != "" {
				// Per clientauth.Options doc, IAM is the scope required to use
				// ActAsServiceAccount feature.
				scopes = []string{clientauth.OAuthScopeIAM}
			}
			logging.Errorf(s.Context, "Looks like you run the server locally and it doesn't have credentials for some OAuth scopes")
			logging.Errorf(s.Context, "Run the following command to set them up: ")
			logging.Errorf(s.Context, "  $ luci-auth login -scopes %q", strings.Join(scopes, " "))
		}
		return scopedAuth{}, err
	}

	s.authPerScope[key] = au
	return au, nil
}

// initAuthDB interprets -auth-db-* flags and sets up fetching of AuthDB.
func (s *Server) initAuthDB() error {
	// Check flags are compatible.
	switch {
	case s.Options.AuthDBPath != "" && s.Options.AuthServiceHost != "":
		return errors.Reason("-auth-db-path and -auth-service-host can't be used together").Err()
	case s.Options.AuthServiceHost == "" && (s.Options.AuthDBDump != "" || s.Options.AuthDBSigner != ""):
		return errors.Reason("-auth-db-dump and -auth-db-signer can be used only with -auth-service-host").Err()
	case s.Options.AuthDBDump != "" && !strings.HasPrefix(s.Options.AuthDBDump, "gs://"):
		return errors.Reason("-auth-db-dump value should start with gs://, got %q", s.Options.AuthDBDump).Err()
	case strings.Contains(s.Options.AuthServiceHost, "/"):
		return errors.Reason("-auth-service-host should be a plain hostname, got %q", s.Options.AuthServiceHost).Err()
	}

	// Fill in defaults.
	if s.Options.AuthServiceHost != "" {
		if s.Options.AuthDBDump == "" {
			s.Options.AuthDBDump = fmt.Sprintf("gs://%s/auth-db", s.Options.AuthServiceHost)
		}
		if s.Options.AuthDBSigner == "" {
			if !strings.HasSuffix(s.Options.AuthServiceHost, ".appspot.com") {
				return errors.Reason("-auth-db-signer is required if -auth-service-host is not *.appspot.com").Err()
			}
			s.Options.AuthDBSigner = fmt.Sprintf("%s@appspot.gserviceaccount.com",
				strings.TrimSuffix(s.Options.AuthServiceHost, ".appspot.com"))
		}
	}

	// Fetch the initial copy of AuthDB. Note that this happens before we start
	// the serving loop, to make sure incoming requests have some AuthDB to use.
	if err := s.refreshAuthDB(s.Context); err != nil {
		return errors.Annotate(err, "failed to load the initial AuthDB version").Err()
	}

	// Periodically refresh it in the background.
	s.RunInBackground("luci.authdb", func(c context.Context) {
		for {
			if r := <-clock.After(c, 30*time.Second); r.Err != nil {
				return // the context is canceled
			}
			if err := s.refreshAuthDB(c); err != nil {
				logging.WithError(err).Errorf(c, "Failed to reload AuthDB, using the cached one")
			}
		}
	})
	return nil
}

// refreshAuthDB reloads AuthDB from the source and stores it in memory.
func (s *Server) refreshAuthDB(c context.Context) error {
	cur, _ := s.authDB.Load().(authdb.DB)
	db, err := s.fetchAuthDB(c, cur)
	if err != nil {
		return err
	}
	s.authDB.Store(db)
	return nil
}

// fetchAuthDB fetches the most recent copy of AuthDB from the external source.
//
// 'cur' is the currently used AuthDB or nil if fetching it for the first time.
// Returns 'cur' as is if it's already fresh.
func (s *Server) fetchAuthDB(c context.Context, cur authdb.DB) (authdb.DB, error) {
	if s.Options.testAuthDB != nil {
		return s.Options.testAuthDB, nil
	}

	// Loading from a local file (useful in integration tests).
	if s.Options.AuthDBPath != "" {
		r, err := os.Open(s.Options.AuthDBPath)
		if err != nil {
			return nil, errors.Annotate(err, "failed to open AuthDB file").Err()
		}
		defer r.Close()
		db, err := authdb.SnapshotDBFromTextProto(r)
		if err != nil {
			return nil, errors.Annotate(err, "failed to load AuthDB file").Err()
		}
		return db, nil
	}

	// Loading from a GCS dump (s.Options.AuthDB* are validated here already).
	if s.Options.AuthDBDump != "" {
		c, cancel := clock.WithTimeout(c, 5*time.Minute)
		defer cancel()
		fetcher := dump.Fetcher{
			StorageDumpPath:    s.Options.AuthDBDump[len("gs://"):],
			AuthServiceURL:     "https://" + s.Options.AuthServiceHost,
			AuthServiceAccount: s.Options.AuthDBSigner,
			OAuthScopes:        auth.CloudOAuthScopes,
		}
		curSnap, _ := cur.(*authdb.SnapshotDB)
		snap, err := fetcher.FetchAuthDB(c, curSnap)
		if err != nil {
			return nil, errors.Annotate(err, "fetching from GCS dump failed").Err()
		}
		return snap, nil
	}

	// In dev mode default to "allow everything".
	if !s.Options.Prod {
		return authdb.DevServerDB{}, nil
	}

	// In prod mode default to "fail on any non-trivial check". Some services may
	// not need to use AuthDB at all and configuring it for them is a hassle. If
	// they try to use it for something vital, they'll see the error.
	return authdb.UnconfiguredDB{
		Error: errors.Reason("a source of AuthDB is not configured, see -auth-* server flags").Err(),
	}, nil
}

// initTSMon initializes time series monitoring state.
func (s *Server) initTSMon() error {
	// We keep tsmon always enabled (flushing to /dev/null if no -ts-mon-* flags
	// are set) so that tsmon's in-process store is populated, and metrics there
	// can be examined via /admin/tsmon. This is useful when developing/debugging
	// tsmon metrics.
	var customMonitor monitor.Monitor
	if s.Options.TsMonAccount == "" || s.Options.TsMonServiceName == "" || s.Options.TsMonJobName == "" {
		logging.Infof(s.Context, "tsmon is in the debug mode: metrics are collected, but flushed to /dev/null (pass -ts-mon-* flags to start uploading metrics)")
		customMonitor = monitor.NewNilMonitor()
	}

	s.tsmon = &tsmon.State{
		CustomMonitor: customMonitor,
		Settings: &tsmon.Settings{
			Enabled:            true,
			ProdXAccount:       s.Options.TsMonAccount,
			FlushIntervalSec:   60,
			ReportRuntimeStats: true,
		},
		Target: func(c context.Context) target.Task {
			// TODO(vadimsh): We pretend to be a GAE app for now to be able to
			// reuse existing dashboards. Each pod pretends to be a separate GAE
			// version. That way we can stop worrying about TaskNumAllocator and just
			// use 0 (since there'll be only one task per "version"). This looks
			// chaotic for deployments with large number of pods.
			return target.Task{
				DataCenter:  "appengine",
				ServiceName: s.Options.TsMonServiceName,
				JobName:     s.Options.TsMonJobName,
				HostName:    s.Options.Hostname,
			}
		},
	}
	if customMonitor != nil {
		tsmon.PortalPage.SetReadOnlySettings(s.tsmon.Settings,
			"Running in the debug mode. Pass all -ts-mon-* command line flags to start uploading metrics.")
	} else {
		tsmon.PortalPage.SetReadOnlySettings(s.tsmon.Settings,
			"Settings are controlled through -ts-mon-* command line flags.")
	}

	// Report our image version as a metric, useful to monitor rollouts.
	tsmoncommon.RegisterCallbackIn(s.Context, func(ctx context.Context) {
		versionMetric.Set(ctx, s.Options.imageVersion())
	})

	// Periodically flush metrics.
	s.RunInBackground("luci.tsmon", s.tsmon.FlushPeriodically)
	return nil
}

// initTracing initialized Stackdriver opencensus.io trace exporter.
func (s *Server) initTracing() error {
	if !s.Options.shouldEnableTracing() {
		return nil
	}

	if !s.Options.GAE {
		// Parse -trace-sampling spec to get a sampler.
		sampling := s.Options.TraceSampling
		if sampling == "" {
			sampling = "0.1qps"
		}
		logging.Infof(s.Context, "Setting up Stackdriver trace exports to %q (%s)", s.Options.CloudProject, sampling)
		var err error
		if s.sampler, err = internal.Sampler(sampling); err != nil {
			return errors.Annotate(err, "bad -trace-sampling").Err()
		}
	} else {
		// On GAE let the GAE make decisions about sampling. If it decides to sample
		// a trace, it will let us know through options of the parent span in
		// X-Cloud-Trace-Context. We will collect only traces from requests that
		// GAE wants to sample itself.
		logging.Infof(s.Context, "Setting up Stackdriver trace exports to %q using GAE sampling strategy", s.Options.CloudProject)
		s.sampler = func(p octrace.SamplingParameters) octrace.SamplingDecision {
			return octrace.SamplingDecision{Sample: p.ParentContext.IsSampled()}
		}
	}

	// Grab the token source to call Stackdriver API.
	ts, err := auth.GetTokenSource(s.Context, auth.AsSelf, auth.WithScopes(auth.CloudOAuthScopes...))
	if err != nil {
		return errors.Annotate(err, "failed to initialize token source").Err()
	}
	opts := []option.ClientOption{option.WithTokenSource(ts)}

	// Register the trace uploader. It is also accidentally metrics uploader, but
	// we shouldn't be using metrics (we have tsmon instead).
	exporter, err := stackdriver.NewExporter(stackdriver.Options{
		ProjectID:               s.Options.CloudProject,
		MonitoringClientOptions: opts, // note: this should be effectively unused
		TraceClientOptions:      opts,
		BundleDelayThreshold:    10 * time.Second,
		BundleCountThreshold:    512,
		DefaultTraceAttributes: map[string]interface{}{
			"cr.dev/image":   s.Options.ContainerImageID,
			"cr.dev/service": s.Options.TsMonServiceName,
			"cr.dev/job":     s.Options.TsMonJobName,
			"cr.dev/host":    s.Options.Hostname,
		},
		OnError: func(err error) {
			logging.Errorf(s.Context, "Stackdriver error: %s", err)
		},
	})
	if err != nil {
		return err
	}
	octrace.RegisterExporter(exporter)

	// No matter what, do not sample "random" top-level spans from background
	// goroutines we don't control. We'll start top spans ourselves in
	// startRequestSpan.
	octrace.ApplyConfig(octrace.Config{DefaultSampler: octrace.NeverSample()})

	// Do the final flush before exiting.
	s.RegisterCleanup(func(context.Context) { exporter.Flush() })
	return nil
}

// initProfiling initialized Stackdriver Profiler.
func (s *Server) initProfiling() error {
	// Skip if not enough configuration is given.
	switch {
	case !s.Options.Prod:
		return nil // silently skip, no need for log spam in dev mode
	case s.Options.CloudProject == "":
		logging.Infof(s.Context, "Stackdriver profiler is disabled: -cloud-project is not set")
		return nil
	case s.Options.ProfilingDisable:
		logging.Infof(s.Context, "Stackdriver profiler is disabled: -profiling-disable is set")
		return nil
	case s.Options.ProfilingServiceID == "" && s.Options.TsMonJobName == "":
		logging.Infof(s.Context, "Stackdriver profiler is disabled: neither -profiling-service-id nor -ts-mon-job-name are set")
		return nil
	}

	// Prefer ProfilingServiceID if given, fall back to TsMonJobName. Replace
	// forbidden '/' symbol.
	serviceID := s.Options.ProfilingServiceID
	if serviceID == "" {
		serviceID = s.Options.TsMonJobName
	}
	serviceID = strings.ReplaceAll(serviceID, "/", "-")

	cfg := profiler.Config{
		ProjectID:      s.Options.CloudProject,
		Service:        serviceID,
		ServiceVersion: s.Options.imageVersion(),
		Instance:       s.Options.Hostname,
		// Note: these two options may potentially have impact on performance, but
		// it is likely small enough not to bother.
		MutexProfiling: true,
		AllocForceGC:   true,
	}

	// Authentication for the profiling agent.
	ts, err := auth.GetTokenSource(s.Context, auth.AsSelf, auth.WithScopes(auth.CloudOAuthScopes...))
	if err != nil {
		return errors.Annotate(err, "failed to initialize token source").Err()
	}

	// Launch the agent that runs in the background and periodically collects and
	// uploads profiles. It fails to launch if Service or ServiceVersion do not
	// pass regexp validation. Make it non-fatal, but still log.
	if err := profiler.Start(cfg, option.WithTokenSource(ts)); err != nil {
		logging.Errorf(s.Context, "Stackdriver profiler is disabled: failed do start - %s", err)
		return nil
	}

	logging.Infof(s.Context, "Setting up Stackdriver profiler (service %q, version %q)", cfg.Service, cfg.ServiceVersion)
	return nil
}

// initMainPort initializes the server on options.HTTPAddr port.
func (s *Server) initMainPort() error {
	s.mainPort = s.AddPort(PortOptions{
		Name:       "main",
		ListenAddr: s.Options.HTTPAddr,
	})
	s.Routes = s.mainPort.Routes

	// Install auth info handlers (under "/auth/api/v1/server/").
	auth.InstallHandlers(s.Routes, router.MiddlewareChain{})

	// Expose public pRPC endpoints (see also ListenAndServe where we put the
	// final interceptors).
	s.PRPC = &prpc.Server{
		Authenticator: &auth.Authenticator{
			Methods: []auth.Method{
				&auth.GoogleOAuth2Method{
					Scopes: []string{clientauth.OAuthScopeEmail},
				},
			},
		},
	}
	discovery.Enable(s.PRPC)
	s.PRPC.InstallHandlers(s.Routes, router.MiddlewareChain{})

	// Install RPCExplorer web app at "/rpcexplorer/".
	rpcexplorer.Install(s.Routes)
	return nil
}

// initAdminPort initializes the server on options.AdminAddr port.
func (s *Server) initAdminPort() error {
	if s.Options.AdminAddr == "-" {
		return nil // the admin port is disabled
	}

	// Admin portal uses XSRF tokens that require a secret key. We generate this
	// key randomly during process startup (i.e. now). It means XSRF tokens in
	// admin HTML pages rendered by a server process are understood only by the
	// exact same process. This is OK for admin pages (they are not behind load
	// balancers and we don't care that a server restart invalidates all tokens).
	secret := make([]byte, 20)
	if _, err := cryptorand.Read(secret); err != nil {
		return err
	}
	store := secrets.NewDerivedStore(secrets.Secret{Current: secret})
	withAdminSecret := router.NewMiddlewareChain(func(c *router.Context, next router.Handler) {
		c.Context = secrets.Set(c.Context, store)
		next(c)
	})

	// Install endpoints accessible through the admin port only.
	adminPort := s.AddPort(PortOptions{
		Name:           "admin",
		ListenAddr:     s.Options.AdminAddr,
		DisableMetrics: true, // do not pollute HTTP metrics with admin-only routes
	})
	routes := adminPort.Routes

	routes.GET("/", router.MiddlewareChain{}, func(c *router.Context) {
		http.Redirect(c.Writer, c.Request, "/admin/portal", http.StatusFound)
	})
	portal.InstallHandlers(routes, withAdminSecret, portal.AssumeTrustedPort)

	// Install pprof endpoints on the admin port. Note that they must not be
	// exposed via the main serving port, since they do no authentication and
	// may leak internal information. Also note that pprof handlers rely on
	// routing structure not supported by our router, so we do a bit of manual
	// routing.
	//
	// See also internal/pprof.go for more profiling goodies exposed through the
	// admin portal.
	routes.GET("/debug/pprof/*path", router.MiddlewareChain{}, func(c *router.Context) {
		switch strings.TrimPrefix(c.Params.ByName("path"), "/") {
		case "cmdline":
			pprof.Cmdline(c.Writer, c.Request)
		case "profile":
			pprof.Profile(c.Writer, c.Request)
		case "symbol":
			pprof.Symbol(c.Writer, c.Request)
		case "trace":
			pprof.Trace(c.Writer, c.Request)
		default:
			pprof.Index(c.Writer, c.Request)
		}
	})
	return nil
}

// startRequestSpan opens a new per-request trace span.
//
// Reuses the existing trace (if specified in the request headers) or starts
// a new one.
func (s *Server) startRequestSpan(ctx context.Context, r *http.Request, skipSampling bool) (context.Context, *octrace.Span) {
	var sampler octrace.Sampler
	if skipSampling {
		sampler = octrace.NeverSample()
	} else {
		sampler = s.sampler
	}

	// Add this span as a child to a span propagated through X-Cloud-Trace-Context
	// header (if any). Start a new root span otherwise.
	var span *octrace.Span
	if parent, hasParent := cloudTraceFormat.SpanContextFromRequest(r); hasParent {
		ctx, span = octrace.StartSpanWithRemoteParent(ctx, "HTTP:"+r.URL.Path, parent,
			octrace.WithSpanKind(octrace.SpanKindServer),
			octrace.WithSampler(sampler),
		)
	} else {
		ctx, span = octrace.StartSpan(ctx, "HTTP:"+r.URL.Path,
			octrace.WithSpanKind(octrace.SpanKindServer),
			octrace.WithSampler(sampler),
		)
	}

	// Request info (these are recognized by Stackdriver natively).
	span.AddAttributes(
		octrace.StringAttribute("/http/host", r.Host),
		octrace.StringAttribute("/http/method", r.Method),
		octrace.StringAttribute("/http/path", r.URL.Path),
	)

	return ctx, span
}

// signerImpl implements signing.Signer on top of *Server.
type signerImpl struct {
	srv *Server
}

// SignBytes signs the blob with some active private key.
func (s signerImpl) SignBytes(ctx context.Context, blob []byte) (keyName string, signature []byte, err error) {
	return "", nil, errors.Reason("server.Server: SignBytes is not implemented").Err()
}

// Certificates returns a bundle with public certificates for all active keys.
func (s signerImpl) Certificates(ctx context.Context) (*signing.PublicCertificates, error) {
	return nil, errors.Reason("server.Server: Certificates is not implemented").Err()
}

// ServiceInfo returns information about the current service.
func (s signerImpl) ServiceInfo(ctx context.Context) (*signing.ServiceInfo, error) {
	return &signing.ServiceInfo{
		AppID:              s.srv.Options.CloudProject,
		AppRuntime:         "go",
		AppRuntimeVersion:  runtime.Version(),
		AppVersion:         s.srv.Options.imageVersion(),
		ServiceAccountName: s.srv.runningAs,
	}, nil
}

// networkAddrsForLog returns a string with IPv4 addresses of local network
// interfaces, if possible.
func networkAddrsForLog() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return fmt.Sprintf("failed to enumerate network interfaces: %s", err)
	}
	var ips []string
	for _, address := range addrs {
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipv4 := ipnet.IP.To4(); ipv4 != nil {
				ips = append(ips, ipv4.String())
			}
		}
	}
	if len(ips) == 0 {
		return "<no IPv4 interfaces>"
	}
	return strings.Join(ips, ", ")
}

// getRemoteIP extracts end-user IP address from X-Forwarded-For header.
func getRemoteIP(r *http.Request) string {
	// X-Forwarded-For header is set by Cloud Load Balancer and GAE frontend and
	// has format:
	//   [<untrusted part>,]<IP that connected to LB>,<unimportant>[,<more>].
	//
	// <untrusted part> may be present if the original request from the Internet
	// comes with X-Forwarded-For header. We can't trust IPs specified there. We
	// assume Cloud Load Balancer and GAE sanitize the format of this field
	// though.
	//
	// <IP that connected to LB> is what we are after.
	//
	// <unimportant> is "global forwarding rule external IP" for GKE or
	// the constant "169.254.1.1" for GAE. We don't care about these.
	//
	// <more> is present only if we proxy the request through more layers of
	// load balancers *while it is already inside GKE cluster*. We assume we don't
	// do that (if we ever do, Options{...} should be extended with a setting that
	// specifies how many layers of load balancers to skip to get to the original
	// IP). On GAE <more> is always empty.
	//
	// See https://cloud.google.com/load-balancing/docs/https for more info.
	forwardedFor := strings.Split(r.Header.Get("X-Forwarded-For"), ",")
	if len(forwardedFor) >= 2 {
		return strings.TrimSpace(forwardedFor[len(forwardedFor)-2])
	}

	// Fallback to the peer IP if X-Forwarded-For is not set. Happens when
	// connecting to the server's port directly from within the cluster.
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return "0.0.0.0"
	}
	return ip
}

// getRequestURL reconstructs original request URL to log it (best effort).
func getRequestURL(r *http.Request) string {
	proto := r.Header.Get("X-Forwarded-Proto")
	if proto != "https" {
		proto = "http"
	}
	host := r.Host
	if r.Host == "" {
		host = "127.0.0.1"
	}
	return fmt.Sprintf("%s://%s%s", proto, host, r.RequestURI)
}

// isHealthCheckRequest is true if the request appears to be coming from
// a known health check probe.
func isHealthCheckRequest(r *http.Request) bool {
	if r.URL.Path == healthEndpoint {
		switch ua := r.UserAgent(); {
		case strings.HasPrefix(ua, "kube-probe/"): // Kubernetes
			return true
		case strings.HasPrefix(ua, "GoogleHC/"): // Cloud Load Balancer
			return true
		}
	}
	return false
}
