package sidecar

import (
	"context"
	"fmt"

	"github.com/cloudnative-pg/machinery/pkg/log"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/utils/ptr"
)

const (
	healthyClusterStatus  = "Cluster in healthy state"
	hibernationAnnotation = "cnpg.io/hibernation"
)

// ClusterHibernator hibernates clusters by setting the cnpg.io/hibernation
// annotation and pausing scheduled backups.
type ClusterHibernator struct {
	client clusterClient
}

// NewClusterHibernator creates a new ClusterHibernator with the given client.
func NewClusterHibernator(client clusterClient) *ClusterHibernator {
	return &ClusterHibernator{client: client}
}

// Hibernate triggers hibernation for the cluster by setting the hibernation
// annotation and pausing scheduled backups.
func (h *ClusterHibernator) Hibernate(ctx context.Context, _, _ string) error {
	if err := h.hibernate(ctx); err != nil {
		return err
	}
	return h.pauseScheduledBackup(ctx)
}

func (h *ClusterHibernator) hibernate(ctx context.Context) error {
	cluster, err := h.client.getCluster(ctx, forceUpdate)
	if err != nil {
		return fmt.Errorf("failed to retrieve cluster: %w", err)
	}

	// Only hibernate healthy clusters
	if cluster.Status.Phase != healthyClusterStatus {
		log.FromContext(ctx).Info("cluster is not healthy, skipping hibernation", "status", cluster.Status.Phase)
		return nil
	}

	// Check if the cluster is already hibernated
	if cluster.Annotations != nil && cluster.Annotations[hibernationAnnotation] == "on" {
		log.FromContext(ctx).Info("cluster is already hibernated")
		return nil
	}

	if cluster.Annotations == nil {
		cluster.Annotations = make(map[string]string)
	}

	// Hibernate the cluster by setting the annotation
	cluster.Annotations[hibernationAnnotation] = "on"
	log.FromContext(ctx).Info("annotating cluster for hibernation", "cluster", cluster.Name)
	if err := h.client.updateCluster(ctx, cluster); err != nil {
		log.FromContext(ctx).Error(err, "failed to annotate cluster for hibernation")
		return err
	}

	return nil
}

func (h *ClusterHibernator) pauseScheduledBackup(ctx context.Context) error {
	scheduledBackup, err := h.client.getClusterScheduledBackup(ctx)
	if err != nil {
		if apierrors.IsNotFound(err) {
			log.FromContext(ctx).Debug("scheduled backup not found, skipping pause")
			return nil
		}
		return fmt.Errorf("failed to get scheduled backup: %w", err)
	}

	log.FromContext(ctx).Info("pausing scheduled backup")
	scheduledBackup.Spec.Suspend = ptr.To(true)
	if err := h.client.updateClusterScheduledBackup(ctx, scheduledBackup); err != nil {
		return fmt.Errorf("failed to update scheduled backup: %w", err)
	}

	return nil
}
