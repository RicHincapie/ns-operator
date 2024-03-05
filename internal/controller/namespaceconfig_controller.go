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
//+kubebuilder:rbac:groups=ric.ric.com,resources=namespaceconfigs/finalizers,verbs=update
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
	crdInstance := &ricv1.NamespaceConfig{}
	//DEFINE THE NAMESPACE OBJECT
	var namespace corev1.Namespace
	const crdFinalizer string = "namespaceConfig.finalizer.ric.com"

	workingNs := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{},
	}
	// LOAD THE CRD INTO INSTANCE
	err := r.Get(ctx, req.NamespacedName, crdInstance)
	// 	COMMENTED OUT BECAUSE WE WILL BE CALLED 2 TIMES: BEFORE THE FINALIZER AND AFTER IT?
	// if err != nil {
	// 	if errors.IsNotFound(err) {
	// 		deletedNsName := crdInstance.Spec.NamespacePrefix + req.NamespacedName.Name
	// 		log.Log.Info("NamespaceConfig CRD " + deletedNsName + " will be deleted." +
	// 			"Proceeding to clean up namespace.")
	// 		// TODO: GRACEFULLY DELETE THE NS AND ITS CONTENT

	// 		workingNs.ObjectMeta.Name = deletedNsName
	// 		if err = r.Client.Delete(ctx, workingNs); err != nil {
	// 			log.Log.Error(err, "Namespace could not be deleted: "+deletedNsName)
	// 			return ctrl.Result{}, err
	// 		}
	// 		return ctrl.Result{}, nil
	// 	}
	// }
	// ------------ SECOND IMPLEMENTATION ---------------
	if err != nil {
		if errors.IsNotFound(err) {
			log.Log.Info("CRD " + req.NamespacedName.Name + " deleted")
			return ctrl.Result{}, nil
		}
		log.Log.Info("Error with CRD "+req.NamespacedName.Name+" deletion. Err: ", err)
		return ctrl.Result{}, err
	}
	nsFullName := namespaceHlp.GenerateNamespaceName(crdInstance.Name, crdInstance.Spec.NamespacePrefix)
	// Check if is not being deleted and needs the finalizer field to be set
	if crdInstance.ObjectMeta.DeletionTimestamp.IsZero() {
		if !namespaceHlp.ContainsString(crdInstance.ObjectMeta.Finalizers, crdFinalizer) {
			crdInstance.ObjectMeta.Finalizers = append(crdInstance.ObjectMeta.Finalizers, crdFinalizer)
			if err := r.Update(ctx, crdInstance); err != nil {
				log.Log.Error(err, "Could not add finalizer to "+req.NamespacedName.Name)
			}
		}
	} else {
		// CRD has a deletion timestamp. Clean up logic
		workingNs.ObjectMeta.Name = nsFullName
		if err := r.Delete(ctx, workingNs); err != nil{
			log.Log.Error(err, "Namespace " + nsFullName + " could not be deleted")
			return ctrl.Result{}, err
		}
		// Remove finalizer from CRD
		crdInstance.ObjectMeta.Finalizers = namespaceHlp.RemoveString(crdInstance.ObjectMeta.Finalizers, crdFinalizer)
		if err := r.Update(ctx, crdInstance); err != nil{
			log.Log.Error(err, "Could not remove finalizer from " + crdInstance.Name)
			return ctrl.Result{}, err
		}
	}

	// ------------ END SECOND IMPLEMENTATION ---------------
	// Verification and ns creation logic
	labelsInCrd := crdInstance.Spec.Labels
	workingNs.ObjectMeta.Name = nsFullName
	workingNs.ObjectMeta.Labels = labelsInCrd
	workingNs.ObjectMeta.Finalizers = append(workingNs.ObjectMeta.Finalizers, crdFinalizer)

	err = r.Client.Get(ctx, types.NamespacedName{Name: nsFullName}, &namespace)
	if err != nil {
		if client.IgnoreNotFound(err) != nil {
			log.Log.Error(err, "Error getting Namespace")
			return ctrl.Result{}, err
		}
		log.Log.Info("Namespace " + nsFullName + " does not exists")

		// Create ns because it does not exists

		if err = r.Create(ctx, workingNs); err != nil {
			log.Log.Error(err, "Namespace could not be created: "+nsFullName)
			return ctrl.Result{}, err
		}
		log.Log.Info("Namespace " + nsFullName + " created and labeled with " + namespaceHlp.MapToStrings(labelsInCrd))
	}
	// HERE GOES THE LOGIC TO CHECK IF THE LABELS ARE PROPERY SET TO AN EXISTING NS
	labelsInLiveNs := namespace.Labels
	labelsToUpdate := namespaceHlp.CompareMaps(labelsInCrd, labelsInLiveNs)
	// Overwrites labels in ns
	workingNs.ObjectMeta.Labels = labelsToUpdate
	if err = r.Update(ctx, workingNs); err != nil{
		log.Log.Error(err, "Could not update labels " +
			namespaceHlp.MapToStrings(labelsToUpdate) +
			" for " + nsFullName)
	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *NamespaceConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&ricv1.NamespaceConfig{}).
		Complete(r)
}
