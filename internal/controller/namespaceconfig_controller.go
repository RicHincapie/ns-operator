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
)

// NamespaceConfigReconciler reconciles a NamespaceConfig object
type NamespaceConfigReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=ric.ric.com,resources=namespaceconfigs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=ric.ric.com,resources=namespaceconfigs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=ric.ric.com,resources=namespaceconfigs/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the NamespaceConfig object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.15.0/pkg/reconcile

// THIS FUNCTION IS CALLED BY K8S API WHENEVER A NamespaceConfig RESOURCE IS CREATED, DELETED OR MODIFIED
// IT IS THE ACTUAL HOOK RECEIVER.
// THE ARGS: REQ IS THE ACTUAL INFORMATION FROM THE K8S API CALL
// THE RETURN IS CALLING BACK THE K8S API WITH INSTRUCTIONS
// THIS IS A POINTER RECEIVER THAT CAN CHANGE THE TYPE RECEIVED. THE OTHER TYPE IS VALUE RECEIVER
//
//	WHICH DOES NOT HAVE THE *
func (r *NamespaceConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	// THIS IS THE ACTUAL CRD SCHEMA TO LOAD THE CRD FOUND IN THE CLUSTER
	instance := &ricv1.NamespaceConfig{}
	err := r.Get(ctx, req.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			log.Log.Info("NamespaceConfig resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
	}
	// TODO(user): your logic here
	//DEFINE THE NAMESPACE OBJECT
	var namespace corev1.Namespace
	// GET THE CRD NAME, WHICH IS THE NAMESPACE NAME WE WANT
	namespaceChecked := instance.Name
	prefix := instance.Spec.NamespacePrefix
	nsFullName := prefix + namespaceChecked
	// LOOKS FOR NS AND PLACE IT IN namespace
	err = r.Client.Get(ctx, types.NamespacedName{Name: nsFullName}, &namespace)
	if err != nil {
		if client.IgnoreNotFound(err) != nil {
			log.Log.Error(err, "Failed to get Namespace")
			return ctrl.Result{}, err
		}
		log.Log.Info("Namespace does not exists", "Namespace ", nsFullName)
		// HERE STARTS THE LOGIC TO CREATE THE NS BECAUSE IT DOES NOT EXISTS
		labelsInCrd := instance.Spec.Labels

		newNamespace := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name:   nsFullName,
				Labels: labelsInCrd,
			},
		}
		if err = r.Create(ctx, newNamespace); err != nil {
			log.Log.Error(err, "Namespace could not be created", nsFullName)
		}
		log.Log.Info("Namespace ", nsFullName, " successfully created.")

		return ctrl.Result{}, err
	}
	// HERE GOES THE LOGIC TO CHECK IF THE LABELS ARE PROPERY SET TO AN EXISTING NS

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *NamespaceConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&ricv1.NamespaceConfig{}).
		Complete(r)
}
