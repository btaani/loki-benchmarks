package utils

import (
	"context"
	"time"

	"github.com/observatorium/loki-benchmarks/internal/metrics"

	"github.com/prometheus/common/model"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func WaitForReadyDeployment(c client.Client, o client.Object, retry, timeout time.Duration) error {
	return wait.Poll(retry, timeout, func() (done bool, err error) {
		dpl := &appsv1.Deployment{}
		key := client.ObjectKeyFromObject(o)

		err = c.Get(context.TODO(), key, dpl)
		if err != nil {
			if errors.IsNotFound(err) {
				return false, nil
			}
			return false, err
		}

		for _, condition := range dpl.Status.Conditions {
			if condition.Type == appsv1.DeploymentAvailable {
				return condition.Status == corev1.ConditionTrue, nil
			}
		}

		return false, nil
	})
}

func WaitUntilReceivedBytes(m *metrics.Client, threshold float64, duration, retry, timeout time.Duration) error {
	return wait.Poll(retry, timeout, func() (done bool, err error) {
		sample, err := m.DistributorBytesReceivedTotal(model.Duration(duration))
		if err != nil {
			return false, err
		}

		if sample >= threshold {
			return true, nil
		}
		return false, nil
	})
}
