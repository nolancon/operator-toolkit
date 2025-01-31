package v1

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/ondat/operator-toolkit/constant"
	tkctrl "github.com/ondat/operator-toolkit/controller"
	"github.com/ondat/operator-toolkit/controller/stateless-action/v1/action"
	"github.com/ondat/operator-toolkit/telemetry"
)

// Name of the instrumentation.
const instrumentationName = constant.LibraryName + "/controller/stateless-action"

// Reconciler is the StatelessAction reconciler.
type Reconciler struct {
	name   string
	ctrlr  Controller
	client client.Client
	scheme *runtime.Scheme

	actionRetryPeriod time.Duration
	actionTimeout     time.Duration
	inst              *telemetry.Instrumentation
}

// ReconcilerOption is used to configure Reconciler.
type ReconcilerOption func(*Reconciler)

// WithName sets the name of the Reconciler.
func WithName(name string) ReconcilerOption {
	return func(r *Reconciler) {
		r.name = name
	}
}

// WithActionRetryPeriod sets the action retry period when it fails.
func WithActionRetryPeriod(duration time.Duration) ReconcilerOption {
	return func(r *Reconciler) {
		r.actionRetryPeriod = duration
	}
}

func WithActionTimeout(duration time.Duration) ReconcilerOption {
	return func(r *Reconciler) {
		r.actionTimeout = duration
	}
}

// WithScheme sets the runtime Scheme of the Reconciler.
func WithScheme(scheme *runtime.Scheme) ReconcilerOption {
	return func(r *Reconciler) {
		r.scheme = scheme
	}
}

// WithInstrumentation configures the instrumentation  of the Reconciler.
func WithInstrumentation(tp trace.TracerProvider, mp metric.MeterProvider, log logr.Logger) ReconcilerOption {
	return func(r *Reconciler) {
		// Populate the instrumentation with reconciler data.
		if r.name != "" {
			log = log.WithValues("reconciler", r.name)
		}
		r.inst = telemetry.NewInstrumentationWithProviders(instrumentationName, tp, mp, log)
	}
}

func (r *Reconciler) Init(mgr ctrl.Manager, ctrlr Controller, opts ...ReconcilerOption) {
	r.ctrlr = ctrlr

	// Use manager if provided. This is helpful in tests to provide explicit
	// client and scheme without a manager.
	if mgr != nil {
		r.client = mgr.GetClient()
		r.scheme = mgr.GetScheme()
	}

	// Run the options to override the defaults.
	for _, opt := range opts {
		opt(r)
	}

	// If instrumentation is nil, create a new instrumentation with default
	// providers.
	if r.inst == nil {
		WithInstrumentation(nil, nil, ctrl.Log)(r)
	}
}

func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (result ctrl.Result, reterr error) {
	ctx, span, _, log := r.inst.Start(ctx, r.name+": Reconcile")
	defer span.End()

	start := time.Now()
	defer tkctrl.LogReconcileFinish(log, "reconciliation finished", start, &result, &reterr)

	span.SetAttributes(attribute.String("object-key", req.NamespacedName.String()))

	controller := r.ctrlr

	// Get an instance of the target object.
	// NOTE: Since the object can be fetched from any backend, we don't know
	// about the error code to be able to perform a proper not found error
	// check. If it's a k8s apimachinery "not found" error, ignore it. Any
	// other error will result in returning error. In order to ignore not found
	// from other backend, return a nil object.
	obj, err := controller.GetObject(ctx, req.NamespacedName)
	if err != nil {
		reterr = client.IgnoreNotFound(err)
		if reterr != nil {
			span.RecordError(reterr)
		}
		return
	}
	// Return if the object is nil.
	if obj == nil {
		span.AddEvent("empty object")
		return
	}

	// Check if an action is required.
	requireAction, err := controller.RequireAction(ctx, obj)
	if err != nil {
		span.RecordError(err)
		reterr = err
		return
	}

	// If an action is required, run an action manager for the target object.
	if requireAction {
		span.AddEvent("Action required, running action manager")
		if err := r.RunActionManager(ctx, obj); err != nil {
			span.RecordError(err)
			reterr = err
			return
		}
	}

	return
}

// RunActionManager runs the actions in the action manager based on the given
// object.
func (r *Reconciler) RunActionManager(ctx context.Context, o interface{}) error {
	ctx, span, _, log := r.inst.Start(ctx, r.name+": run action manager")
	defer span.End()

	span.AddEvent("Build action manager")
	actmgr, err := r.ctrlr.BuildActionManager(o)
	if err != nil {
		span.RecordError(err)
		return errors.Wrapf(err, "failed to build action manager")
	}

	// Get the objects to run action on.
	objects, err := actmgr.GetObjects(ctx)
	if err != nil {
		span.RecordError(err)
		return errors.Wrapf(err, "failed to get objects from action manager")
	}

	span.AddEvent(fmt.Sprintf("Running actions for %d objects", len(objects)))

	// Run the action in a goroutine.
	for _, obj := range objects {
		go func(o interface{}) {
			if runErr := r.RunAction(actmgr, o); runErr != nil {
				log.Error(runErr, "failed to run action")
			}
		}(obj)
	}

	return nil
}

// RunAction checks if an action needs to be run before running it. It also
// runs a deferred function at the end.
func (r *Reconciler) RunAction(actmgr action.Manager, o interface{}) (retErr error) {
	name, err := actmgr.GetName(o)
	if err != nil {
		retErr = errors.Wrapf(err, "failed to get action manager name")
		return
	}

	// Create a context with timeout to be able to cancel the action if it
	// can't be completed within the given time.
	ctx, cancel := context.WithTimeout(context.Background(), r.actionTimeout)
	defer cancel()

	ctx, span, _, log := r.inst.Start(ctx, r.name+": run action")
	defer span.End()

	// Set action info in the logger.
	log = log.WithValues("action", name)

	span.SetAttributes(
		attribute.String("actionName", name),
		attribute.Int64("timeout", int64(r.actionTimeout)),
		attribute.Int64("retryPeriod", int64(r.actionRetryPeriod)),
	)

	// Defer the action Defer() function.
	defer func() {
		if deferErr := actmgr.Defer(ctx, o); deferErr != nil {
			span.RecordError(deferErr)
			retErr = errors.Wrapf(deferErr, "failed to run deferred action")
			return
		}
	}()

	// First run, handle any failure by continuing execution and retry.
	span.AddEvent("First action run")
	if runErr := actmgr.Run(ctx, o); runErr != nil {
		span.RecordError(runErr)
		log.Info("action run failed, will retry", "error", runErr)
	}

	// Check and run the action periodically if the check fails.
	for {
		select {
		case <-time.After(r.actionRetryPeriod):
			checkResult, checkErr := actmgr.Check(ctx, o)
			if checkErr != nil {
				log.Error(checkErr, "failed to perform action check, retrying")
				continue
			}
			if checkResult {
				span.AddEvent("Check result true, rerun action")
				if runErr := actmgr.Run(ctx, o); runErr != nil {
					log.Error(runErr, "action run retry failed")
				}
			} else {
				// Action successful, end the action.
				log.V(6).Info("action successful", "object", o)
				return
			}
		case <-ctx.Done():
			log.Info("context cancelled, terminating action")
			return
		}
	}
}
