/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	ricv1 "github.com/RicHincapie/ns-operator/api/v1"
	namespaceHlp "github.com/RicHincapie/ns-operator/pkg/namespace"
)

// NamespaceConfigReconciler reconciles a NamespaceConfig object
type NamespaceConfigReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=ric.ric.com,resources=namespaceconfigs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=ric.ric.com,resources=namespaceconfigs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=ric.ric.com,resources=namespaceconfigs/finalizers,verbs=get;list;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=namespaces,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the NamespaceConfig object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.15.0/pkg/reconcile

func (r *NamespaceConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	crdInstance := &ricv1.NamespaceConfig{}
	var namespace corev1.Namespace
	const crdFinalizer string = "ric.com/namespaceconfig"

	workingNs := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{},
	}
	err := r.Get(ctx, req.NamespacedName, crdInstance)
	if err != nil {
		if errors.IsNotFound(err) {
			log.Log.Info("CRD " + req.NamespacedName.Name + " deleted.")
			return ctrl.Result{}, nil
		} else {
			log.Log.Info("Error with CRD "+req.NamespacedName.Name+" deletion. Err: ", err)
			return ctrl.Result{}, err
		}
	}
	nsFullName := namespaceHlp.GenerateNamespaceName(crdInstance.Name, crdInstance.Spec.NamespacePrefix)
	labelsInCrd := crdInstance.Spec.Labels
	workingNs.Name = nsFullName
	workingNs.ObjectMeta.Labels = labelsInCrd
	// workingNs.Finalizers = append(workingNs.Finalizers, crdFinalizer)
	// Check if its not being deleted and needs the finalizer field to be set
	if crdInstance.DeletionTimestamp.IsZero() {
		if !namespaceHlp.ContainsString(crdInstance.Finalizers, crdFinalizer) {
			log.Log.Info("Adding finalizer to CRD ", "finalizer", crdFinalizer)
			crdInstance.Finalizers = append(crdInstance.Finalizers, crdFinalizer)
			if err := r.Update(ctx, crdInstance); err != nil {
				log.Log.Error(err, "Could not add finalizer to "+req.NamespacedName.Name)
				return ctrl.Result{}, err
			}
		}
		err = r.Client.Get(ctx, types.NamespacedName{Name: nsFullName}, &namespace)
		if err != nil {
			if client.IgnoreNotFound(err) != nil {
				log.Log.Error(err, "Error getting Namespace")
				return ctrl.Result{}, err
			}
			log.Log.Info("Namespace " + nsFullName + " does not exists. Creating it")
			// Create ns because it does not exists
			if err = r.Create(ctx, workingNs); err != nil {
				log.Log.Error(err, "Namespace could not be created: "+nsFullName)
				return ctrl.Result{}, err
			}
			log.Log.Info("Namespace " + nsFullName + " created and labeled with " + namespaceHlp.MapToStrings(labelsInCrd))
			return ctrl.Result{}, nil
		}
		// Check labels in live ns
		labelsInLiveNs := namespace.Labels
		labelsToUpdate := namespaceHlp.MergeMaps(labelsInCrd, labelsInLiveNs)
		// Enforces labels in ns
		workingNs.Labels = labelsToUpdate
		if err = r.Update(ctx, workingNs); err != nil {
			log.Log.Error(err, "Could not update labels "+
				namespaceHlp.MapToStrings(labelsToUpdate)+
				" for "+nsFullName)
			return ctrl.Result{}, err
		}
	} else {
		// CRD has a deletion timestamp. Clean up logic
		workingNs.Name = nsFullName
		if err := r.Delete(ctx, workingNs); err != nil {
			log.Log.Error(err, "Namespace "+nsFullName+" could not be deleted")
			return ctrl.Result{Requeue: false}, err
		}
		// Remove finalizer from CRD
		crdInstance.Finalizers = namespaceHlp.RemoveString(crdInstance.Finalizers, crdFinalizer)
		if err := r.Update(ctx, crdInstance); err != nil {
			log.Log.Error(err, "Could not remove finalizer from "+crdInstance.Name)
			return ctrl.Result{}, err
		}
	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *NamespaceConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&ricv1.NamespaceConfig{}).
		Complete(r)
}
