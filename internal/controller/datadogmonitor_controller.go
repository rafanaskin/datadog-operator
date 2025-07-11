// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package controller

import (
	"context"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	ctrlbuilder "sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	datadoghqv1alpha1 "github.com/DataDog/datadog-operator/api/datadoghq/v1alpha1"
	"github.com/DataDog/datadog-operator/internal/controller/datadogmonitor"
	"github.com/DataDog/datadog-operator/pkg/controller/utils/datadog"
	"github.com/DataDog/datadog-operator/pkg/datadogclient"
)

// DatadogMonitorReconciler reconciles a DatadogMonitor object.
type DatadogMonitorReconciler struct {
	Client                 client.Client
	DDClient               datadogclient.DatadogMonitorClient
	Log                    logr.Logger
	Scheme                 *runtime.Scheme
	Recorder               record.EventRecorder
	operatorMetricsEnabled bool
	internal               *datadogmonitor.Reconciler
}

// +kubebuilder:rbac:groups=datadoghq.com,resources=datadogmonitors,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=datadoghq.com,resources=datadogmonitors/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=datadoghq.com,resources=datadogmonitors/finalizers,verbs=get;list;watch;create;update;patch;delete

// Reconcile loop for DatadogMonitor.
func (r *DatadogMonitorReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return r.internal.Reconcile(ctx, req)
}

// SetupWithManager creates a new DatadogMonitor controller.
func (r *DatadogMonitorReconciler) SetupWithManager(mgr ctrl.Manager, metricForwardersMgr datadog.MetricsForwardersManager) error {
	internal, err := datadogmonitor.NewReconciler(r.Client, r.DDClient, r.Scheme, r.Log, r.Recorder, r.operatorMetricsEnabled, metricForwardersMgr)
	if err != nil {
		return err
	}
	r.internal = internal

	builder := ctrl.NewControllerManagedBy(mgr)

	var builderOptions []ctrlbuilder.ForOption
	if r.operatorMetricsEnabled {
		builderOptions = append(builderOptions, ctrlbuilder.WithPredicates(predicate.Funcs{
			// On `DatadogMonitor` object creation, we register a metrics forwarder for it.
			CreateFunc: func(e event.CreateEvent) bool {
				metricForwardersMgr.Register(e.Object)
				return true
			},
		}))
	}

	if err := builder.For(&datadoghqv1alpha1.DatadogMonitor{}, builderOptions...).WithEventFilter(predicate.GenerationChangedPredicate{}).Complete(r); err != nil {
		return err
	}

	return nil
}
