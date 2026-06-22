// Package controllers holds the FalseFlag operator's reconcilers,
// one per CRD. Every reconciler shares an *operator.APIClient and
// the manager's client.Client; the actual reconciliation logic
// lives in the per-resource files (project_controller.go, etc.).
//
// This file just declares the shared dependencies type so Reconcile
// implementations stay consistent across resources.
package controllers
