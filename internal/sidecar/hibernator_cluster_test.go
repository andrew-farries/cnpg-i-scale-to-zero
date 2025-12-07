package sidecar

import (
	"context"
	"errors"
	"testing"

	cnpgv1 "github.com/cloudnative-pg/cloudnative-pg/api/v1"
	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/utils/ptr"
)

func TestClusterHibernator_Hibernate(t *testing.T) {
	t.Parallel()

	errTest := errors.New("oh noes")
	notFoundErr := apierrors.NewNotFound(schema.GroupResource{Group: "postgresql.cnpg.io", Resource: "scheduledbackups"}, "test")

	tests := []struct {
		name   string
		client *mockClusterClient

		wantErr error
	}{
		{
			name: "cluster is not healthy, should skip hibernation",
			client: &mockClusterClient{
				getClusterFunc: func(ctx context.Context, forceUpdate bool) (*cnpgv1.Cluster, error) {
					return &cnpgv1.Cluster{
						Status: cnpgv1.ClusterStatus{
							Phase: "NotHealthy",
						},
						ObjectMeta: metav1.ObjectMeta{
							Annotations: map[string]string{},
						},
					}, nil
				},
				updateClusterFunc: func(ctx context.Context, cluster *cnpgv1.Cluster) error {
					return errors.New("updateClusterFn should not be called")
				},
				getClusterScheduledBackupFunc: func(ctx context.Context) (*cnpgv1.ScheduledBackup, error) {
					return nil, notFoundErr
				},
			},
			wantErr: nil,
		},
		{
			name: "cluster is already hibernated, should do nothing",
			client: &mockClusterClient{
				getClusterFunc: func(ctx context.Context, forceUpdate bool) (*cnpgv1.Cluster, error) {
					return &cnpgv1.Cluster{
						Status: cnpgv1.ClusterStatus{
							Phase: healthyClusterStatus,
						},
						ObjectMeta: metav1.ObjectMeta{
							Annotations: map[string]string{
								hibernationAnnotation: "on",
							},
						},
					}, nil
				},
				updateClusterFunc: func(ctx context.Context, cluster *cnpgv1.Cluster) error {
					return errors.New("updateClusterFn should not be called")
				},
				getClusterScheduledBackupFunc: func(ctx context.Context) (*cnpgv1.ScheduledBackup, error) {
					return nil, notFoundErr
				},
			},
			wantErr: nil,
		},
		{
			name: "cluster is healthy, nil annotations, should hibernate and pause scheduled backup",
			client: &mockClusterClient{
				getClusterFunc: func(ctx context.Context, forceUpdate bool) (*cnpgv1.Cluster, error) {
					return &cnpgv1.Cluster{
						Status: cnpgv1.ClusterStatus{
							Phase: healthyClusterStatus,
						},
						ObjectMeta: metav1.ObjectMeta{},
					}, nil
				},
				updateClusterFunc: func(ctx context.Context, cluster *cnpgv1.Cluster) error {
					require.Equal(t, "on", cluster.Annotations[hibernationAnnotation])
					return nil
				},
				getClusterScheduledBackupFunc: func(ctx context.Context) (*cnpgv1.ScheduledBackup, error) {
					return &cnpgv1.ScheduledBackup{
						Spec: cnpgv1.ScheduledBackupSpec{},
					}, nil
				},
				updateClusterScheduledBackupFunc: func(ctx context.Context, sb *cnpgv1.ScheduledBackup) error {
					require.Equal(t, ptr.To(true), sb.Spec.Suspend)
					return nil
				},
			},
			wantErr: nil,
		},
		{
			name: "cluster is healthy, hibernation succeeds, no scheduled backup exists",
			client: &mockClusterClient{
				getClusterFunc: func(ctx context.Context, forceUpdate bool) (*cnpgv1.Cluster, error) {
					return &cnpgv1.Cluster{
						Status: cnpgv1.ClusterStatus{
							Phase: healthyClusterStatus,
						},
						ObjectMeta: metav1.ObjectMeta{
							Annotations: map[string]string{},
						},
					}, nil
				},
				updateClusterFunc: func(ctx context.Context, cluster *cnpgv1.Cluster) error {
					require.Equal(t, "on", cluster.Annotations[hibernationAnnotation])
					return nil
				},
				getClusterScheduledBackupFunc: func(ctx context.Context) (*cnpgv1.ScheduledBackup, error) {
					return nil, notFoundErr
				},
			},
			wantErr: nil,
		},
		{
			name: "cluster is healthy, hibernation succeeds, pause scheduled backup fails",
			client: &mockClusterClient{
				getClusterFunc: func(ctx context.Context, forceUpdate bool) (*cnpgv1.Cluster, error) {
					return &cnpgv1.Cluster{
						Status: cnpgv1.ClusterStatus{
							Phase: healthyClusterStatus,
						},
						ObjectMeta: metav1.ObjectMeta{
							Annotations: map[string]string{},
						},
					}, nil
				},
				updateClusterFunc: func(ctx context.Context, cluster *cnpgv1.Cluster) error {
					return nil
				},
				getClusterScheduledBackupFunc: func(ctx context.Context) (*cnpgv1.ScheduledBackup, error) {
					return &cnpgv1.ScheduledBackup{
						Spec: cnpgv1.ScheduledBackupSpec{},
					}, nil
				},
				updateClusterScheduledBackupFunc: func(ctx context.Context, sb *cnpgv1.ScheduledBackup) error {
					return errTest
				},
			},
			wantErr: errTest,
		},
		{
			name: "cluster is healthy, updateCluster returns error",
			client: &mockClusterClient{
				getClusterFunc: func(ctx context.Context, forceUpdate bool) (*cnpgv1.Cluster, error) {
					return &cnpgv1.Cluster{
						Status: cnpgv1.ClusterStatus{
							Phase: healthyClusterStatus,
						},
						ObjectMeta: metav1.ObjectMeta{
							Annotations: map[string]string{},
						},
					}, nil
				},
				updateClusterFunc: func(ctx context.Context, cluster *cnpgv1.Cluster) error {
					return errTest
				},
			},
			wantErr: errTest,
		},
		{
			name: "getCluster returns error",
			client: &mockClusterClient{
				getClusterFunc: func(ctx context.Context, forceUpdate bool) (*cnpgv1.Cluster, error) {
					return nil, errTest
				},
			},
			wantErr: errTest,
		},
		{
			name: "updateCluster returns errReplicaInstance",
			client: &mockClusterClient{
				getClusterFunc: func(ctx context.Context, forceUpdate bool) (*cnpgv1.Cluster, error) {
					return &cnpgv1.Cluster{
						Status: cnpgv1.ClusterStatus{
							Phase: healthyClusterStatus,
						},
						ObjectMeta: metav1.ObjectMeta{
							Annotations: map[string]string{},
						},
					}, nil
				},
				updateClusterFunc: func(ctx context.Context, cluster *cnpgv1.Cluster) error {
					return errReplicaInstance
				},
			},
			wantErr: errReplicaInstance,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			h := NewClusterHibernator(tc.client)
			err := h.Hibernate(context.Background(), "", "")
			require.ErrorIs(t, err, tc.wantErr)
		})
	}
}
